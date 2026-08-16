// WlGate implementation + C ABI. The hot-path read functions are inline in
// the header; this translation unit holds the C ABI wrappers + the JSON flag
// parser (called only on heartbeat, not on the hot path).
#include "wl_gate.hpp"
#include <cstdlib>
#include <cstring>

extern "C" {

int wl_gate_is_alive() {
    return tigerwallet::wl::WlGate::instance().is_alive() ? 1 : 0;
}

// Returns a pointer to a static buffer holding the reason. The caller MUST
// copy before the next call (standard C-string-return convention).
const char* wl_gate_reason() {
    static thread_local std::string buf;
    buf = tigerwallet::wl::WlGate::instance().reason();
    return buf.c_str();
}

void wl_gate_set_alive(int alive, const char* reason) {
    tigerwallet::wl::WlGate::instance().set_alive(alive != 0, reason);
}

int wl_gate_fetcher_enabled(const char* product, const char* fetcher) {
    if (!product || !fetcher) return 0;
    return tigerwallet::wl::WlGate::instance().fetcher_enabled(product, fetcher) ? 1 : 0;
}

// Minimal JSON array parser for the flag snapshot. Accepts:
//   [{"product":"p","fetcher":"f","enabled":true}, ...]
// No external deps (no nlohmann/json) to keep the lib dependency-free.
void wl_gate_set_flags(const char* json_array) {
    if (!json_array) {
        tigerwallet::wl::WlGate::instance().set_flags({});
        return;
    }
    std::vector<tigerwallet::wl::FlagEntry> flags;
    const char* p = json_array;
    while (*p) {
        // find next "product"
        const char* prod_key = strstr(p, "\"product\"");
        if (!prod_key) break;
        const char* pc = strchr(prod_key + 9, ':');
        if (!pc) break;
        const char* ps = strchr(pc, '"');
        if (!ps) break;
        const char* pe = strchr(ps + 1, '"');
        if (!pe) break;
        std::string product(ps + 1, pe);

        const char* fetch_key = strstr(pe, "\"fetcher\"");
        if (!fetch_key) break;
        const char* fc = strchr(fetch_key + 9, ':');
        if (!fc) break;
        const char* fs = strchr(fc, '"');
        if (!fs) break;
        const char* fe = strchr(fs + 1, '"');
        if (!fe) break;
        std::string fetcher(fs + 1, fe);

        const char* en_key = strstr(fe, "\"enabled\"");
        if (!en_key) break;
        const char* ec = strchr(en_key + 9, ':');
        if (!ec) break;
        ec++;
        while (*ec == ' ' || *ec == '\t') ec++;
        bool enabled = (strncmp(ec, "true", 4) == 0);

        flags.push_back({std::move(product), std::move(fetcher), enabled});
        p = en_key;
    }
    tigerwallet::wl::WlGate::instance().set_flags(flags);
}

} // extern "C"
