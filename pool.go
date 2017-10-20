package reactor

import (
	"sync"
)

// Pool provides a dynamically growing pool of workers capable of rendering.
type Pool struct {
	code    string
	version string

	workers []*Worker
	mu      sync.Mutex
}

// NewPool creates a new Pool of workers with the given server code. It
// creates a single Worker with the given code. Additional workers will
// be created on-demand as needed.
func NewPool(code string) *Pool {
	return &Pool{
		code:    code,
		version: checksum(code),
	}
}

// UpdateCode updates the server code for the pool, causing any existing
// workers running an older version of the code to be closed in the future.
// Any requests that are currently in-flight will be allowed to finish.
func (p *Pool) UpdateCode(code string) {
	p.mu.Lock()
	p.code = code
	p.version = checksum(code)
	p.mu.Unlock()
}

// Render renders a React component with a worker from the pool. If a worker
// with the current code version is not available, a new worker will be created.
func (p *Pool) Render(req *Request) (*Response, error) {
	w, err := p.Get()
	if err != nil {
		return nil, err
	}

	resp, err := w.Render(req)
	if err != nil {
		w.Close()
		return nil, err
	}

	p.Put(w)

	return resp, nil
}

// Get returns the next worker from the pool, creating a new worker if needed.
// Workers with previous code versions will be discarded, resulting in a new
// worker being created.
func (p *Pool) Get() (*Worker, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.workers) > 0 {
		w := p.workers[0]
		p.workers = p.workers[1:]
		if w.closed || w.version != p.version {
			w.Close()
			continue
		}
		return w, nil
	}

	return NewWorker(p.code)
}

// Put returns a worker to the pool to be re-used in the future
func (p *Pool) Put(w *Worker) {
	p.mu.Lock()
	p.workers = append(p.workers, w)
	p.mu.Unlock()
}
