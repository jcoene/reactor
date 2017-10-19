package v8

// #include <stdlib.h>
// #include <string.h>
// #include "v8_c.h"
// #cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include -std=c++11
// #cgo darwin LDFLAGS: -L${SRCDIR}/lib/darwin_x86_64 -lv8_base -lv8_libbase -lv8_snapshot -lv8_libsampler -lv8_libplatform -ldl -pthread
// #cgo linux LDFLAGS: -L${SRCDIR}/lib/linux_x86_64 -lv8_base -lv8_libbase -lv8_snapshot -lv8_libsampler -lv8_libplatform -ldl -pthread
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

var once sync.Once

var ErrReleasedContext = errors.New("released context")

func initV8() {
	C.V8_Init()
}

// Context is a v8::Context wrapped in it's own v8::Isolate. It must be
// manually released to avoid leaking references.
type Context struct {
	ptr C.ContextPtr
	mu  sync.Mutex
}

// Value is a v8::Persistent<v8::Value> associated with a v8::Context. It
// must be manually released to avoid leaking references. It is safe to
// release a nil *Value.
type Value struct {
	ptr C.ValuePtr
	ctx *Context
}

// NewContext creates a new Context. It should be released after use.
func NewContext() *Context {
	once.Do(func() {
		initV8()
	})

	ctx := &Context{
		ptr: C.V8_Context_New(),
	}

	runtime.SetFinalizer(ctx, func(ctx *Context) {
		ctx.Release()
	})

	return ctx
}

// Release releases the Context, including it's internal v8::Context and
// v8::Isolate. Any Values with outstanding references will become unusable
// and may cause a segmentation fault if tried to access.
func (ctx *Context) Release() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.ptr != nil {
		C.V8_Context_Release(ctx.ptr)
		ctx.ptr = nil
	}
}

// Call calls the given function with the provided arguments. The arguments
// will be JSON encoded and passed to Eval.
func (ctx *Context) Call(name string, vs ...interface{}) (*Value, error) {
	args := make([]string, len(vs))
	for i, v := range vs {
		buf, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("can't encode argument %d (%v): %s", i, v, err)
		}
		args[i] = string(buf)
	}
	code := fmt.Sprintf("%s(%s)", name, strings.Join(args, ","))
	val, err := ctx.Eval(code, "")
	return val, err
}

// Eval evaluates the given code inside of the Context. Either the
// returned Value or error will be present, never both. If returned, the
// given Value must be manually released to avoid leaking references.
func (ctx *Context) Eval(code, filename string) (*Value, error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.ptr == nil {
		return nil, fmt.Errorf("invalid context")
	}

	c_code := C.CString(code)
	c_filename := C.CString(filename)
	result := C.V8_Context_Eval(ctx.ptr, c_code, c_filename)
	C.free(unsafe.Pointer(c_filename))
	C.free(unsafe.Pointer(c_code))
	return ctx.decodeResult(result)
}

// EvalRelease calls Eval, returning only the error (if present). If Eval
// returns a Value, it will be released.
func (ctx *Context) EvalRelease(code, filename string) error {
	val, err := ctx.Eval(code, filename)
	val.Release()
	return err
}

// decodeResult turns a Result, which contains a Value or error, into
// the appropriate go *Value and error types. If a Value is returned,
// it must be manually released to avoid leaking references.
func (ctx *Context) decodeResult(res C.Result) (v *Value, err error) {
	if res.v_ptr != nil {
		v = &Value{
			ctx: ctx,
			ptr: res.v_ptr,
		}
	}
	if res.e.ptr != nil {
		s := C.GoStringN(res.e.ptr, res.e.len)
		C.free(unsafe.Pointer(res.e.ptr))

		sc := string([]byte(s))
		err = errors.New(sc)
	}
	return
}

// String returns the string value of the given Value. If the Value or it's
// internal pointer are nil, "undefined" will be returned. It is safe to call
// String on a nil *Value.
func (val *Value) String() string {
	if val == nil || val.ptr == nil || val.ctx == nil || val.ctx.ptr == nil {
		return "undefined"
	}

	val.ctx.mu.Lock()
	defer val.ctx.mu.Unlock()
	if val.ctx.ptr == nil {
		return "undefined"
	}

	c_s := C.V8_Value_String(val.ctx.ptr, val.ptr)
	s := C.GoStringN(c_s.ptr, c_s.len)
	C.free(unsafe.Pointer(c_s.ptr))
	sc := string([]byte(s))
	return sc
}

// Release releases the underlying v8::Persistent<v8::Value>. This must be done
// on any non-nil values returned from any Context method in order to avoid
// leaking references. Calling Release on a Value whose Context has been
// Released may cause a segmentation fault.
func (val *Value) Release() {
	if val == nil || val.ctx == nil || val.ptr == nil {
		return
	}

	val.ctx.mu.Lock()
	defer val.ctx.mu.Unlock()

	// do our best to prevent a segfault, but a race can theoretically happen.
	if val.ctx.ptr == nil {
		fmt.Println("WARNING: You attempted to release a v8.Value which is associated with a")
		fmt.Println("         v8.Context that has already been released. This is your bug and")
		fmt.Println("         may result in a segmentation fault if you're unlucky.")
		return
	}

	C.V8_Value_Release(val.ctx.ptr, val.ptr)
	val.ctx = nil
	val.ptr = nil
}
