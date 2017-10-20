// +build !race

package reactor

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequestRenderTimeout(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

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
