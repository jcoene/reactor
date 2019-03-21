package reactor

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

var bundle = func() string {
	buf, err := ioutil.ReadFile("example/bundle.js")
	if err != nil {
		panic(err)
	}
	return string(buf)
}()

func BenchmarkRender(b *testing.B) {
	// create a new pool
	pool := NewPool(bundle)

	req := &Request{
		Name: "Widget",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := pool.Render(req); err != nil {
			b.Fatal(err)
		}
	}
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

func assertEquals(t *testing.T, e, g interface{}) {
	es := fmt.Sprintf("%v", e)
	gs := fmt.Sprintf("%v", e)
	if es != gs {
		t.Errorf("expected '%s', got '%s'", e, gs)
	}
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	return reflect.ValueOf(v).IsNil()
}
