package reactor

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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
	mu  sync.Mutex
}

type responseError struct {
	resp *Response
	err  error
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
	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}

	ch := make(chan responseError, 1)
	go func() {
		resp, err := w.render(req)
		ch <- responseError{resp: resp, err: err}
	}()

	select {
	case re := <-ch:
		return re.resp, re.err
	case <-time.After(req.Timeout):
		return nil, ErrTimedOut
	}
}

// Close closes the worker, releasing resources and refusing future requests
func (w *Worker) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.closed = true
	if w.ctx != nil {
		w.ctx.Release()
		w.ctx = nil
	}
}

// render obtains a lock on the worker and renders the given request
func (w *Worker) render(req *Request) (*Response, error) {
	t := time.Now()

	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil, ErrClosed
	}
	val, err := w.ctx.Call("render", string(buf))
	if err != nil {
		return nil, err
	}
	buf = []byte(val.String())
	val.Release()

	resp := &Response{}
	if err := json.Unmarshal(buf, resp); err != nil {
		return nil, err
	}
	resp.Timer = time.Since(t)

	return resp, nil
}

// checksum computes the md5 sum of the given code
func checksum(code string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(code)))
}
