// Package node implements a maelstrom node.
package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/FollowTheProcess/flyio/pkg/msg"
)

// Node represents a single node in the maelstrom network.
type Node struct {
	stdin         io.Reader        // Messages coming into this node
	stdout        io.Writer        // Messages being sent from this node
	encoder       *json.Encoder    // Encoder writing to stdout
	in            chan msg.Message // Messages in from stdin
	out           chan msg.Message // Messages out to stdout
	errs          chan error       // Errors
	id            string           // The ID of this node, set from the init message
	nodeIDs       []string         // The IDs of all the other nodes in the network (including this one)
	nextMessageID int              // A sequence number locally unique to this node to insert into messages

	mu sync.Mutex // Synchronise incrementing message id
	wg sync.WaitGroup
}

// New constructs and returns a new Node.
func New(stdin io.Reader, stdout io.Writer) *Node {
	return &Node{
		stdin:   stdin,
		stdout:  stdout,
		encoder: json.NewEncoder(stdout),
		in:      make(chan msg.Message),
		out:     make(chan msg.Message),
	}
}

// Init initialises a new Node with it's config.
func (n *Node) Init(id string, nodeIDs []string) {
	n.id = id
	n.nodeIDs = nodeIDs
}

// Run runs the main event handling loop in the Node.
func (n *Node) Run() error {
	n.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		n.read()
	}(&n.wg)

	for message := range n.in {
		n.wg.Add(1)
		go func(wg *sync.WaitGroup, message msg.Message) {
			defer wg.Done()
			if err := n.Handle(message); err != nil {
				n.errs <- err
			}
		}(&n.wg, message)
	}

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		if err := n.write(); err != nil {
			n.errs <- err
		}
	}(&n.wg)

	n.wg.Wait()

	return nil
}

// read reads newline separated messages from stdin and puts them on the in channel.
func (n *Node) read() {
	n.wg.Add(1)
	defer n.wg.Done()
	scanner := bufio.NewScanner(n.stdin)
	for scanner.Scan() {
		var message msg.Message
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			n.errs <- fmt.Errorf("could not decode message from stdin: %w", err)
		}
		n.in <- message
	}
	if err := scanner.Err(); err != nil {
		n.errs <- fmt.Errorf("scanner error: %v", err)
	}

	// No more input, close the in channel
	close(n.in)
}

// write reads messages from the out channel and writes them to stdout.
func (n *Node) write() error {
	for {
		select {
		case message := <-n.out:
			if err := n.encoder.Encode(message); err != nil {
				return fmt.Errorf("could not encode message to JSON: %v", err)
			}
		case err := <-n.errs:
			return fmt.Errorf("err: %v", err)
		}
	}
}

// Handle handles a single message into the Node, and sends the reply.
func (n *Node) Handle(message msg.Message) error {
	switch typ := message.Type(); typ {
	case "init":
		return n.handleInit(message)
	case "echo":
		return n.handleEcho(message)
	default:
		return fmt.Errorf("unhandled message type: %q", typ)
	}
}

// handleInit handles an init message and replies with an init_ok.
func (n *Node) handleInit(message msg.Message) error {
	// We know the message body is an init already
	var received msg.Init
	if err := json.Unmarshal(message.Body, &received); err != nil {
		return fmt.Errorf("could not unmarshal init body: %w", err)
	}

	// Configure our node from the init message
	n.mu.Lock()
	n.Init(received.NodeID, received.NodeIDs)
	n.nextMessageID++
	n.mu.Unlock()

	// Send the init_ok reply
	replyBody := msg.Body{
		Type:      "init_ok",
		MessageID: n.nextMessageID,
		InReplyTo: received.MessageID,
	}

	body, err := json.Marshal(replyBody)
	if err != nil {
		return fmt.Errorf("could not marshal init_ok reply body: %w", err)
	}

	replyMessage := msg.Message{
		Src:  n.id,
		Dest: message.Src,
		Body: body,
	}

	if err := n.encoder.Encode(replyMessage); err != nil {
		return fmt.Errorf("could not encode init_ok reply to JSON: %w", err)
	}

	return nil
}

// handleEcho handles an echo message and replies with an echo_ok.
func (n *Node) handleEcho(message msg.Message) error {
	var received msg.Echo
	if err := json.Unmarshal(message.Body, &received); err != nil {
		return fmt.Errorf("could not unmarshal echo body: %w", err)
	}

	n.mu.Lock()
	n.nextMessageID++
	n.mu.Unlock()

	// Send the echo_ok reply
	replyBody := msg.Echo{
		Echo: received.Echo,
		Body: msg.Body{
			Type:      "echo_ok",
			MessageID: n.nextMessageID,
			InReplyTo: received.MessageID,
		},
	}

	body, err := json.Marshal(replyBody)
	if err != nil {
		return fmt.Errorf("could not marshal echo_ok reply body: %w", err)
	}

	replyMessage := msg.Message{
		Src:  n.id,
		Dest: message.Src,
		Body: body,
	}

	if err := n.encoder.Encode(replyMessage); err != nil {
		return fmt.Errorf("could not encode echo_ok reply to JSON: %w", err)
	}

	return nil
}
