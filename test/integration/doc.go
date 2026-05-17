// Package integration contains end-to-end integration tests for the media
// pipeline.
//
// The tests drive the public HTTP API through the full
// validate -> plan -> execute pipeline, mirroring how cmd/api wires the
// server. The pipeline tests shell out to ffmpeg and skip themselves when it
// is not installed.
//
// These tests are guarded by the "integration" build tag and are excluded
// from the default `go test ./...` run. Run them with:
//
//	go test -tags=integration ./test/integration/...
//
// or via the Makefile target:
//
//	make test-integration
package integration
