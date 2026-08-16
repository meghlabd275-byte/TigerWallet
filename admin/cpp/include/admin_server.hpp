/**
 * TigerAdmin C++ Core - Server Header
 */
#pragma once

#include "admin_config.hpp"
#include "admin_logger.hpp"

#include <string>
#include <vector>
#include <map>
#include <functional>
#include <optional>
#include <thread>
#include <atomic>
#include <mutex>

namespace tiger {
namespace admin {

// ============================================================================
// HTTP Types
// ============================================================================

enum class HttpMethod { GET, POST, PUT, DELETE, PATCH };

enum class HttpStatus {
    OK = 200,
    CREATED = 201,
    NO_CONTENT = 204,
    BAD_REQUEST = 400,
    UNAUTHORIZED = 401,
    FORBIDDEN = 403,
    NOT_FOUND = 404,
    CONFLICT = 409,
    UNPROCESSABLE = 422,
    INTERNAL_ERROR = 500,
    SERVICE_UNAVAILABLE = 503
};

struct HttpRequest {
    HttpMethod method = HttpMethod::GET;
    std::string path;
    std::string body;
    std::string ip_address;
    std::map<std::string, std::string> headers;
    std::map<std::string, std::string> query_params;

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

using RequestHandler = std::function<HttpResponse(const HttpRequest&)>;
using Middleware = std::function<HttpResponse(const HttpRequest&, RequestHandler)>;

// Free function declared here, defined in admin_server.cpp
std::vector<std::string> split(const std::string& s, char delimiter);

// ============================================================================
// Router
// ============================================================================

class Router {
public:
    void add_route(HttpMethod method, const std::string& path, RequestHandler handler);
    void get(const std::string& path, RequestHandler handler);
    void post(const std::string& path, RequestHandler handler);
    void put(const std::string& path, RequestHandler handler);
    void delete_route(const std::string& path, RequestHandler handler);

    std::optional<RequestHandler> find_handler(HttpMethod method, const std::string& path);

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
// Middleware Chain
// ============================================================================

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
    explicit AdminServer(const Config& config);
    ~AdminServer();

    bool start();
    void stop();
    bool is_running() const;
    Router& router();

    void broadcast(const std::string& message);
    void send_to_client(const std::string& client_id, const std::string& message);

    uint64_t total_requests() const;
    uint64_t active_connections() const;

    void register_routes();
    void register_middleware();

private:
    void accept_connections();
    void handle_request(int client_socket, const std::string& client_ip);
    void process_request(const HttpRequest& request);

    Config config_;
    int server_socket_ = -1;
    std::atomic<bool> running_{false};
    int worker_count_;
    std::vector<std::thread> worker_threads_;
    std::atomic<uint64_t> total_requests_{0};
    std::atomic<uint64_t> active_connections_{0};

    Router router_;
    MiddlewareChain middleware_chain_;

    std::mutex ws_mutex_;
    std::map<std::string, int> ws_clients_;
};

} // namespace admin
} // namespace tiger
