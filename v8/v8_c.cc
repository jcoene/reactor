#include "v8_c.h"

#include "libplatform/libplatform.h"
#include "v8.h"

#include <cstdlib>
#include <cstring>
#include <string>
#include <sstream>
#include <stdio.h>

typedef struct {
  v8::Persistent<v8::Context> ptr;
  v8::Isolate* isolate;
} Context;

typedef v8::Persistent<v8::Value> V8_Persistent_Value;

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

void DEBUG(const char *msg) {
  if(std::getenv("DEBUG")) {
    printf("CC %s\n", msg);
  }
}

std::string report_exception(v8::Isolate* isolate, v8::TryCatch& try_catch) {
  DEBUG("report_exception enter");

  std::stringstream ss;
  ss << "Uncaught exception: ";

  std::string exceptionStr = str(try_catch.Exception());
  ss << exceptionStr; // TODO(aroman) JSON-ify objects?

  if (!try_catch.Message().IsEmpty()) {
    DEBUG("report_exception try_catch not empty");
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
  } else {
    DEBUG("report_exception try_catch empty");
  }

  if (!try_catch.StackTrace().IsEmpty()) {
    DEBUG("report_exception stack_trace not empty");
    ss << std::endl << "Stack trace: " << str(try_catch.StackTrace());
  } else {
    DEBUG("report_exception stack_trace empty");
  }

  DEBUG("report_exception exit");
  return ss.str();
}

// Called from Go

void V8_Init() {
  v8::Platform *platform = v8::platform::CreateDefaultPlatform();
  v8::V8::InitializePlatform(platform);
  v8::V8::Initialize();
}

ContextPtr V8_Context_New() {
  // Create a v8::Isolate
  v8::Isolate::CreateParams create_params;
  create_params.array_buffer_allocator = v8::ArrayBuffer::Allocator::NewDefaultAllocator();
  v8::Isolate* isolate = v8::Isolate::New(create_params);
  v8::Locker locker(isolate);
  v8::Isolate::Scope isolate_scope(isolate);
  v8::HandleScope handle_scope(isolate);

  v8::V8::SetCaptureStackTraceForUncaughtExceptions(true);

  v8::Local<v8::ObjectTemplate> globals = v8::ObjectTemplate::New(isolate);

  Context* context = new Context;
  context->ptr.Reset(isolate, v8::Context::New(isolate, nullptr, globals));
  context->isolate = isolate;
  return static_cast<ContextPtr>(context);
}

// releaseIsolate retrieves the V8_Context and sets ISOLATE_SCOPE, then
// resets the v8::Context and returns the v8::Isolate it used to contain.
v8::Isolate* releaseIsolate(ContextPtr context_ptr) {
  DEBUG("releaseIsolate enter");
  CONTEXT_SCOPE(context_ptr);
  context->ptr.Reset();
  DEBUG("releaseIsolate exit");
  return isolate;
}

// V8_Context_Release releases the Context, first by resetting the internal
// v8::Context then disposing of the v8::Isolate.
void V8_Context_Release(ContextPtr context_ptr) {
  DEBUG("V8_Context_Release enter");
  // Release the isolate from the context
  v8::Isolate* isolate = releaseIsolate(context_ptr);
  // Dispose of the isolate
  isolate->Dispose();
  DEBUG("V8_Context_Release exit");
}

// V8_Context_Eval compiles and run the given code inside of the context.
Result V8_Context_Eval(ContextPtr context_ptr, const char* code, const char* filename) {
  DEBUG("V8_Context_Eval enter");
  VALUE_SCOPE(context_ptr);

  v8::TryCatch try_catch;
  try_catch.SetVerbose(false);

  Result res = { nullptr, nullptr };

  v8::Local<v8::Script> script = v8::Script::Compile(
      v8::String::NewFromUtf8(isolate, code),
      v8::String::NewFromUtf8(isolate, filename));

  if (script.IsEmpty()) {
    DEBUG("V8_Context_Eval script_is_empty");
    res.e = DupString(report_exception(isolate, try_catch));
    DEBUG("V8_Context_Eval exit 1");
    return res;
  }

  v8::Local<v8::Value> result = script->Run();

  if (result.IsEmpty()) {
    DEBUG("V8_Context_Eval result_is_empty");
    res.e = DupString(report_exception(isolate, try_catch));
  } else {
    DEBUG("V8_Context_Eval result_is_full");
    V8_Persistent_Value* val = new V8_Persistent_Value(isolate, result);
    res.v_ptr = static_cast<ValuePtr>(val);
  }

  DEBUG("V8_Context_Eval exit");
	return res;
}

String V8_Value_String(ContextPtr context_ptr, ValuePtr value_ptr) {
  DEBUG("V8_Value_String enter");
  VALUE_SCOPE(context_ptr);

  v8::Local<v8::Value> value = static_cast<V8_Persistent_Value*>(value_ptr)->Get(isolate);
  DEBUG("V8_Value_String exit");
  return DupString(value->ToString());
}

void V8_Value_Release(ContextPtr context_ptr, ValuePtr value_ptr) {
  DEBUG("V8_Value_Release enter");
  VALUE_SCOPE(context_ptr);

  V8_Persistent_Value* value = static_cast<V8_Persistent_Value*>(value_ptr);
  value->Reset();
  DEBUG("V8_Value_Release exit");
  delete value;
}
