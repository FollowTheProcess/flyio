// Echo implements the echo maelstrom node.
package main

import (
	"fmt"
	"os"

	"github.com/FollowTheProcess/flyio/pkg/node"
)

func main() {
	node := node.New(os.Stdin, os.Stdout)
	if err := node.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
