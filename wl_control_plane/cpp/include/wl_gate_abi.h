// C ABI surface for the WlGate hot-path checker. This header is pure C (no
// C++ stdlib includes) so cgo can compile it with the C compiler. The actual
// implementation lives in wl_gate.cpp (compiled as C++).
#ifndef WL_GATE_ABI_H
#define WL_GATE_ABI_H

#include <stdlib.h> /* for free() used by cgo callers */

#ifdef __cplusplus
extern "C" {
#endif

int  wl_gate_is_alive(void);
const char* wl_gate_reason(void);
void wl_gate_set_alive(int alive, const char* reason);
int  wl_gate_fetcher_enabled(const char* product, const char* fetcher);
void wl_gate_set_flags(const char* json_array);

#ifdef __cplusplus
}
#endif

#endif /* WL_GATE_ABI_H */
