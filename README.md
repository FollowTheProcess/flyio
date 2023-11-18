# Flyio

[![License](https://img.shields.io/github/license/FollowTheProcess/flyio)](https://github.com/FollowTheProcess/flyio)
[![Go Report Card](https://goreportcard.com/badge/github.com/FollowTheProcess/flyio)](https://goreportcard.com/report/github.com/FollowTheProcess/flyio)
[![GitHub](https://img.shields.io/github/v/release/FollowTheProcess/flyio?logo=github&sort=semver)](https://github.com/FollowTheProcess/flyio)
[![CI](https://github.com/FollowTheProcess/flyio/workflows/CI/badge.svg)](https://github.com/FollowTheProcess/flyio/actions?query=workflow%3ACI)
[![codecov](https://codecov.io/gh/FollowTheProcess/flyio/branch/main/graph/badge.svg)](https://codecov.io/gh/FollowTheProcess/flyio)

**Solving the [fly.io] [distributed systems challenges] in Go**

## Project Description

This is my take on solutions for the challenges detailed at <https://fly.io/dist-sys>

Although I'm doing it in Go, I've chosen not to use the provided [maelstrom] library to implement the node and messages. There are a few reasons for this:

- Using their library to take care of all the plumbing feels like cheating 😉
- Their Go implementation has a couple of drawbacks:
  - Each message goes through JSON serialisation/deserialisation multiple times on it's way through the Node
  - The code is very sequential, the handlers kick off a few goroutines but it's not what I'd call "concurrent first"
  - Some gaps in error handling, JSON serialisation/deserialisation can fail silently for example, as does writing to stdout
  - The `HandlerFunc` pattern doesn't really work for the Node as you need access to it to call the `node.Reply` method, leading to large closures defined inline rather than tidy handler functions
- Fun
- I have a nice idea for how to handle this concurrently:
  - A reader goroutine reading from stdin, parsing maelstrom messages and putting them on a channel to be handled
  - Multiple goroutines pulling messages off the inbound channel, handling them in parallel, and putting replies on a reply channel
  - A writer goroutine pulling replies from the reply channel and writing them to stdout

## Design

As above, the message handling is split into 3 concurrent concerns:

- Input parsing
- Message handling
- Reply writing

```mermaid
   stateDiagram-v2

    [*] --> Parser : Messages into STDIN
    state fork_state <<fork>>
      Parser --> fork_state : Emit parsed messages on a channel
      fork_state --> Handler1 : Receive from channel
      fork_state --> Handler2 : Receive from channel
      fork_state --> Handler3 : Receive from channel
      fork_state --> HandlerN... : Receive from channel

      state join_state <<join>>
      Handler1 --> join_state : Write to replies channel
      Handler2 --> join_state : Write to replies channel
      Handler3 --> join_state : Write to replies channel
      HandlerN... --> join_state : Write to replies channel
      join_state --> [*] : Writer goroutine to STDOUT
```

[fly.io]: https://fly.io
[distributed systems challenges]: https://fly.io/dist-sys
[maelstrom]: https://github.com/jepsen-io/maelstrom
