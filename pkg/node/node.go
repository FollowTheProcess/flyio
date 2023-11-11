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

// Result is the result of a message processing operation.
type Result struct {
	Err     error       // Any error in processing
	Message msg.Message // The read or written message
}

// Node is a maelstrom node.
type Node struct {
	stdin         io.Reader     // Messages in from the maelstrom network
	stdout        io.Writer     // Messages out to the network
	id            string        // The ID of this node
	nodeIDs       []string      // The IDs of the nodes in the network (including this one)
	nextMessageID atomic.Uint64 // Incrementing message ID
	mu            sync.RWMutex  // Protecting concurrent access
}

// New constructs and returns a new Node.
func New(stdin io.Reader, stdout io.Writer) *Node {
	return &Node{
		stdin:  stdin,
		stdout: stdout,
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

// NodeIDs returns the topology of the network, it is safe to call from concurrent goroutines
// as it acquires a read lock on the NodeIDs.
func (n *Node) NodeIDs() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.nodeIDs
}

// incrementMessageID bumps the message ID atomically by 1.
func (n *Node) incrementMessageID() {
	n.nextMessageID.Store(n.nextMessageID.Add(uint64(1)))
}

// Handle handles a stream of messages coming in on the inbound channel, dispatches to the correct
// handler to generate the reply, and then puts the reply on the replies channel.
func (n *Node) Handle(inbound <-chan Result, replies chan<- Result) {
	go func() {
		for result := range inbound {
			if result.Err != nil {
				replies <- Result{Err: fmt.Errorf("Handle: inbound message had an error: %w", result.Err)}
				continue // Next message
			}
			switch typ := result.Message.Type(); typ {
			case "init":
				n.handleInit(result.Message, replies)
			case "echo":
				n.handleEcho(result.Message, replies)
			default:
				replies <- Result{Err: fmt.Errorf("Handle: unhandled message type (%s), message: %+v", typ, result.Message)}
			}
		}
		// No more messages in, close the replies channel
		close(replies)
	}()
}

// handleInit handles an incoming init message and puts it's reply on the replies channel.
func (n *Node) handleInit(message msg.Message, replies chan<- Result) {
	var body msg.Init
	if err := json.Unmarshal(message.Body, &body); err != nil {
		replies <- Result{Err: fmt.Errorf("handleInit: unmarshal init body: %w", err)}
	}
	// Initialise our node from the config
	n.Init(body.NodeID, body.NodeIDs)
	us := n.ID()

	n.incrementMessageID()

	// Send the reply
	initOkBody := msg.Init{
		NodeID:  us,
		NodeIDs: n.NodeIDs(),
		Body: msg.Body{
			Type:      "init_ok",
			MessageID: int(n.nextMessageID.Load()),
			InReplyTo: body.MessageID,
		},
	}

	replyBody, err := json.Marshal(initOkBody)
	if err != nil {
		replies <- Result{Err: fmt.Errorf("handleInit: marshal init_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  us,
		Dest: message.Src,
		Body: replyBody,
	}

	replies <- Result{Message: reply}
}

// handleEcho handles an incoming echo message and put's it's reply on the replies channel.
func (n *Node) handleEcho(message msg.Message, replies chan<- Result) {
	var body msg.Echo
	if err := json.Unmarshal(message.Body, &body); err != nil {
		replies <- Result{Err: fmt.Errorf("handleEcho: unmarshal echo body: %w", err)}
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
		replies <- Result{Err: fmt.Errorf("handleEcho: marshal echo_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  n.ID(),
		Dest: message.Src,
		Body: replyBody,
	}

	replies <- Result{Message: reply}
}

// Read reads input from stdin, parses into maelstrom messages and puts them on a channel
// to be consumed.
func Read(stdin io.Reader) <-chan Result {
	results := make(chan Result)
	go func() {
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			var message msg.Message
			if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
				results <- Result{Err: fmt.Errorf("could not decode JSON from stdin: %w", err)}
			}
			results <- Result{Message: message}
		}
		if err := scanner.Err(); err != nil {
			results <- Result{Err: fmt.Errorf("scanner error: %w", err)}
		}
		close(results)
	}()

	return results
}

// Write pulls replies from a channel, and writes them to stdout serially in the order
// they are received in.
func Write(replies <-chan Result, stdout io.Writer) error {
	encoder := json.NewEncoder(stdout)
	for reply := range replies {
		if reply.Err != nil {
			return fmt.Errorf("Write: reply had an error: %w", reply.Err)
		}
		if err := encoder.Encode(reply.Message); err != nil {
			return fmt.Errorf("Write: could not encode reply to JSON: %w", err)
		}
	}
	return nil
}

// Run runs the node loop, receiving messages and generating replies.
func (n *Node) Run() error {
	// TODO: This
	return nil
}
