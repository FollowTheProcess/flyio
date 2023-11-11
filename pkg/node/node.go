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

// Result is the result of a message processing operation.
type Result struct {
	Err     error       // Any error in processing
	Message msg.Message // The read or written message
}

// Node is a maelstrom node.
type Node struct {
	stdin   io.Reader // Messages in from the maelstrom network
	stdout  io.Writer // Messages out to the network
	id      string    // The ID of this node
	nodeIDs []string  // The IDs of the nodes in the network (including this one)

	mu sync.RWMutex // Protecting concurrent access
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

// Write pulls replies from a channel, and writes them to stdout.
func Write(replies <-chan msg.Message, stdout io.Writer) error {
	encoder := json.NewEncoder(stdout)
	for reply := range replies {
		if err := encoder.Encode(reply); err != nil {
			return fmt.Errorf("could not encode reply to JSON: %w", err)
		}
	}
	return nil
}

// Run runs the node loop, receiving messages and generating replies.
func (n *Node) Run() error {
	// TODO: This
	return nil
}
