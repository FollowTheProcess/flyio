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
- The way the Go implementation has been done means the same message gets `json.Marshal/Unmarshal'd` several times which isn't ideal for performance
- Fun
- I have a nice idea for how to handle this concurrently:
  - A reader goroutine reading from stdin, parsing maelstrom messages and putting them on a channel to be handled
  - Multiple goroutines pulling messages off the inbound channel, handling them in parallel, and putting replies on a reply channel
  - A writer goroutine pulling replies from the reply channel and writing them to stdout

[fly.io]: https://fly.io
[distributed systems challenges]: https://fly.io/dist-sys
[maelstrom]: https://github.com/jepsen-io/maelstrom
