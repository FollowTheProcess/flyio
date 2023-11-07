// Package msg provides data structures representing the messages defined in the maelstrom message protocol.
package msg

import (
	"encoding/json"

	"github.com/valyala/fastjson"
)

// Message represents a single maelstrom message.
type Message struct {
	Src  string          `json:"src,omitempty"`  // The node ID of the source of this message
	Dest string          `json:"dest,omitempty"` // The node ID of the destination for this message
	Body json.RawMessage `json:"body,omitempty"` // The raw JSON body, could be any type
}

// Type returns the type of the message.
func (m Message) Type() string {
	return fastjson.GetString(m.Body, "type")
}

// Body represents the common components of a maelstrom message.
type Body struct {
	Type      string `json:"type,omitempty"`        // The message type
	ErrorText string `json:"text,omitempty"`        // Error message, if one occurred
	MessageID int    `json:"msg_id,omitempty"`      // Identifier for the message, unique to the source node
	InReplyTo int    `json:"in_reply_to,omitempty"` // Identifier for the message this message is replying to
	ErrorCode int    `json:"code,omitempty"`        // Error code, if one occurred
}

// Init represents an init message.
type Init struct {
	NodeID  string   `json:"node_id,omitempty"`  // The ID of the node receiving this message
	NodeIDs []string `json:"node_ids,omitempty"` // The IDs of all the nodes in the network (including the recipient)
	Body
}

// Echo represents an echo message.
type Echo struct {
	Echo string `json:"echo,omitempty"` // The message to echo back
	Body
}
