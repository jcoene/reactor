package reactor

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jcoene/reactor/v8"
)

var (
	ErrClosed   = errors.New("worker closed")
	ErrTimedOut = errors.New("timed out")
)

// Worker is a V8 runtime capable of rendering React components
type Worker struct {
	version string
	closed  bool

	ctx *v8.Context
}

type responseError struct {
	response *Response
	err      error
}

// NewWorker returns a new Worker with the given server script loaded
func NewWorker(code string) (*Worker, error) {
	ctx := v8.NewContext()

	if err := ctx.EvalRelease(code, "server.js"); err != nil {
		ctx.Release()
		return nil, err
	}

	return &Worker{
		version: checksum(code),
		ctx:     ctx,
	}, nil
}

// Render renders a React component using the embedded v8 runtime.
func (w *Worker) Render(req *Request) (*Response, error) {
	t := time.Now()

	if w.closed {
		return nil, ErrClosed
	}

	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}

	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan responseError, 1)

	go func() {
		val, err := w.ctx.Call("render", string(buf))
		defer val.Release()

		if err != nil {
			ch <- responseError{err: err}
			return
		}

		resp := &Response{}
		if err := json.Unmarshal([]byte(val.String()), resp); err != nil {
			ch <- responseError{err: err}
			return
		}

		resp.Timer = time.Since(t)
		ch <- responseError{response: resp}
	}()

	select {
	case re := <-ch:
		return re.response, re.err
	case <-time.After(req.Timeout):
		return nil, ErrTimedOut
	}
}

// Close closes the worker, releasing resources and refusing future requests
func (w *Worker) Close() {
	if w.ctx != nil {
		w.ctx.Release()
		w.ctx = nil
	}
	w.closed = true
}

// checksum computes the md5 sum of the given code
func checksum(code string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(code)))
}
