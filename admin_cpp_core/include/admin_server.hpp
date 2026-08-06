/**
 * TigerAdmin C++ Core - HTTP Server
 * High-performance async HTTP server with WebSocket support
 */

#ifndef TIGER_ADMIN_SERVER_HPP
#define TIGER_ADMIN_SERVER_HPP

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <functional>
#include <memory>
#include <atomic>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <optional>
#include "admin_config.hpp"
#include "admin_models.hpp"
#include "admin_connection_pool.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// HTTP Types
// ============================================================================

enum class HttpMethod {
    GET,
    POST,
    PUT,
    DELETE,
    PATCH,
    OPTIONS
};

enum class HttpStatus {
    OK = 200,
    CREATED = 201,
    NO_CONTENT = 204,
    BAD_REQUEST = 400,
    UNAUTHORIZED = 401,
    FORBIDDEN = 403,
    NOT_FOUND = 404,
    CONFLICT = 409,
    INTERNAL_SERVER_ERROR = 500,
    SERVICE_UNAVAILABLE = 503
};

struct HttpRequest {
    HttpMethod method;
    std::string path;
    std::string query_string;
    std::map<std::string, std::string> headers;
    std::map<std::string, std::string> query_params;
    std::string body;
    std::string ip_address;
    std::string user_agent;
    
    std::optional<std::string> get_header(const std::string& name) const;
    std::optional<std::string> get_query(const std::string& name) const;
};

struct HttpResponse {
    HttpStatus status = HttpStatus::OK;
    std::string body;
    std::map<std::string, std::string> headers;
    
    static HttpResponse json(HttpStatus status, const std::string& json_body);
    static HttpResponse success(const std::string& message);
    static HttpResponse error(HttpStatus status, const std::string& message);
};

// ============================================================================
// WebSocket Types
// ============================================================================

struct WebSocketMessage {
    std::string data;
    std::string client_id;
    bool binary = false;
};

using WebSocketCallback = std::function<void(const WebSocketMessage&)>;

// ============================================================================
// Request Handler
// ============================================================================

using RequestHandler = std::function<HttpResponse(const HttpRequest&)>;

// ============================================================================
// Router
// ============================================================================

class Router {
public:
    void add_route(HttpMethod method, const std::string& path, 
                   RequestHandler handler);
    
    void get(const std::string& path, RequestHandler handler);
    void post(const std::string& path, RequestHandler handler);
    void put(const std::string& path, RequestHandler handler);
    void delete_route(const std::string& path, RequestHandler handler);
    
    std::optional<RequestHandler> find_handler(HttpMethod method, 
                                                const std::string& path);
    
private:
    struct Route {
        HttpMethod method;
        std::string path;
        RequestHandler handler;
    };
    
    std::vector<Route> routes_;
    
    bool match_path(const std::string& pattern, const std::string& path,
                    std::map<std::string, std::string>& params);
};

// ============================================================================
// Middleware
// ============================================================================

using Middleware = std::function<HttpResponse(const HttpRequest&, 
                                              RequestHandler)>;

class MiddlewareChain {
public:
    void add(Middleware middleware);
    RequestHandler wrap(RequestHandler handler);
    
private:
    std::vector<Middleware> middlewares_;
};

// ============================================================================
// Admin Server
// ============================================================================

class AdminServer {
public:
    AdminServer(const Config& config);
    ~AdminServer();
    
    // Start/stop
    bool start();
    void stop();
    bool is_running() const;
    
    // Routing
    Router& router();
    
    // WebSocket
    void broadcast(const std::string& message);
    void send_to_client(const std::string& client_id, 
                        const std::string& message);
    
    // Stats
    uint64_t total_requests() const;
    uint64_t active_connections() const;
    
private:
    Config config_;
    bool running_ = false;
    std::atomic<uint64_t> total_requests_{0};
    std::atomic<uint64_t> active_connections_{0};
    
    Router router_;
    MiddlewareChain middleware_;
    
    std::vector<std::thread> worker_threads_;
    int worker_count_;
    
    // Socket
    int server_socket_ = -1;
    
    // WebSocket clients
    std::mutex ws_mutex_;
    std::map<std::string, int> ws_clients_;
    
    void accept_connections();
    void handle_request(int client_socket);
    void process_request(const HttpRequest& request);
    
    // Route registration
    void register_routes();
    void register_middleware();
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_SERVER_HPP
