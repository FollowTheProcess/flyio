// Package node implements a maelstrom node.
package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/FollowTheProcess/flyio/pkg/msg"
)

// result is the result of a message processing operation.
type result struct {
	err     error       // Any error in processing
	message msg.Message // The read or written message
}

// Node is a maelstrom node.
type Node struct {
	stdin         io.Reader     // Where to read messages in from
	stdout        io.Writer     // Where to write messages out to
	in            chan result   // Decoded messages coming in to be handled
	replies       chan result   // Encoded replies being sent out
	id            string        // The ID of this node
	nodeIDs       []string      // The IDs of the nodes in the network (including this one)
	nextMessageID atomic.Uint64 // Incrementing message ID
	mu            sync.RWMutex  // Protecting concurrent access
}

// New constructs and returns a new Node.
func New(stdin io.Reader, stdout io.Writer) *Node {
	return &Node{
		stdin:   stdin,
		stdout:  stdout,
		in:      make(chan result),
		replies: make(chan result),
	}
}

// Init initialises a new Node with it's config, delivered in the first
// message from the maelstrom network.
func (n *Node) Init(id string, nodeIDs []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.id = id
	n.nodeIDs = nodeIDs
}

// ID returns the id of the current node, it is safe to call from concurrent goroutines
// as it acquires a read lock on the ID.
func (n *Node) ID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.id
}

// incrementMessageID bumps the message ID atomically by 1.
func (n *Node) incrementMessageID() {
	n.nextMessageID.Store(n.nextMessageID.Add(uint64(1)))
}

// handle handles a stream of messages coming in on the inbound channel, dispatches to the correct
// handler to generate the reply, and then puts the reply on the replies channel.
func (n *Node) handle() {
	go func() {
		for res := range n.in {
			if res.err != nil {
				n.replies <- result{err: fmt.Errorf("Handle: inbound message had an error: %w", res.err)}
				continue // Next message
			}
			switch typ := res.message.Type(); typ {
			case "init":
				n.handleInit(res.message)
			case "echo":
				n.handleEcho(res.message)
			default:
				n.replies <- result{err: fmt.Errorf("Handle: unhandled message type (%s), message: %+v", typ, res.message)}
			}
		}
		// No more messages in, close the replies channel
		close(n.replies)
	}()
}

// handleInit handles an incoming init message and puts it's reply on the replies channel.
func (n *Node) handleInit(message msg.Message) {
	var body msg.Init
	if err := json.Unmarshal(message.Body, &body); err != nil {
		n.replies <- result{err: fmt.Errorf("handleInit: unmarshal init body: %w", err)}
	}
	// Initialise our node from the config
	n.Init(body.NodeID, body.NodeIDs)

	n.incrementMessageID()

	// Send the reply
	initOkBody := msg.Init{
		Body: msg.Body{
			Type:      "init_ok",
			MessageID: int(n.nextMessageID.Load()),
			InReplyTo: body.MessageID,
		},
	}

	replyBody, err := json.Marshal(initOkBody)
	if err != nil {
		n.replies <- result{err: fmt.Errorf("handleInit: marshal init_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  n.ID(),
		Dest: message.Src,
		Body: replyBody,
	}

	n.replies <- result{message: reply}
}

// handleEcho handles an incoming echo message and put's it's reply on the replies channel.
func (n *Node) handleEcho(message msg.Message) {
	var body msg.Echo
	if err := json.Unmarshal(message.Body, &body); err != nil {
		n.replies <- result{err: fmt.Errorf("handleEcho: unmarshal echo body: %w", err)}
	}

	n.incrementMessageID()

	// Send the reply
	echoOkBody := msg.Echo{
		Echo: body.Echo,
		Body: msg.Body{
			Type:      "echo_ok",
			MessageID: int(n.nextMessageID.Load()),
			InReplyTo: body.MessageID,
		},
	}

	replyBody, err := json.Marshal(echoOkBody)
	if err != nil {
		n.replies <- result{err: fmt.Errorf("handleEcho: marshal echo_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  n.ID(),
		Dest: message.Src,
		Body: replyBody,
	}

	n.replies <- result{message: reply}
}

// read reads input from stdin, parses into maelstrom messages and puts them on a channel
// to be consumed.
func (n *Node) read() {
	go func() {
		scanner := bufio.NewScanner(n.stdin)
		for scanner.Scan() {
			var message msg.Message
			if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
				n.in <- result{err: fmt.Errorf("could not decode JSON from stdin: %w", err)}
			}
			n.in <- result{message: message}
		}
		if err := scanner.Err(); err != nil {
			n.in <- result{err: fmt.Errorf("scanner error: %w", err)}
		}
		close(n.in)
	}()
}

// write pulls replies from a channel, and writes them to stdout serially in the order
// they are received in.
func (n *Node) write() error {
	encoder := json.NewEncoder(n.stdout)
	for reply := range n.replies {
		if reply.err != nil {
			return fmt.Errorf("Write: reply had an error: %w", reply.err)
		}
		if err := encoder.Encode(reply.message); err != nil {
			return fmt.Errorf("Write: could not encode reply to JSON: %w", err)
		}
	}
	return nil
}

// Run runs the node loop, receiving messages and generating replies.
func (n *Node) Run() error {
	n.read()
	// TODO: More than 1 concurrent handler
	n.handle()
	return n.write()
}
