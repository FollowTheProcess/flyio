package node_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/msg"
	"github.com/FollowTheProcess/flyio/pkg/node"
	"github.com/FollowTheProcess/test"
)

func TestHandleInit(t *testing.T) {
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

func TestHandleEcho(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	n := node.New(stdin, stdout)

	// Manually initialise node rather than provide an init message
	n.Init("n3", []string{"n1", "n2", "n3"})

	echoBody := msg.Echo{
		Echo: "Please echo 27",
		Body: msg.Body{
			Type:      "echo",
			MessageID: 69,
		},
	}

	body, err := json.Marshal(echoBody)
	test.Ok(t, err)

	echoMessage := msg.Message{
		Src:  "c3",
		Dest: "n3",
		Body: body,
	}

	// Call handle on the message
	err = n.Handle(echoMessage)
	test.Ok(t, err, "node.Handle")

	// Stdout should now have an echo_ok in it
	var echoOK msg.Message

	err = json.Unmarshal(stdout.Bytes(), &echoOK)
	test.Ok(t, err, "Unmarshal echo_ok")

	test.Equal(t, echoOK.Src, "n3")
	test.Equal(t, echoOK.Dest, "c3")

	var echoOKBody msg.Body
	err = json.Unmarshal(echoOK.Body, &echoOKBody)
	test.Ok(t, err, "Unmarshal echo_ok body")

	want := msg.Body{
		Type:      "echo_ok",
		MessageID: 1,
		InReplyTo: 69,
	}

	test.Diff(t, echoOKBody, want)
}

func TestNodeRun(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	testdata := test.Data(t)

	// Fire message sequences into stdin
	file := filepath.Join(testdata, "in", "init.jsonl")
	contents, err := os.ReadFile(file)
	test.Ok(t, err, "read input jsonl file")
	_, err = stdin.Write(contents)
	test.Ok(t, err, "couldn't write to stdin buffer")

	// Run our node
	n := node.New(stdin, stdout)
	test.Ok(t, n.Run(), "node.Run() error")

	// Stdout should now contain an init_ok
	wantFile := filepath.Join(testdata, "out", "init_ok.jsonl")
	want, err := os.ReadFile(wantFile)
	test.Ok(t, err, "read expected jsonl")

	test.Diff(t, stdout.String(), string(want))
}
