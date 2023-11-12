// Package node implements a maelstrom node.
package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/FollowTheProcess/flyio/pkg/msg"
	"github.com/google/uuid"
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

// incrementMessageID bumps the message ID atomically by 1.
func (n *Node) incrementMessageID() {
	n.nextMessageID.Store(n.nextMessageID.Add(uint64(1)))
}

// handle handles a stream of messages coming in on the inbound channel, dispatches to the correct
// handler to generate the reply, and then puts the reply on the replies channel.
func (n *Node) handle(inputs <-chan result, replies chan<- result, wg *sync.WaitGroup) {
	defer wg.Done()
	for res := range inputs {
		if res.err != nil {
			replies <- result{err: fmt.Errorf("Handle: inbound message had an error: %w", res.err)}
			continue // Next message
		}
		switch typ := res.message.Type(); typ {
		case "init":
			n.handleInit(res.message, replies)
		case "echo":
			n.handleEcho(res.message, replies)
		case "generate":
			n.handleGenerate(res.message, replies)
		default:
			replies <- result{err: fmt.Errorf("Handle: unhandled message type: %q", typ)}
		}
	}
}

// handleInit handles an incoming init message and puts it's reply on the replies channel.
func (n *Node) handleInit(message msg.Message, replies chan<- result) {
	var body msg.Init
	if err := json.Unmarshal(message.Body, &body); err != nil {
		replies <- result{err: fmt.Errorf("handleInit: unmarshal init body: %w", err)}
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
		replies <- result{err: fmt.Errorf("handleInit: marshal init_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  n.ID(),
		Dest: message.Src,
		Body: replyBody,
	}

	replies <- result{message: reply}
}

// handleEcho handles an incoming echo message and puts it's reply on the replies channel.
func (n *Node) handleEcho(message msg.Message, replies chan<- result) {
	var body msg.Echo
	if err := json.Unmarshal(message.Body, &body); err != nil {
		replies <- result{err: fmt.Errorf("handleEcho: unmarshal echo body: %w", err)}
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
		replies <- result{err: fmt.Errorf("handleEcho: marshal echo_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  n.ID(),
		Dest: message.Src,
		Body: replyBody,
	}

	replies <- result{message: reply}
}

// handleGenerate handles an incoming generate message and puts it's reply on the replies channel.
func (n *Node) handleGenerate(message msg.Message, replies chan<- result) {
	var body msg.Body
	if err := json.Unmarshal(message.Body, &body); err != nil {
		replies <- result{err: fmt.Errorf("handleGenerate: unmarshal generate body: %w", err)}
	}

	n.incrementMessageID()

	generateOkBody := msg.Generate{
		ID: uuid.NewString(),
		Body: msg.Body{
			Type:      "generate_ok",
			MessageID: int(n.nextMessageID.Load()),
			InReplyTo: body.MessageID,
		},
	}

	replyBody, err := json.Marshal(generateOkBody)
	if err != nil {
		replies <- result{err: fmt.Errorf("handleGenerate: marshal generate_ok body: %w", err)}
	}

	reply := msg.Message{
		Src:  n.ID(),
		Dest: message.Src,
		Body: replyBody,
	}

	replies <- result{message: reply}
}

// read reads input from stdin, parses into maelstrom messages and puts them on a channel
// to be consumed.
func (n *Node) read() <-chan result {
	inputs := make(chan result)
	go func() {
		scanner := bufio.NewScanner(n.stdin)
		for scanner.Scan() {
			var message msg.Message
			if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
				inputs <- result{err: fmt.Errorf("could not decode JSON from stdin: %w", err)}
			}
			inputs <- result{message: message}
		}
		if err := scanner.Err(); err != nil {
			inputs <- result{err: fmt.Errorf("scanner error: %w", err)}
		}
		// No more messages in, close the inputs channel
		close(inputs)
	}()
	return inputs
}

// Run runs the node loop, receiving messages and generating replies.
func (n *Node) Run() error {
	var wg sync.WaitGroup
	replies := make(chan result)
	inputs := n.read()

	// Fire off a load of concurrent handlers
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go n.handle(inputs, replies, &wg)
	}

	// Wait for them all to complete, then close the replies channel as we know
	// there will be no more replies if all the handlers have completed. Note, this is in
	// a goroutine so it doesn't block pulling from the replies channel below
	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(replies)
	}(&wg)

	encoder := json.NewEncoder(n.stdout)
	for reply := range replies {
		if reply.err != nil {
			return fmt.Errorf("Write: reply had an error: %w", reply.err)
		}
		if err := encoder.Encode(reply.message); err != nil {
			return fmt.Errorf("Write: could not encode reply to JSON: %w", err)
		}
	}

	return nil
}
