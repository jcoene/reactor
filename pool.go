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
func NewPool(code string) (*Pool, error) {
	p := &Pool{}
	if err := p.UpdateCode(code); err != nil {
		return nil, err
	}
	return p, nil
}

// UpdateCode updates the server code for the pool, creating a new Worker
// with the given code and retiring all workers running old code. Any
// requests that are currently in-flight will be allowed to finish.
//
// In the case where the new code cannot be loaded onto a worker, an error
// will be returned and the existing code will remain active.
func (p *Pool) UpdateCode(code string) error {
	w, err := NewWorker(code)
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.code = code
	p.version = checksum(code)
	p.workers = []*Worker{w}
	p.mu.Unlock()

	return nil
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
		if w.version == p.version && !w.closed {
			return w, nil
		}
	}

	return NewWorker(p.code)
}

// Put returns a worker to the pool to be re-used in the future
func (p *Pool) Put(w *Worker) {
	p.mu.Lock()
	p.workers = append(p.workers, w)
	p.mu.Unlock()
}
