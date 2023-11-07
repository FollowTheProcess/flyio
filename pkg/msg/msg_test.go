package msg_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/msg"
	"github.com/FollowTheProcess/test"
)

func TestMessageJSON(t *testing.T) {
	testdata := test.Data(t)
	t.Run("init", func(t *testing.T) {
		file := filepath.Join(testdata, "init.json")
		contents, err := os.ReadFile(file)
		test.Ok(t, err)

		var message msg.Message
		err = json.Unmarshal(contents, &message)
		test.Ok(t, err)

		test.Equal(t, message.Src, "c1")
		test.Equal(t, message.Dest, "n1")
		test.Equal(t, message.Type(), "init")

		var init msg.Init
		err = json.Unmarshal(message.Body, &init)
		test.Ok(t, err)

		want := msg.Init{
			NodeID:  "n3",
			NodeIDs: []string{"n1", "n2", "n3"},
			Body: msg.Body{
				Type:      "init",
				ErrorText: "",
				MessageID: 1,
				InReplyTo: 0,
				ErrorCode: 0,
			},
		}

		test.Diff(t, init, want)
	})
}
