#ifndef V8_C_BRIDGE_H
#define V8_C_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

// Go pointer types
typedef void* ContextPtr;
typedef void* ValuePtr;

// Go accessible string type
typedef struct {
  const char* ptr;
  int len;
} String;

// Go accessible error type
typedef String Error;

// Go accessible Result type
typedef struct {
  ValuePtr v_ptr;
  Error e;
} Result;

typedef struct { int Major, Minor, Build, Patch; } Version;
extern Version version;

// Go accessible functions
extern void       V8_Init();
extern ContextPtr V8_Context_New();
extern void       V8_Context_Release(ContextPtr ptr);
extern Result     V8_Context_Eval(ContextPtr ptr, const char* code, const char* filename);
extern String     V8_Value_String(ContextPtr context_ptr, ValuePtr value_ptr);
extern void       V8_Value_Release(ContextPtr context_ptr, ValuePtr value_ptr);

#ifdef __cplusplus
}
#endif

#endif // !defined(V8_C_BRIDGE_H)
