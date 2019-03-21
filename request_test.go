// +build !race

package reactor

import (
	"os"
	"testing"
	"time"
)

func TestRequestRenderTimeout(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip()
	}

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
		assertNil(t, resp)
		assertNotNil(t, err)
		if err != nil {
			assertContains(t, err.Error(), "timed out")
		}
	}
}
