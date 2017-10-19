package v8

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestReleaseValueAfterContext(t *testing.T) {

	ctx := NewContext()
	val, err := ctx.Eval("5 + 5;", "")
	assertNil(t, err)
	assertNotNil(t, val)
	assertEquals(t, 10, val.String())
	ctx.Release()
	val.Release()
}

func TestValueReleaseTwice(t *testing.T) {

	ctx := NewContext()
	val, err := ctx.Eval("5 + 5;", "")
	assertNil(t, err)
	assertNotNil(t, val)
	assertEquals(t, 10, val.String())
	val.Release()
	val.Release()
	ctx.Release()
}

func TestContextReleaseTwice(t *testing.T) {

	ctx := NewContext()
	ctx.Release()
	ctx.Release()
}

func TestContextTorture(t *testing.T) {

	spawn(20, func() {
		iterate(20, func() {
			NewContext().Release()
		})
	})
}

func TestDeferReleases(t *testing.T) {

	ctx := NewContext()
	defer ctx.Release()

	val, err := ctx.Eval("5 + 5;", "")
	defer val.Release()
	assertNil(t, err)
	assertNotNil(t, val)
	assertEquals(t, 10, val.String())
}

func TestReleaseNil(t *testing.T) {

	val := &Value{}
	val.Release()
}

func TestEvalScript(t *testing.T) {

	code := `
		function double(n) {
			return n * 2;
		}

		double(2);
	`

	withContext(func(ctx *Context) {
		val, err := ctx.Eval(code, "server.js")
		if val == nil || val.String() != "4" {
			t.Errorf("unexpected result: %s", val.String())
		}
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	})
}

func TestEvalInvalidScript(t *testing.T) {
	code := `double(2);`

	withContext(func(ctx *Context) {
		val, err := ctx.Eval(code, "server.js")
		if val != nil {
			t.Fatalf("unexpected value: %+v", val)
		}
		if err == nil {
			t.Fatalf("did not encounter expected error")
		}
		if !strings.Contains(err.Error(), "ReferenceError: double is not defined") {
			t.Fatalf("incomplete error: %s", err.Error())
		}
	})
}

func TestSegmentFault(t *testing.T) {
	type person struct {
		name string
	}

	var nobody *person

	spawn(3, func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recovered without segfaulting due to", err)
			}
		}()

		ctx := NewContext()
		ctx.Release()

		fmt.Println(nobody.name) // causes a panic
	})

}

func TestConcurrentCall(t *testing.T) {
	code := `
		function process(json) {
			var req = JSON.parse(json);
			req.n2 = req.n * 2;
			req.s2 = req.s + '2';
			req.ok = true;
			return JSON.stringify(req);
		}
	`

	type req struct {
		N  int    `json:"n"`
		N2 int    `json:"n2"`
		S  string `json:"s"`
		S2 string `json:"s2"`
		Ok bool   `json:"ok"`
	}

	withContext(func(ctx *Context) {
		threads := 5
		iterations := 10000
		val, err := ctx.Eval(code, "bundle.js")
		val.Release()

		if err != nil {
			t.Fatalf("cannot eval code: %s", err)
		}

		wg := sync.WaitGroup{}
		for i := 0; i < threads; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					in := &req{
						N: j,
						S: fmt.Sprintf("'\"搜索\"' %d", j),
					}

					// cmd := fmt.Sprintf(`process('%s');`, encodeJson(in))
					// val, err := ctx.Eval(cmd, "")
					val, err := ctx.Call("process", encodeJson(in))
					s := val.String()
					val.Release()

					if err != nil {
						t.Errorf("unexpected error: %s", err)
						continue
					}

					out := &req{}
					if err := json.Unmarshal([]byte(s), out); err != nil {
						t.Errorf("unable to decode json '%s': %s", s, err)
						continue
					}

					if in.N != out.N {
						t.Errorf("N not returned: %d != %d", in.N, out.N)
					}
					if out.N*2 != out.N2 {
						t.Errorf("N2 not doubled: %d * 2 != %d", out.N, out.N2)
					}
					if in.S != out.S {
						t.Errorf("S not returned: '%s' != '%s'", in.S, out.S)
					}
					if in.S+"2" != out.S2 {
						t.Errorf("S2 not concatenated: '%s' + '2' != '%s'", in.S, out.S2)
					}
					if out.Ok != true {
						t.Errorf("ok not true: %v", out.Ok)
					}
				}
			}()
		}

		wg.Wait()
	})
}

func decodeJson(s string, v interface{}) {
	if err := json.Unmarshal([]byte(s), v); err != nil {
		panic(err)
	}
}
func encodeJson(v interface{}) string {
	buf, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

func TestConcurrentEval(t *testing.T) {
	code := `function double(n) { return n * 2; }`

	threads := 5
	iterations := 10000

	wg := sync.WaitGroup{}
	for i := 0; i < threads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			withContext(func(ctx *Context) {
				val, err := ctx.Eval(code, "bundle2.js")
				val.Release()
				if err != nil {
					t.Errorf("unable to eval code: %s", err)
					return
				}

				for j := 0; j < iterations; j++ {
					x := rand.Intn(10000)
					script := fmt.Sprintf("double(%d)", x)
					expect := fmt.Sprintf("%d", x*2)
					val, err := ctx.Eval(script, "")
					if err != nil {
						t.Errorf("unable to call double: %s", err)
						return
					}
					if s := val.String(); s != expect {
						t.Errorf("unexpected %s result: %s (wanted %s)", script, s, expect)
					}
					val.Release()
				}
			})
		}()
	}

	wg.Wait()
}

func withContext(fn func(ctx *Context)) {
	//iso := NewIsolate()
	// ctx := iso.NewContext()
	ctx := NewContext()
	fn(ctx)
	ctx.Release()
	// iso.Release()
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

func spawn(n int, fn func()) {
	wg := sync.WaitGroup{}
	iterate(n, func() {
		wg.Add(1)
		defer wg.Done()
		fn()
	})
	wg.Wait()
}

func iterate(n int, fn func()) {
	for i := 0; i < n; i++ {
		fn()
	}
}
