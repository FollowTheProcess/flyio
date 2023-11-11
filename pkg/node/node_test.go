package node_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

			want, err := os.ReadFile(wantFile)
			test.Ok(t, err)

			// Normalise line endings... stupid windows
			want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
			got := bytes.ReplaceAll(stdout.Bytes(), []byte("\r\n"), []byte("\n"))

			// The files will be in order, but since the node handles messages concurrently
			// there is no guarantee that the order is preserved between message in and reply out
			// nor need there be as each reply has it's in_reply_to ID. However for the test, we need
			// deterministic output so we build slices of line separated JSON and sort them, then
			// compare the sorted slices
			wantLines := strings.Split(string(want), "\n")
			gotLines := strings.Split(string(got), "\n")

			slices.Sort(wantLines)
			slices.Sort(gotLines)

			test.Diff(t, gotLines, wantLines)
		})
	}
}

func BenchmarkNodeRun(b *testing.B) {
	content, err := os.ReadFile(filepath.Join(test.Data(b), "bench", "bench.jsonl"))
	test.Ok(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := node.New(bytes.NewReader(content), &bytes.Buffer{})
		if err := n.Run(); err != nil {
			b.Fatalf("node.Run() returned an error: %v", err)
		}
	}
}
