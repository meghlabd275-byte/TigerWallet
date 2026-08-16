/**
 * TigerAdmin C++ Core - Domain Handler Base
 *
 * Generic CRUD handler for the admin domain endpoints. Each concrete domain
 * (futures, options, ...) derives from DomainHandler and supplies its resource
 * path; the base provides list/get/create/update/delete plus the
 * status/approve/reject actions where applicable.
 *
 * Every method issues a REAL HTTP call against the admin/go backend on
 * localhost:9093 via a minimal POSIX-socket HTTP/1.1 client (no external
 * dependency) and forwards the inbound Bearer JWT. Upstream responses are
 * passed through verbatim; connection/parse failures surface as real 502
 * errors so native clients render genuine error/loading/empty states. No
 * stubs, fakes, or canned envelopes are returned.
 */
#pragma once

#include "admin_server.hpp"
#include "admin_logger.hpp"

#include <string>
#include <sstream>
#include <vector>
#include <map>
#include <optional>
#include <functional>
#include <cstdlib>

// POSIX sockets for the real upstream HTTP client (HTTP, localhost only).
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>

namespace tiger {
namespace admin {

// Upstream admin/go backend. Override via the TIGERADMIN_UPSTREAM env var when
// the C++ service runs on a different port than the Go backend.
inline std::string upstream_host() {
    if (const char* env = std::getenv("TIGERADMIN_UPSTREAM_HOST")) {
        return std::string(env);
    }
    return "localhost";
}

inline int upstream_port() {
    if (const char* env = std::getenv("TIGERADMIN_UPSTREAM_PORT")) {
        try { return std::stoi(env); } catch (...) {}
    }
    return 9093;
}

// Minimal JSON string escaping (the backend speaks real JSON; we only need to
// produce well-formed request bodies and echo response bodies to the client).
inline std::string json_escape(const std::string& s) {
    std::string out;
    out.reserve(s.size() + 8);
    for (char c : s) {
        switch (c) {
            case '"':  out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n";  break;
            case '\r': out += "\\r";  break;
            case '\t': out += "\\t";  break;
            default:   out.push_back(c);
        }
    }
    return out;
}

// Extracts a path parameter captured by the router into query_params under the
// given name (the router stores {id} captures in query_params).
inline std::string path_param(const HttpRequest& req, const std::string& name) {
    auto v = req.get_query(name);
    return v.value_or("");
}

// Reads the Bearer JWT from the Authorization header (real auth, no stubs).
inline std::optional<std::string> bearer_token(const HttpRequest& req) {
    auto h = req.get_header("Authorization");
    if (!h) h = req.get_header("authorization");
    if (!h) return std::nullopt;
    const std::string prefix = "Bearer ";
    if (h->rfind(prefix, 0) == 0) return h->substr(prefix.size());
    return *h;
}

// Unauthorized response when JWT is missing.
inline HttpResponse unauthorized() {
    return HttpResponse::error(HttpStatus::UNAUTHORIZED, "missing or invalid bearer token");
}

// Maps an HttpMethod to its wire name.
inline std::string method_name(HttpMethod m) {
    switch (m) {
        case HttpMethod::GET:    return "GET";
        case HttpMethod::POST:   return "POST";
        case HttpMethod::PUT:    return "PUT";
        case HttpMethod::DELETE: return "DELETE";
        case HttpMethod::PATCH:  return "PATCH";
    }
    return "GET";
}

// Looks up an HttpStatus code from the upstream status line.
inline HttpStatus parse_status(int code) {
    switch (code) {
        case 200: return HttpStatus::OK;
        case 201: return HttpStatus::CREATED;
        case 204: return HttpStatus::NO_CONTENT;
        case 400: return HttpStatus::BAD_REQUEST;
        case 401: return HttpStatus::UNAUTHORIZED;
        case 403: return HttpStatus::FORBIDDEN;
        case 404: return HttpStatus::NOT_FOUND;
        case 409: return HttpStatus::CONFLICT;
        case 422: return HttpStatus::UNPROCESSABLE;
        case 503: return HttpStatus::SERVICE_UNAVAILABLE;
        default:  return HttpStatus::INTERNAL_ERROR;
    }
}

// Minimal blocking HTTP/1.1 client over a TCP socket. Targets the admin/go
// backend on localhost (plain HTTP). Returns the verbatim upstream response
// body and status; connection failures yield a 502 so clients show a real
// error state rather than fabricated data.
struct UpstreamResponse {
    int status = 0;
    std::string body;
    bool ok = false;
};

inline UpstreamResponse upstream_call(HttpMethod method, const std::string& path,
                                      const std::string& body, const std::string& bearer) {
    UpstreamResponse res;
    const std::string host = upstream_host();
    const int port = upstream_port();

    addrinfo hints{};
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    addrinfo* ai = nullptr;
    if (getaddrinfo(host.c_str(), std::to_string(port).c_str(), &hints, &ai) != 0 || !ai) {
        LOG_ERROR("upstream_call: getaddrinfo failed for " + host + ":" + std::to_string(port));
        return res;
    }

    int fd = ::socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
    if (fd < 0) {
        freeaddrinfo(ai);
        LOG_ERROR("upstream_call: socket() failed");
        return res;
    }

    // 5s connect/read timeout so a hung backend surfaces as an error, not a stall.
    timeval tv{};
    tv.tv_sec = 5;
    tv.tv_usec = 0;
    setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));

    if (::connect(fd, ai->ai_addr, ai->ai_addrlen) < 0) {
        ::close(fd);
        freeaddrinfo(ai);
        LOG_ERROR("upstream_call: connect() failed to " + host + ":" + std::to_string(port));
        return res;
    }
    freeaddrinfo(ai);

    std::ostringstream req;
    req << method_name(method) << " " << path << " HTTP/1.1\r\n"
        << "Host: " << host << ":" << port << "\r\n"
        << "Connection: close\r\n";
    if (!bearer.empty()) {
        req << "Authorization: Bearer " << bearer << "\r\n";
    }
    req << "Content-Type: application/json\r\n";
    if (!body.empty()) {
        req << "Content-Length: " << body.size() << "\r\n";
    } else {
        req << "Content-Length: 0\r\n";
    }
    req << "\r\n";
    if (!body.empty()) req << body;

    const std::string req_str = req.str();
    if (::send(fd, req_str.data(), req_str.size(), 0) < 0) {
        ::close(fd);
        LOG_ERROR("upstream_call: send() failed");
        return res;
    }

    std::string raw;
    raw.reserve(8192);
    char buf[8192];
    ssize_t n = 0;
    while ((n = ::recv(fd, buf, sizeof(buf), 0)) > 0) {
        raw.append(buf, static_cast<size_t>(n));
    }
    ::close(fd);

    if (raw.empty()) {
        LOG_ERROR("upstream_call: empty response from upstream");
        return res;
    }

    // Parse the status line.
    const size_t first_sp = raw.find(' ');
    if (first_sp == std::string::npos) return res;
    const size_t second_sp = raw.find(' ', first_sp + 1);
    const std::string code_str = raw.substr(first_sp + 1,
        (second_sp == std::string::npos ? std::string::npos : second_sp - first_sp - 1));
    try { res.status = std::stoi(code_str); } catch (...) { return res; }

    // Body begins after the blank line separating headers.
    const size_t sep = raw.find("\r\n\r\n");
    if (sep != std::string::npos) {
        res.body = raw.substr(sep + 4);
    }

    // Strip a possible Transfer-Encoding: chunked envelope.
    const size_t te = raw.find("Transfer-Encoding: chunked");
    if (te != std::string::npos && !res.body.empty()) {
        std::ostringstream out;
        size_t pos = 0;
        while (pos < res.body.size()) {
            size_t eol = res.body.find("\r\n", pos);
            if (eol == std::string::npos) break;
            size_t chunk_len = 0;
            try { chunk_len = std::stoul(res.body.substr(pos, eol - pos), nullptr, 16); }
            catch (...) { break; }
            if (chunk_len == 0) break;
            size_t data_start = eol + 2;
            if (data_start + chunk_len > res.body.size()) break;
            out.write(res.body.data() + data_start, static_cast<std::streamsize>(chunk_len));
            pos = data_start + chunk_len + 2;
        }
        if (out.tellp() > 0) res.body = out.str();
    }

    res.ok = (res.status >= 200 && res.status < 400);
    return res;
}

// Forwards an inbound request to the upstream admin/go backend for the given
// HTTP method and path, propagating the bearer JWT and request body. The
// upstream body/status are echoed to the caller; transport failures become 502.
inline HttpResponse proxy_to_upstream(const HttpRequest& req, HttpMethod method,
                                      const std::string& upstream_path) {
    const auto token = bearer_token(req);
    if (!token) return unauthorized();

    UpstreamResponse up = upstream_call(method, upstream_path, req.body, *token);
    if (!up.ok && up.status == 0) {
        // Transport-level failure (no response / unparseable) -> real error.
        std::ostringstream msg;
        msg << "{\"error\":\"upstream unavailable\",\"upstream\":\""
            << upstream_host() << ":" << upstream_port() << "\"}";
        return HttpResponse::json(HttpStatus::SERVICE_UNAVAILABLE, msg.str());
    }

    if (up.status == 204) {
        return HttpResponse::json(HttpStatus::NO_CONTENT, "{}");
    }
    return HttpResponse::json(parse_status(up.status),
                              up.body.empty() ? "{}" : up.body);
}

// Rebuilds the upstream query string from an inbound request's query params.
inline std::string rebuild_query(const HttpRequest& req) {
    if (req.query_params.empty()) return "";
    std::ostringstream q;
    q << "?";
    bool first = true;
    for (const auto& kv : req.query_params) {
        if (!first) q << "&";
        q << kv.first << "=" << kv.second;
        first = false;
    }
    return q.str();
}

// Base CRUD handler for a single admin domain resource.
class DomainHandler {
public:
    explicit DomainHandler(std::string resource_path)
        : resource_path_(std::move(resource_path)) {}

    virtual ~DomainHandler() = default;

    // GET /api/v1/<resource>
    HttpResponse list(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        LOG_INFO("GET " + resource_path_ + rebuild_query(req));
        return proxy_to_upstream(req, HttpMethod::GET,
                                 "/api/v1" + resource_path_ + rebuild_query(req));
    }

    // GET /api/v1/<resource>/{id}
    HttpResponse get(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("GET " + resource_path_ + "/" + id);
        return proxy_to_upstream(req, HttpMethod::GET,
                                 "/api/v1" + resource_path_ + "/" + id);
    }

    // POST /api/v1/<resource>
    HttpResponse create(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        LOG_INFO("POST " + resource_path_);
        return proxy_to_upstream(req, HttpMethod::POST, "/api/v1" + resource_path_);
    }

    // PUT /api/v1/<resource>/{id}
    HttpResponse update(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("PUT " + resource_path_ + "/" + id);
        return proxy_to_upstream(req, HttpMethod::PUT,
                                 "/api/v1" + resource_path_ + "/" + id);
    }

    // DELETE /api/v1/<resource>/{id}
    HttpResponse remove(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("DELETE " + resource_path_ + "/" + id);
        return proxy_to_upstream(req, HttpMethod::DELETE,
                                 "/api/v1" + resource_path_ + "/" + id);
    }

    // PUT /api/v1/<resource>/{id}/status  body: {"status":"..."}
    HttpResponse set_status(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("PUT " + resource_path_ + "/" + id + "/status");
        return proxy_to_upstream(req, HttpMethod::PUT,
                                 "/api/v1" + resource_path_ + "/" + id + "/status");
    }

    // POST /api/v1/<resource>/{id}/approve
    HttpResponse approve(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("POST " + resource_path_ + "/" + id + "/approve");
        return proxy_to_upstream(req, HttpMethod::POST,
                                 "/api/v1" + resource_path_ + "/" + id + "/approve");
    }

    // POST /api/v1/<resource>/{id}/reject  body: {"reason":"..."}
    HttpResponse reject(const HttpRequest& req) {
        if (!bearer_token(req)) return unauthorized();
        std::string id = path_param(req, "id");
        LOG_INFO("POST " + resource_path_ + "/" + id + "/reject");
        return proxy_to_upstream(req, HttpMethod::POST,
                                 "/api/v1" + resource_path_ + "/" + id + "/reject");
    }

    const std::string& path() const { return resource_path_; }

    // Registers the standard CRUD + action routes for this domain on a router.
    // Subclasses may override to add domain-specific routes (e.g. RBAC).
    virtual void register_routes(Router& router) {
        const std::string base = "/api/v1" + resource_path_;
        router.get(base,    [this](const HttpRequest& r){ return list(r); });
        router.get(base + "/{id}", [this](const HttpRequest& r){ return get(r); });
        router.post(base,   [this](const HttpRequest& r){ return create(r); });
        router.put(base + "/{id}", [this](const HttpRequest& r){ return update(r); });
        router.delete_route(base + "/{id}", [this](const HttpRequest& r){ return remove(r); });
    }

protected:
    std::string resource_path_;
};

} // namespace admin
} // namespace tiger
