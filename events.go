package main

import (
	"bytes"
	"github.com/divan/gotrace/trace"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

// EventSource defines anything that can
// return tracing events.
type EventSource interface {
	Events() ([]*trace.Event, error)
}

// TraceSource implements EventSource for
// pre-made trace file
type TraceSource struct {
	// Trace is the path to the trace file.
	Trace string

	// Binary is a path to binary, needed to symbolize stacks.
	// In the future it may be dropped, see
	// https://groups.google.com/d/topic/golang-dev/PGX1H8IbhFU
	Binary string
}

// NewTraceSource inits new TraceSource.
func NewTraceSource(trace, binary string) *TraceSource {
	return &TraceSource{
		Trace:  trace,
		Binary: binary,
	}
}

// Events reads trace file from filesystem and symbolizes
// stacks.
func (t *TraceSource) Events() ([]*trace.Event, error) {
	f, err := os.Open(t.Trace)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseTrace(f, t.Binary)
}

func parseTrace(r io.Reader, binary string) ([]*trace.Event, error) {
	events, err := trace.Parse(r)
	if err != nil {
		return nil, err
	}

	err = trace.Symbolize(events, binary)

	return events, err
}

// NativeRun implements EventSource for running app locally,
// using native Go installation.
type NativeRun struct {
	Path string
}

func NewNativeRun(path string) *NativeRun {
	return &NativeRun{
		Path: path,
	}
}

// Events rewrites package if needed, adding Go execution tracer
// to the main function, then builds and runs package using default Go
// installation and returns parsed events.
func (r *NativeRun) Events() ([]*trace.Event, error) {
	// rewrite AST

	tmpBinary, err := ioutil.TempFile("", "gotracer_build")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBinary.Name())

	// build binary
	// TODO: replace build&run part with "go run" when there is no more need
	// to keep binary
	err = exec.Command("go", "build", "-o", tmpBinary.Name(), r.Path).Run()
	if err != nil {
		// TODO: test on most common errors, possibly add stderr to
		// error information or smth.
		return nil, err
	}

	// run
	var stderr bytes.Buffer
	cmd := exec.Command(tmpBinary.Name())
	cmd.Stderr = &stderr
	if err = cmd.Run(); err != nil {
		// TODO: test on most common errors, possibly add stderr to
		// error information or smth.
		return nil, err
	}

	// parse trace
	return parseTrace(&stderr, tmpBinary.Name())
}

type DockerRun struct{}
