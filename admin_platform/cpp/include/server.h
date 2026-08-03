#pragma once

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <functional>
#include <thread>
#include <atomic>
#include <mutex>
#include <queue>
#include <uv.h>

namespace tiger {

// HTTP Types
using HTTPMethod = enum {
    HTTP_GET,
    HTTP_POST,
    HTTP_PUT,
    HTTP_PATCH,
    HTTP_DELETE,
    HTTP_OPTIONS,
    HTTP_HEAD
};

using HTTPStatus = enum {
    // 1xx Informational
    HTTP_100_CONTINUE,
    HTTP_101_SWITCHING_PROTOCOLS,
    
    // 2xx Success
    HTTP_200_OK,
    HTTP_201_CREATED,
    HTTP_202_ACCEPTED,
    HTTP_204_NO_CONTENT,
    
    // 3xx Redirection
    HTTP_301_MOVED_PERMANENTLY,
    HTTP_302_FOUND,
    HTTP_304_NOT_MODIFIED,
    
    // 4xx Client Error
    HTTP_400_BAD_REQUEST,
    HTTP_401_UNAUTHORIZED,
    HTTP_403_FORBIDDEN,
    HTTP_404_NOT_FOUND,
    HTTP_405_METHOD_NOT_ALLOWED,
    HTTP_408_REQUEST_TIMEOUT,
    HTTP_409_CONFLICT,
    HTTP_422_UNPROCESSABLE_ENTITY,
    HTTP_429_TOO_MANY_REQUESTS,
    
    // 5xx Server Error
    HTTP_500_INTERNAL_SERVER_ERROR,
    HTTP_501_NOT_IMPLEMENTED,
    HTTP_502_BAD_GATEWAY,
    HTTP_503_SERVICE_UNAVAILABLE,
    HTTP_504_GATEWAY_TIMEOUT
};

struct HTTPRequest {
    std::string id;
    HTTPMethod method;
    std::string path;
    std::string query_string;
    std::string body;
    std::map<std::string, std::string> headers;
    std::map<std::string, std::string> query_params;
    std::map<std::string, std::string> path_params;
    std::string remote_ip;
    std::string remote_port;
    std::string protocol;
    
    std::string get_header(const std::string& name) const;
    std::string get_query_param(const std::string& name) const;
    std::string get_path_param(const std::string& name) const;
    bool has_header(const std::string& name) const;
    bool has_query_param(const std::string& name) const;
};

struct HTTPResponse {
    HTTPStatus status_code;
    std::string body;
    std::map<std::string, std::string> headers;
    
    HTTPResponse() : status_code(HTTP_200_OK) {}
    
    HTTPResponse& status(HTTPStatus code) {
        status_code = code;
        return *this;
    }
    
    HTTPResponse& json(const std::string& json_body) {
        body = json_body;
        headers["Content-Type"] = "application/json";
        return *this;
    }
    
    HTTPResponse& text(const std::string& text_body) {
        body = text_body;
        headers["Content-Type"] = "text/plain";
        return *this;
    }
    
    HTTPResponse& html(const std::string& html_body) {
        body = html_body;
        headers["Content-Type"] = "text/html";
        return *this;
    }
    
    HTTPResponse& header(const std::string& name, const std::string& value) {
        headers[name] = value;
        return *this;
    }
    
    HTTPResponse& cors() {
        headers["Access-Control-Allow-Origin"] = "*";
        headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, PATCH, DELETE, OPTIONS";
        headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization, X-Requested-With";
        return *this;
    }
    
    HTTPResponse& cache(int seconds) {
        headers["Cache-Control"] = "public, max-age=" + std::to_string(seconds);
        return *this;
    }
    
    HTTPResponse& no_cache() {
        headers["Cache-Control"] = "no-store, no-cache, must-revalidate";
        headers["Pragma"] = "no-cache";
        return *this;
    }
};

// Route handler
using RouteHandler = std::function<HTTPResponse(const HTTPRequest&)>;

// Middleware
using Middleware = std::function<bool(HTTPRequest&, HTTPResponse&)>;

// Route definition
struct Route {
    HTTPMethod method;
    std::string path;
    RouteHandler handler;
    std::vector<Middleware> middlewares;
    std::string name;
    bool require_auth;
    std::vector<std::string> required_permissions;
    
    Route(HTTPMethod m, const std::string& p, RouteHandler h)
        : method(m), path(p), handler(std::move(h)), 
          require_auth(false) {}
};

// URL path pattern parser
class PathPattern {
public:
    PathPattern(const std::string& pattern);
    ~PathPattern() = default;
    
    bool match(const std::string& path, std::map<std::string, std::string>& params);
    std::string pattern() const { return pattern_; }
    
private:
    std::string pattern_;
    std::vector<std::string> segments_;
    std::map<size_t, std::pair<std::string, bool>> param_positions_; // position -> (name, is_wildcard)
};

// HTTP Server
class HTTPServer {
public:
    HTTPServer(const std::string& host, int port);
    ~HTTPServer();
    
    // Server lifecycle
    bool start();
    void stop();
    bool is_running() const;
    
    // Configuration
    void set_num_threads(int num_threads);
    void set_request_timeout(int timeout_seconds);
    void set_max_body_size(size_t max_size);
    
    // Route registration
    HTTPServer& get(const std::string& path, RouteHandler handler);
    HTTPServer& post(const std::string& path, RouteHandler handler);
    HTTPServer& put(const std::string& path, RouteHandler handler);
    HTTPServer& patch(const std::string& path, RouteHandler handler);
    HTTPServer& delete_(const std::string& path, RouteHandler handler);
    HTTPServer& options(const std::string& path, RouteHandler handler);
    
    HTTPServer& use(Middleware middleware);
    HTTPServer& use_auth(Middleware auth_middleware);
    
    // Route with name and permissions
    HTTPServer& get(const std::string& path, RouteHandler handler, 
                   const std::string& name, bool require_auth = false,
                   const std::vector<std::string>& permissions = {});
    HTTPServer& post(const std::string& path, RouteHandler handler,
                    const std::string& name, bool require_auth = false,
                    const std::vector<std::string>& permissions = {});
                    
    // Static files
    HTTPServer& serve_static(const std::string& mount_point, const std::string& directory);
    HTTPServer& serve_file(const std::string& path, const std::string& file_path);
    
    // Error handlers
    HTTPServer& set_error_handler(HTTPStatus status, RouteHandler handler);
    HTTPServer& set_not_found_handler(RouteHandler handler);
    HTTPServer& set_internal_error_handler(RouteHandler handler);
    
    // Statistics
    struct ServerStats {
        uint64_t total_requests;
        uint64_t active_requests;
        uint64_t completed_requests;
        uint64_t failed_requests;
        uint64_t total_bytes_sent;
        uint64_t total_bytes_received;
        std::map<int, uint64_t> requests_by_status;
        std::map<std::string, uint64_t> requests_by_method;
        std::map<std::string, uint64_t> requests_by_path;
    };
    
    ServerStats get_stats() const;
    void reset_stats();
    
    // Event handlers
    std::function<void(const HTTPRequest&, const HTTPResponse&)> on_request;
    std::function<void(const HTTPRequest&, const std::exception&)> on_error;
    
private:
    std::string host_;
    int port_;
    int num_threads_;
    int request_timeout_;
    size_t max_body_size_;
    std::atomic<bool> running_;
    
    std::vector<Route> routes_;
    std::vector<Middleware> global_middlewares_;
    std::vector<Middleware> auth_middlewares_;
    std::map<HTTPStatus, RouteHandler> error_handlers_;
    RouteHandler not_found_handler_;
    RouteHandler internal_error_handler_;
    
    std::map<std::string, std::string> static_files_;
    
    ServerStats stats_;
    mutable std::mutex stats_mutex_;
    
    // Request processing
    void process_request(uv_stream_t* client, const char* data, size_t len);
    HTTPResponse handle_request(const HTTPRequest& request);
    Route* find_route(const HTTPRequest& request);
    
    // Response sending
    void send_response(uv_stream_t* client, const HTTPRequest& request, const HTTPResponse& response);
    std::string build_response_string(const HTTPRequest& request, const HTTPResponse& response);
    
    // Utility
    HTTPMethod parse_method(const std::string& method);
    std::string status_to_string(HTTPStatus status);
    std::map<std::string, std::string> parse_query_string(const std::string& query);
    std::map<std::string, std::string> parse_headers(const std::vector<std::string>& lines);
};

// WebSocket types
struct WSMessage {
    enum Type {
        TEXT,
        BINARY,
        PING,
        PONG,
        CLOSE
    };
    
    Type type;
    std::string data;
    int opcode;
};

using WSHandler = std::function<void(class WebSocket* ws, const WSMessage& message)>;

class WebSocket {
public:
    WebSocket(uv_stream_t* stream, const std::string& protocol);
    ~WebSocket();
    
    void send(const std::string& message);
    void send_binary(const std::string& data);
    void ping();
    void close(int code = 1000, const std::string& reason = "");
    
    std::string remote_ip() const;
    std::string path() const;
    bool is_alive() const;
    
    void set_handler(WSHandler handler);
    
private:
    uv_stream_t* stream_;
    std::string protocol_;
    std::string path_;
    bool alive_;
    WSHandler handler_;
    
    void process_message(const char* data, size_t len);
    void write_frame(const char* data, size_t len, int opcode);
};

class WebSocketServer {
public:
    WebSocketServer(HTTPServer* http_server);
    ~WebSocketServer();
    
    void on_connection(uv_stream_t* stream);
    void handle_message(WebSocket* ws, const WSMessage& message);
    
private:
    HTTPServer* http_server_;
    std::map<std::string, WSHandler> handlers_;
};

} // namespace tiger
