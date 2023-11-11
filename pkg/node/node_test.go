package node_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/node"
	"github.com/FollowTheProcess/test"
)

func TestNodeRun(t *testing.T) {
	testdata := test.Data(t)
	tests := []struct {
		name string // Name of the test case
		in   string // Name of the file relative to testdata/in containing messages into the node
		want string // Name of the file relative to testdata/out containing expected outputs
	}{
		{
			name: "init",
			in:   "init.jsonl",
			want: "init_ok.jsonl",
		},
		{
			name: "echo",
			in:   "echo.jsonl",
			want: "echo_ok.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := filepath.Join(testdata, "in", tt.in)
			wantFile := filepath.Join(testdata, "out", tt.want)
			f, err := os.Open(input)
			test.Ok(t, err)
			defer f.Close()

			stdout := &bytes.Buffer{}
			n := node.New(f, stdout)

			err = n.Run()
			test.Ok(t, err, "node.Run() returned an error")

			got, err := os.ReadFile(wantFile)
			test.Ok(t, err)

			test.Diff(t, stdout.String(), string(got))
		})
	}
}
