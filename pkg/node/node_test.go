package node_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/msg"
	"github.com/FollowTheProcess/flyio/pkg/node"
	"github.com/FollowTheProcess/test"
)

const (
	initMessage   = `{"src": "c1","dest": "n1","body": {"type": "init","msg_id": 1,"node_id": "n1","node_ids": ["n1","n2","n3"]}}` + "\n"
	echoOkMessage = `{"src": "n1","dest": "c1","body": {"type": "echo_ok","msg_id": 1,"in_reply_to": 1,"echo": "Please echo 35"}}` + "\n"
)

func TestRead(t *testing.T) {
	stdin := &bytes.Buffer{}

	// Fire messages into stdin
	stdin.Write([]byte(initMessage))

	results := node.Read(stdin)

	// Should only be one
	result, ok := <-results

	test.True(t, ok)

	test.Ok(t, result.Err, "Result contained an error")
	test.Equal(t, result.Message.Src, "c1")
	test.Equal(t, result.Message.Dest, "n1")

	var init msg.Init
	test.Ok(t, json.Unmarshal(result.Message.Body, &init))
	test.Equal(t, init.Type, "init")
	test.Equal(t, init.MessageID, 1)
	test.Equal(t, init.NodeID, "n1")
	test.EqualFunc(t, init.NodeIDs, []string{"n1", "n2", "n3"}, slices.Equal)

	// This should now be the zero value as the channel should be closed
	another, ok := <-results
	test.False(t, ok)
	test.Diff(t, another, node.Result{})
}

func TestWrite(t *testing.T) {
	stdout := &bytes.Buffer{}
	replies := make(chan msg.Message)

	message := msg.Message{
		Src:  "n1",
		Dest: "c1",
		Body: []byte(echoOkMessage),
	}

	// Put the reply on the channel
	go func() {
		replies <- message
		close(replies)
	}()

	// Call write
	err := node.Write(replies, stdout)
	test.Ok(t, err, "node.Write() returned an error")

	want := `{"src":"n1","dest":"c1","body":{"src":"n1","dest":"c1","body":{"type":"echo_ok","msg_id":1,"in_reply_to":1,"echo":"Please echo 35"}}}`

	// Stdout should now have the message in it
	test.Equal(t, strings.TrimSpace(stdout.String()), strings.TrimSpace(want))
}
