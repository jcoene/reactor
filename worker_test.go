package reactor

import (
	"testing"
)

func TestWorkerEmptyCode(t *testing.T) {
	w, err := NewWorker("")
	assertNil(t, err)
	assertNotNil(t, w)
	if w != nil {
		assertEquals(t, checksum(""), w.version)
		assertEquals(t, false, w.closed)
	}
}

func TestWorkerCallCloseSafety(t *testing.T) {
	w, err := NewWorker(`function render() { return '{"html": "<div>OK</div>"}'; }`)
	assertNil(t, err)

	for i := 0; i < 300; i++ {
		if i > 100 && i < 200 {
			go w.Close()
		}
		resp, err := w.Render(&Request{})
		if err == nil {
			assertContains(t, resp.HTML, "OK")
		} else {
			assertContains(t, err.Error(), "worker closed")
		}
	}
}

func TestWorkerInvalidCode(t *testing.T) {
	w, err := NewWorker("throw 'hi';")
	assertNil(t, w)
	assertNotNil(t, err)
	if err != nil {
		assertContains(t, err.Error(), "Uncaught exception: hi")
	}
}

func TestWorkerRenderClosed(t *testing.T) {
	w, err := NewWorker("")
	assertNil(t, err)
	assertNotNil(t, w)
	w.Close()

	resp, err := w.Render(&Request{})
	assertNil(t, resp)
	assertNotNil(t, err)
	if err != nil {
		assertContains(t, err.Error(), "closed")
	}
	assertEquals(t, true, w.closed)
	assertNil(t, w.ctx)
}
