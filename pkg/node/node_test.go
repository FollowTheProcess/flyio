package node_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/msg"
	"github.com/FollowTheProcess/flyio/pkg/node"
	"github.com/FollowTheProcess/test"
)

const (
	initMessage = `{"src": "c1","dest": "n1","body": {"type": "init","msg_id": 1,"node_id": "n1","node_ids": ["n1","n2","n3"]}}` + "\n"
)

func TestRead(t *testing.T) {
	stdin := &bytes.Buffer{}

	// Fire messages into stdin
	stdin.Write([]byte(initMessage))

	results := node.Read(stdin)

	// Should only be one
	result := <-results

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
	test.Diff(t, <-results, node.Result{})
}
