package reactor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkerEmptyCode(t *testing.T) {
	assert := assert.New(t)

	w, err := NewWorker("")
	assert.Nil(err)
	assert.NotNil(w)
	if w != nil {
		assert.EqualValues(checksum(""), w.version)
		assert.False(w.closed)
	}
}

func TestWorkerCallCloseSafety(t *testing.T) {
	assert := assert.New(t)

	w, err := NewWorker(`function render() { return '{"html": "<div>OK</div>"}'; }`)
	assert.Nil(err)

	for i := 0; i < 300; i++ {
		if i > 100 && i < 200 {
			go w.Close()
		}
		resp, err := w.Render(&Request{})
		if err == nil {
			assert.Contains(resp.HTML, "OK")
		} else {
			assert.Contains(err.Error(), "worker closed")
		}
	}
}

func TestWorkerInvalidCode(t *testing.T) {
	assert := assert.New(t)

	w, err := NewWorker("throw 'hi';")
	assert.Nil(w)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), "Uncaught exception: hi")
	}
}

func TestWorkerRenderClosed(t *testing.T) {
	assert := assert.New(t)

	w, err := NewWorker("")
	assert.Nil(err)
	assert.NotNil(w)
	w.Close()

	resp, err := w.Render(&Request{})
	assert.Nil(resp)
	assert.NotNil(err)
	if err != nil {
		assert.Contains(err.Error(), "closed")
	}
	assert.True(w.closed)
	assert.Nil(w.ctx)
}
