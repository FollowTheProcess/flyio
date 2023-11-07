package node_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/msg"
	"github.com/FollowTheProcess/flyio/pkg/node"
	"github.com/FollowTheProcess/test"
)

func TestHandle(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	n := node.New(stdin, stdout)

	initBody := msg.Init{
		NodeID:  "n1",
		NodeIDs: []string{"n1", "n2", "n3"},
		Body: msg.Body{
			Type:      "init",
			MessageID: 1,
		},
	}

	body, err := json.Marshal(initBody)
	test.Ok(t, err)

	initMessage := msg.Message{
		Src:  "c1",
		Dest: "n1",
		Body: body,
	}

	// Call handle on the message
	err = n.Handle(initMessage)
	test.Ok(t, err, "node.Handle")

	// Stdout should now have an init_ok in it
	var initOK msg.Message

	err = json.Unmarshal(stdout.Bytes(), &initOK)
	test.Ok(t, err, "Unmarshal init_ok")

	test.Equal(t, initOK.Src, "n1")
	test.Equal(t, initOK.Dest, "c1")

	var initOKBody msg.Body
	err = json.Unmarshal(initOK.Body, &initOKBody)
	test.Ok(t, err, "Unmarshal init_ok body")

	want := msg.Body{
		Type:      "init_ok",
		MessageID: 1,
		InReplyTo: 1,
	}

	test.Diff(t, initOKBody, want)
}
