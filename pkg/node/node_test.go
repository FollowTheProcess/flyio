package node_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/FollowTheProcess/flyio/pkg/node"
	"github.com/FollowTheProcess/test"
	"github.com/kinbiko/jsonassert"
)

func TestNodeRun(t *testing.T) {
	testdata := test.Data(t)
	tests := []struct {
		name string // Name of the test case
		in   string // Name of the file relative to testdata/in containing messages into the node
		want string // Name of the file relative to testdata/out containing expected outputs
		seen []int  // Message IDs we should fool the node into thinking it's seen
		init bool   // Whether to fake initialise the node, so we don't have to send an init message every time
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
			init: true,
		},
		{
			name: "generate",
			in:   "generate.jsonl",
			want: "generate_ok.jsonl",
			init: true,
		},
		{
			name: "broadcast",
			in:   "broadcast.jsonl",
			want: "broadcast_ok.jsonl",
			init: true,
		},
		{
			name: "read",
			in:   "read.jsonl",
			want: "read_ok.jsonl",
			init: true,
			seen: []int{1, 2, 3, 4},
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

			if tt.init {
				n.Init("n1", []string{"n1", "n2", "n3"})
			}

			if len(tt.seen) != 0 {
				n.SetSeen(tt.seen...)
			}

			err = n.Run()
			test.Ok(t, err, "node.Run() returned an error")

			want, err := os.ReadFile(wantFile)
			test.Ok(t, err)

			// Normalise line endings... stupid windows
			want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
			got := bytes.ReplaceAll(stdout.Bytes(), []byte("\r\n"), []byte("\n"))

			// Each line should be valid JSON
			wantLines := bytes.Split(want, []byte("\n"))
			gotLines := bytes.Split(got, []byte("\n"))

			// One JSON line, and a newline at the end
			test.Equal(t, len(wantLines), 2)
			test.Equal(t, len(gotLines), 2)

			ja := jsonassert.New(t)

			ja.Assertf(string(gotLines[0]), string(wantLines[0]))
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
