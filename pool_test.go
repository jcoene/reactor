package reactor

import (
	"fmt"
	"sync"
	"testing"
)

func TestPoolRenderEmptyCode(t *testing.T) {
	p := NewPool("")

	resp, err := p.Render(&Request{})
	assertNil(t, resp)
	assertNotNil(t, err)
	if err != nil {
		assertContains(t, err.Error(), "ReferenceError: render is not defined")
	}
}

func TestPoolRenderInvalidCode(t *testing.T) {
	p := NewPool("throw 'hi';")

	resp, err := p.Render(&Request{})
	assertNil(t, resp)
	assertNotNil(t, err)
	if err != nil {
		assertContains(t, err.Error(), "Uncaught exception: hi")
	}
}

func TestPoolUpdateCode(t *testing.T) {
	code1 := `function render() { return '{"html": "<div>1</div>"}'; }`
	code2 := `function render() { return '{"html": "<div>2</div>"}'; }`

	p := NewPool(code1)
	for i := 0; i < 5; i++ {
		resp, err := p.Render(&Request{})
		assertNil(t, err)
		assertNotNil(t, resp)
		if resp != nil {
			assertContains(t, resp.HTML, "1")
		}
	}

	p.UpdateCode(code2)
	for i := 0; i < 5; i++ {
		resp, err := p.Render(&Request{})
		assertNil(t, err)
		assertNotNil(t, resp)
		if resp != nil {
			assertContains(t, resp.HTML, "2")
		}
	}
}

func TestPoolRenderTorture(t *testing.T) {
	threads := 20
	requests := 5000

	// create a new pool
	pool := NewPool(bundle)

	wg := sync.WaitGroup{}

	// render components successfully
	for i := 0; i < threads; i++ {
		wg.Add(1)

		go func(i int) {
			serial := fmt.Sprintf("N-%d-A", i)
			req := &Request{
				Name: "Widget",
				Props: map[string]interface{}{
					"serial": serial,
					"date":   "2017-10-17",
				},
			}
			for j := 0; j < requests; j++ {
				resp, err := pool.Render(req)
				assertNil(t, err)
				assertNotNil(t, resp)
				if resp != nil {
					assertContains(t, resp.HTML, serial)
					assertContains(t, resp.HTML, "manufactured at")
					assertContains(t, resp.HTML, "2017-10-17")
				}
			}

			wg.Done()
		}(i)
	}

	// render components unsuccessfully
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func(i int) {
			req := &Request{
				Name: "WrongWidget",
			}

			resp, err := pool.Render(req)
			assertNil(t, resp)
			assertNotNil(t, err)
			if err != nil {
				assertContains(t, err.Error(), "Cannot find module './WrongWidget.jsx'")
				assertContains(t, err.Error(), "at server.js")
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
}
