// Echo implements the echo maelstrom node.
package main

import (
	"fmt"
	"os"

	"github.com/FollowTheProcess/flyio/pkg/node"
)

func main() {
	// TODO: I want to separate this into 3 concurrent parts:
	// 1) A reader from stdin who's job is to decode messages and put them on a channel
	// 2) Handlers in the middle to pull from the channel and generate replies (concurrently if possible)
	// 3) A writer to stdout who's job is to read from the reply channel and encode it as JSON to stdout
	node := node.New(os.Stdin, os.Stdout)
	if err := node.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
