#include "v8_c_bridge.h"

#include "libplatform/libplatform.h"
#include "v8.h"

#include <cstdlib>
#include <cstring>
#include <string>
#include <sstream>
#include <stdio.h>

#define ISOLATE_SCOPE(isolate_ptr) \
  v8::Isolate* isolate = (isolate_ptr); \
  v8::Locker locker(isolate); \
  v8::Isolate::Scope isolate_scope(isolate);

#define CONTEXT_SCOPE(context_ptr) \
  Context* context = static_cast<Context*>(context_ptr); \
  ISOLATE_SCOPE(context->isolate);

#define VALUE_SCOPE(context_ptr) \
  CONTEXT_SCOPE(context_ptr); \
  v8::HandleScope handle_scope(isolate); \
  v8::Local<v8::Context> local_context(context->ptr.Get(isolate)); \
  v8::Context::Scope context_scope(local_context);

typedef struct {
  v8::Persistent<v8::Context> ptr;
  v8::Isolate* isolate;
} Context;

typedef v8::Persistent<v8::Value> V8_Persistent_Value;

String DupString(const v8::String::Utf8Value& src) {
  char* data = static_cast<char*>(malloc(src.length()));
  memcpy(data, *src, src.length());
  return (String){data, src.length()};
}
String DupString(const v8::Local<v8::Value>& val) {
  return DupString(v8::String::Utf8Value(val));
}
String DupString(const char* msg) {
  const char* data = strdup(msg);
  return (String){data, int(strlen(msg))};
}
String DupString(const std::string& src) {
  char* data = static_cast<char*>(malloc(src.length()));
  memcpy(data, src.data(), src.length());
  return (String){data, int(src.length())};
}

std::string str(v8::Local<v8::Value> value) {
  v8::String::Utf8Value s(value);
  if (s.length() == 0) {
    return "";
  }
  return *s;
}

std::string report_exception(v8::Isolate* isolate, v8::TryCatch& try_catch) {
  std::stringstream ss;
  ss << "Uncaught exception: ";

  std::string exceptionStr = str(try_catch.Exception());
  ss << exceptionStr; // TODO(aroman) JSON-ify objects?

  if (!try_catch.Message().IsEmpty()) {
    if (!try_catch.Message()->GetScriptResourceName()->IsUndefined()) {
      ss << std::endl
         << "at " << str(try_catch.Message()->GetScriptResourceName()) << ":"
         << try_catch.Message()->GetLineNumber() << ":"
         << try_catch.Message()->GetStartColumn() << std::endl
         << "  " << str(try_catch.Message()->GetSourceLine()) << std::endl
         << "  ";
      int start = try_catch.Message()->GetStartColumn();
      int end = try_catch.Message()->GetEndColumn();
      for (int i = 0; i < start; i++) {
        ss << " ";
      }
      for (int i = start; i < end; i++) {
        ss << "^";
      }
    }
  }

  if (!try_catch.StackTrace().IsEmpty()) {
    ss << std::endl << "Stack trace: " << str(try_catch.StackTrace());
  }

  return ss.str();
}

// Called from Go

extern "C" {

Version version = {V8_MAJOR_VERSION, V8_MINOR_VERSION, V8_BUILD_NUMBER, V8_PATCH_LEVEL};

void V8_Init() {
  v8::Platform *platform = v8::platform::CreateDefaultPlatform();
  v8::V8::InitializePlatform(platform);
  v8::V8::Initialize();
  return;
}

ContextPtr V8_Context_New() {
  // Create a v8::Isolate
  v8::Isolate::CreateParams create_params;
  create_params.array_buffer_allocator = v8::ArrayBuffer::Allocator::NewDefaultAllocator();
  v8::Isolate* isolate = v8::Isolate::New(create_params);
  v8::Locker locker(isolate);
  v8::Isolate::Scope isolate_scope(isolate);
  v8::HandleScope handle_scope(isolate);

  // v8::V8::SetCaptureStackTraceForUncaughtExceptions(true);

  v8::Local<v8::ObjectTemplate> globals = v8::ObjectTemplate::New(isolate);

  Context* context = new Context;
  context->ptr.Reset(isolate, v8::Context::New(isolate, nullptr, globals));
  context->isolate = isolate;
  return static_cast<ContextPtr>(context);
}

// releaseIsolate retrieves the V8_Context and sets ISOLATE_SCOPE, then
// resets the v8::Context and returns the v8::Isolate it used to contain.
v8::Isolate* releaseIsolate(ContextPtr context_ptr) {
  CONTEXT_SCOPE(context_ptr);
  context->ptr.Reset();
  return isolate;
}

// V8_Context_Release releases the Context, first by resetting the internal
// v8::Context then disposing of the v8::Isolate.
void V8_Context_Release(ContextPtr context_ptr) {
  // Release the isolate from the context
  v8::Isolate* isolate = releaseIsolate(context_ptr);
  // Dispose of the isolate
  isolate->Dispose();
}

// V8_Context_Eval compiles and run the given code inside of the context.
Result V8_Context_Eval(ContextPtr context_ptr, const char* code, const char* filename) {
  VALUE_SCOPE(context_ptr);

  v8::TryCatch try_catch;
  try_catch.SetVerbose(false);

  Result res = { nullptr, nullptr };

  v8::Local<v8::Script> script = v8::Script::Compile(
      v8::String::NewFromUtf8(isolate, code),
      v8::String::NewFromUtf8(isolate, filename));

  if (script.IsEmpty()) {
    res.e = DupString(report_exception(isolate, try_catch));
    return res;
  }

  v8::Local<v8::Value> result = script->Run();

  if (result.IsEmpty()) {
    res.e = DupString(report_exception(isolate, try_catch));
  } else {
    V8_Persistent_Value* val = new V8_Persistent_Value(isolate, result);
    res.v_ptr = static_cast<ValuePtr>(val);
  }

  return res;
}

String V8_Value_String(ContextPtr context_ptr, ValuePtr value_ptr) {
  VALUE_SCOPE(context_ptr);
  v8::Local<v8::Value> value = static_cast<V8_Persistent_Value*>(value_ptr)->Get(isolate);
  return DupString(value->ToString());
}

void V8_Value_Release(ContextPtr context_ptr, ValuePtr value_ptr) {
  VALUE_SCOPE(context_ptr);
  V8_Persistent_Value* value = static_cast<V8_Persistent_Value*>(value_ptr);
  value->Reset();
  delete value;
}

} // extern "C"
