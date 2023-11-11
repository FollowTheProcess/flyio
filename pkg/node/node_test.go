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
	replies := make(chan node.Result)

	message := msg.Message{
		Src:  "n1",
		Dest: "c1",
		Body: []byte(echoOkMessage),
	}

	// Put the reply on the channel
	go func() {
		replies <- node.Result{Message: message}
		close(replies)
	}()

	// Call write
	err := node.Write(replies, stdout)
	test.Ok(t, err, "node.Write() returned an error")

	want := `{"src":"n1","dest":"c1","body":{"src":"n1","dest":"c1","body":{"type":"echo_ok","msg_id":1,"in_reply_to":1,"echo":"Please echo 35"}}}`

	// Stdout should now have the message in it
	test.Equal(t, strings.TrimSpace(stdout.String()), strings.TrimSpace(want))
}

func TestHandleInit(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	n := node.New(stdin, stdout)

	inbound := make(chan node.Result)
	replies := make(chan node.Result)

	// Put an init message on the inbound channel
	go func() {
		init := msg.Init{
			NodeID:  "n1",
			NodeIDs: []string{"n1", "n2", "n3"},
			Body: msg.Body{
				Type:      "init",
				MessageID: 1,
			},
		}
		initBody, err := json.Marshal(init)
		test.Ok(t, err)
		inbound <- node.Result{
			Message: msg.Message{
				Src:  "c1",
				Dest: "n1",
				Body: initBody,
			},
		}
		close(inbound)
	}()

	n.Handle(inbound, replies)

	// Read what should be an init_ok off the replies channel
	reply, ok := <-replies
	test.True(t, ok)
	test.Ok(t, reply.Err)

	test.Equal(t, reply.Message.Src, "n1")
	test.Equal(t, reply.Message.Dest, "c1")

	var replyBody msg.Body
	err := json.Unmarshal(reply.Message.Body, &replyBody)
	test.Ok(t, err)

	want := msg.Body{
		Type:      "init_ok",
		MessageID: 1,
		InReplyTo: 1,
	}

	test.Diff(t, replyBody, want)

	// Replies should now be closed
	_, ok = <-replies
	test.False(t, ok)
}

func TestHandleEcho(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	n := node.New(stdin, stdout)

	// Fake being initialised
	n.Init("n1", []string{"n1", "n2", "n3"})

	inbound := make(chan node.Result)
	replies := make(chan node.Result)

	// Put an echo message on the inbound channel
	go func() {
		echo := msg.Echo{
			Echo: "Please echo 27",
			Body: msg.Body{
				Type:      "echo",
				MessageID: 1,
			},
		}
		echoBody, err := json.Marshal(echo)
		test.Ok(t, err)
		inbound <- node.Result{
			Message: msg.Message{
				Src:  "c1",
				Dest: "n1",
				Body: echoBody,
			},
		}
		close(inbound)
	}()

	n.Handle(inbound, replies)

	// Read what should be an echo_ok off the replies channel
	reply, ok := <-replies
	test.True(t, ok)
	test.Ok(t, reply.Err)

	test.Equal(t, reply.Message.Src, "n1")
	test.Equal(t, reply.Message.Dest, "c1")

	var replyBody msg.Echo
	err := json.Unmarshal(reply.Message.Body, &replyBody)
	test.Ok(t, err)

	want := msg.Echo{
		Echo: "Please echo 27",
		Body: msg.Body{
			Type:      "echo_ok",
			MessageID: 1,
			InReplyTo: 1,
		},
	}

	test.Diff(t, replyBody, want)

	// Replies should now be closed
	_, ok = <-replies
	test.False(t, ok)
}
