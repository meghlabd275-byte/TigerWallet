/**
 * TigerAdmin C++ Core - Server Implementation
 */

#include "admin_server.hpp"
#include "admin_logger.hpp"
#include "admin_domains.hpp"
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <cstring>
#include <errno.h>

namespace tiger {
namespace admin {

// ============================================================================
// HTTP Request/Response Implementation
// ============================================================================

std::optional<std::string> HttpRequest::get_header(const std::string& name) const {
    auto it = headers.find(name);
    if (it != headers.end()) {
        return it->second;
    }
    return std::nullopt;
}

std::optional<std::string> HttpRequest::get_query(const std::string& name) const {
    auto it = query_params.find(name);
    if (it != query_params.end()) {
        return it->second;
    }
    return std::nullopt;
}

HttpResponse HttpResponse::json(HttpStatus status, const std::string& json_body) {
    HttpResponse response;
    response.status = status;
    response.body = json_body;
    response.headers["Content-Type"] = "application/json";
    return response;
}

HttpResponse HttpResponse::success(const std::string& message) {
    return json(HttpStatus::OK, "{\"message\":\"" + message + "\"}");
}

HttpResponse HttpResponse::error(HttpStatus status, const std::string& message) {
    return json(status, "{\"error\":\"" + message + "\"}");
}

// ============================================================================
// Router Implementation
// ============================================================================

void Router::add_route(HttpMethod method, const std::string& path,
                        RequestHandler handler) {
    routes_.push_back({method, path, handler});
}

void Router::get(const std::string& path, RequestHandler handler) {
    add_route(HttpMethod::GET, path, handler);
}

void Router::post(const std::string& path, RequestHandler handler) {
    add_route(HttpMethod::POST, path, handler);
}

void Router::put(const std::string& path, RequestHandler handler) {
    add_route(HttpMethod::PUT, path, handler);
}

void Router::delete_route(const std::string& path, RequestHandler handler) {
    add_route(HttpMethod::DELETE, path, handler);
}

std::optional<RequestHandler> Router::find_handler(
    HttpMethod method, const std::string& path) {
    
    std::map<std::string, std::string> params;
    
    for (const auto& route : routes_) {
        if (route.method == method && match_path(route.path, path, params)) {
            // Return a wrapped handler that populates params
            return [handler = route.handler, params](const HttpRequest& req) {
                HttpRequest modified_req = req;
                for (const auto& p : params) {
                    modified_req.query_params[p.first] = p.second;
                }
                return handler(modified_req);
            };
        }
    }
    
    return std::nullopt;
}

bool Router::match_path(const std::string& pattern, const std::string& path,
                        std::map<std::string, std::string>& params) {
    // Simple path matching with {param} support
    auto pattern_parts = split(pattern, '/');
    auto path_parts = split(path, '/');
    
    if (pattern_parts.size() != path_parts.size()) {
        return false;
    }
    
    for (size_t i = 0; i < pattern_parts.size(); ++i) {
        if (pattern_parts[i].empty()) continue;
        
        if (pattern_parts[i][0] == '{' && pattern_parts[i].back() == '}') {
            // This is a parameter
            std::string param_name = pattern_parts[i].substr(1, 
                pattern_parts[i].length() - 2);
            params[param_name] = path_parts[i];
        } else if (pattern_parts[i] != path_parts[i]) {
            return false;
        }
    }
    
    return true;
}

std::vector<std::string> split(const std::string& s, char delimiter) {
    std::vector<std::string> parts;
    std::string part;
    for (char c : s) {
        if (c == delimiter) {
            parts.push_back(part);
            part.clear();
        } else {
            part += c;
        }
    }
    parts.push_back(part);
    return parts;
}

// ============================================================================
// Middleware Chain Implementation
// ============================================================================

void MiddlewareChain::add(Middleware middleware) {
    middlewares_.push_back(middleware);
}

RequestHandler MiddlewareChain::wrap(RequestHandler handler) {
    return [this, handler](const HttpRequest& req) {
        RequestHandler current = handler;
        
        // Apply middlewares in reverse order
        for (auto it = middlewares_.rbegin(); 
             it != middlewares_.rend(); ++it) {
            auto middleware = *it;
            auto next = current;
            current = [middleware, next](const HttpRequest& r) {
                return middleware(r, next);
            };
        }
        
        return current(req);
    };
}

// ============================================================================
// Admin Server Implementation
// ============================================================================

AdminServer::AdminServer(const Config& config) 
    : config_(config), worker_count_(config.worker_threads) {}

AdminServer::~AdminServer() {
    stop();
}

bool AdminServer::start() {
    // Create socket
    server_socket_ = socket(AF_INET, SOCK_STREAM, 0);
    if (server_socket_ < 0) {
        LOG_ERROR("Failed to create socket: " + std::string(strerror(errno)));
        return false;
    }
    
    // Set socket options
    int opt = 1;
    setsockopt(server_socket_, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    
    // Bind
    struct sockaddr_in address;
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = INADDR_ANY;
    address.sin_port = htons(config_.port);
    
    if (bind(server_socket_, (struct sockaddr*)&address, sizeof(address)) < 0) {
        LOG_ERROR("Failed to bind to port " + std::to_string(config_.port));
        return false;
    }
    
    // Listen
    if (listen(server_socket_, 128) < 0) {
        LOG_ERROR("Failed to listen on socket");
        return false;
    }
    
    running_ = true;
    
    // Start worker threads
    for (int i = 0; i < worker_count_; ++i) {
        worker_threads_.emplace_back(&AdminServer::accept_connections, this);
    }
    
    LOG_INFO("Server started on " + config_.host + ":" + 
             std::to_string(config_.port));
    
    return true;
}

void AdminServer::stop() {
    if (!running_) return;
    
    running_ = false;
    
    // Close server socket
    if (server_socket_ >= 0) {
        close(server_socket_);
        server_socket_ = -1;
    }
    
    // Join worker threads
    for (auto& thread : worker_threads_) {
        if (thread.joinable()) {
            thread.join();
        }
    }
    
    LOG_INFO("Server stopped");
}

bool AdminServer::is_running() const {
    return running_;
}

Router& AdminServer::router() {
    return router_;
}

void AdminServer::broadcast(const std::string& message) {
    std::lock_guard<std::mutex> lock(ws_mutex_);
    // Broadcast to all connected clients
    for (const auto& client : ws_clients_) {
        // Send to each client
    }
}

void AdminServer::send_to_client(const std::string& client_id,
                                  const std::string& message) {
    std::lock_guard<std::mutex> lock(ws_mutex_);
    auto it = ws_clients_.find(client_id);
    if (it != ws_clients_.end()) {
        // Send to specific client
    }
}

uint64_t AdminServer::total_requests() const {
    return total_requests_.load();
}

uint64_t AdminServer::active_connections() const {
    return active_connections_.load();
}

void AdminServer::accept_connections() {
    while (running_) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);
        
        int client_socket = accept(server_socket_, 
                                   (struct sockaddr*)&client_addr, 
                                   &client_len);
        
        if (client_socket < 0) {
            if (running_) {
                LOG_ERROR("Failed to accept connection");
            }
            continue;
        }
        
        active_connections_++;
        
        // Handle request in separate thread or queue
        std::thread(&AdminServer::handle_request, this, client_socket,
                    std::string(inet_ntoa(client_addr.sin_addr))).detach();
    }
}

void AdminServer::handle_request(int client_socket, const std::string& client_ip) {
    char buffer[8192] = {0};
    ssize_t bytes_read = read(client_socket, buffer, sizeof(buffer) - 1);
    
    if (bytes_read > 0) {
        buffer[bytes_read] = '\0';
        
        // Parse HTTP request
        HttpRequest request;
        request.ip_address = client_ip;
        
        // Simple HTTP parsing
        std::string request_str(buffer);
        auto lines = split(request_str, '\n');
        
        if (!lines.empty()) {
            auto request_line = split(lines[0], ' ');
            if (request_line.size() >= 2) {
                // Parse method
                if (request_line[0] == "GET") request.method = HttpMethod::GET;
                else if (request_line[0] == "POST") request.method = HttpMethod::POST;
                else if (request_line[0] == "PUT") request.method = HttpMethod::PUT;
                else if (request_line[0] == "DELETE") request.method = HttpMethod::DELETE;
                
                // Parse path
                auto path_parts = split(request_line[1], '?');
                request.path = path_parts[0];
                
                if (path_parts.size() > 1) {
                    // Parse query string
                    auto query_params_vec = split(path_parts[1], '&');
                    for (const auto& param : query_params_vec) {
                        auto kv = split(param, '=');
                        if (kv.size() == 2) {
                            request.query_params[kv[0]] = kv[1];
                        }
                    }
                }
            }
            
            // Parse headers and body
            // ...
        }
        
        // Find handler
        auto handler = router_.find_handler(request.method, request.path);
        
        HttpResponse response;
        if (handler) {
            response = handler.value()(request);
        } else {
            response = HttpResponse::error(HttpStatus::NOT_FOUND, 
                                          "Endpoint not found");
        }
        
        // Send response
        std::string response_str = "HTTP/1.1 " + 
            std::to_string(static_cast<int>(response.status)) + " OK\r\n";
        response_str += "Content-Type: application/json\r\n";
        response_str += "Content-Length: " + 
            std::to_string(response.body.length()) + "\r\n";
        response_str += "Connection: close\r\n";
        response_str += "\r\n";
        response_str += response.body;
        
        write(client_socket, response_str.c_str(), response_str.length());
    }
    
    close(client_socket);
    active_connections_--;
    total_requests_++;
}

void AdminServer::process_request(const HttpRequest& request) {
    // Route to appropriate handler
}

void AdminServer::register_routes() {
    // Wire the 12 admin domain handlers (futures, options, copy-trading,
    // convert, onramp, offramp, p2p-clients, partners, rewards, marketing,
    // roles, p2p-merchants) against the admin/go backend on port 9093.
    register_domain_routes(router_);
}

void AdminServer::register_middleware() {
    // Will be populated by AdminHandler
}

} // namespace admin
} // namespace tiger
