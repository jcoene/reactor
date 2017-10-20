package reactor

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoolRenderEmptyCode(t *testing.T) {
	assert := assert.New(t)

	p := NewPool("")

	resp, err := p.Render(&Request{})
	assert.Nil(resp)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), "ReferenceError: render is not defined")
	}
}

func TestPoolRenderInvalidCode(t *testing.T) {
	assert := assert.New(t)

	p := NewPool("throw 'hi';")

	resp, err := p.Render(&Request{})
	assert.Nil(resp)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), "Uncaught exception: hi")
	}
}

func TestPoolRenderTimeout(t *testing.T) {
	assert := assert.New(t)

	pool := NewPool(bundle)

	for i := 0; i < 10; i++ {
		req := &Request{
			Name: "Widget",
			Props: map[string]interface{}{
				"serial": "1",
			},
			Timeout: 10 * time.Nanosecond,
		}

		resp, err := pool.Render(req)
		assert.Nil(resp)
		assert.NotNil(err)
		if err != nil {
			assert.Contains(err.Error(), "timed out")
		}
	}
}

func TestPoolUpdateCode(t *testing.T) {
	assert := assert.New(t)

	code1 := `function render() { return '{"html": "<div>1</div>"}'; }`
	code2 := `function render() { return '{"html": "<div>2</div>"}'; }`

	p := NewPool(code1)
	for i := 0; i < 5; i++ {
		resp, err := p.Render(&Request{})
		assert.Nil(err)
		assert.NotNil(resp)
		if resp != nil {
			assert.Contains(resp.HTML, "1")
		}
	}

	p.UpdateCode(code2)
	for i := 0; i < 5; i++ {
		resp, err := p.Render(&Request{})
		assert.Nil(err)
		assert.NotNil(resp)
		if resp != nil {
			assert.Contains(resp.HTML, "2")
		}
	}
}

func TestPoolRenderTorture(t *testing.T) {
	assert := assert.New(t)

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
				assert.Nil(err)
				assert.NotNil(resp)
				if resp != nil {
					assert.Contains(resp.HTML, serial)
					assert.Contains(resp.HTML, "manufactured at")
					assert.Contains(resp.HTML, "2017-10-17")
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
			assert.Nil(resp)
			assert.NotNil(err)
			if err != nil {
				assert.Contains(err.Error(), "Cannot find module './WrongWidget.jsx'")
				assert.Contains(err.Error(), "at server.js")
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
}
