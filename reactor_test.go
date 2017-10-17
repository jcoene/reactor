package reactor

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"
	"testing"
)

var bundle = func() string {
	buf, err := ioutil.ReadFile("example/bundle.js")
	if err != nil {
		panic(err)
	}
	return string(buf)
}()

func TestRender(t *testing.T) {
	// create a new pool
	pool, err := NewPool(bundle)
	if err != nil {
		t.Fatalf("cannot create pool: %s", err)
	}

	wg := sync.WaitGroup{}

	// render components successfully
	for i := 0; i < 100; i++ {
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

			resp, err := pool.Render(req)
			assertNil(t, err)
			assertContains(t, resp.HTML, serial)
			assertContains(t, resp.HTML, "manufactured at")
			assertContains(t, resp.HTML, "2017-10-17")

			wg.Done()
		}(i)
	}

	// render components unsuccessfully
	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			req := &Request{
				Name: "WrongWidget",
			}

			resp, err := pool.Render(req)
			assertNil(t, resp)
			assertNotNil(t, err)
			assertContains(t, err.Error(), "Cannot find module './WrongWidget.jsx'")
			assertContains(t, err.Error(), "at server.js")

			wg.Done()
		}(i)
	}

	wg.Wait()
}

func assertNil(t *testing.T, v interface{}) {
	if !isNil(v) {
		t.Errorf("expected nil, got '%+v'", v)
	}
}

func assertNotNil(t *testing.T, v interface{}) {
	if isNil(v) {
		t.Errorf("expected value to be non-nil")
	}
}

func assertContains(t *testing.T, s, substr string) {
	if !strings.Contains(s, substr) {
		t.Errorf("expected '%s' to contain '%s'", s, substr)
	}
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	return reflect.ValueOf(v).IsNil()
}
