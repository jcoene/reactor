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
	ErrClosed = fmt.Errorf("worker closed")
)

// Worker is a V8 runtime capable of rendering React components
type Worker struct {
	version string
	closed  bool

	iso *v8.Isolate
	ctx *v8.Context
	fn  *v8.Value
}

type responseError struct {
	response *Response
	err      error
}

// NewWorker returns a new Worker with the given server script loaded
func NewWorker(code string) (*Worker, error) {
	iso := v8.NewIsolate()
	ctx := iso.NewContext()

	if _, err := ctx.Eval(code, "server.js"); err != nil {
		return nil, err
	}

	fn, err := ctx.Global().Get("render")
	if err != nil {
		return nil, err
	}

	return &Worker{
		version: checksum(code),
		iso:     iso,
		ctx:     ctx,
		fn:      fn,
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

	args, err := w.ctx.Create(req)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %s", err)
	}

	ch := make(chan responseError, 1)

	cb := w.ctx.Bind("rendered", func(in v8.CallbackArgs) (*v8.Value, error) {
		buf := []byte(in.Arg(0).String())

		resp := &Response{}
		if err := json.Unmarshal(buf, resp); err != nil {
			ch <- responseError{response: nil, err: err}
			return nil, nil
		}
		if resp.Error != "" {
			ch <- responseError{response: nil, err: errors.New(resp.Error)}
			return nil, nil
		}
		resp.Timer = time.Since(t)

		ch <- responseError{
			response: resp,
			err:      nil,
		}
		return nil, nil
	})

	if _, err := w.fn.Call(w.fn, args, cb); err != nil {
		return nil, fmt.Errorf("render call failed: %s", err)
	}

	select {
	case re := <-ch:
		return re.response, re.err
	case <-time.After(req.Timeout):
		return nil, fmt.Errorf("timed out after %v", req.Timeout)
	}
}

// Close closes the worker, releasing resources and refusing future requests
func (w *Worker) Close() {
	w.fn = nil
	w.ctx = nil
	w.iso = nil
	w.closed = true
}

// checksum computes the md5 sum of the given code
func checksum(code string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(code)))
}
