/**
 * TigerAdmin C++ Core - WebSocket Implementation
 */

#include "admin_websocket.hpp"
#include "admin_logger.hpp"

namespace tiger {
namespace admin {

WebSocketService& WebSocketService::instance() {
    static WebSocketService service;
    return service;
}

void WebSocketService::initialize() { LOG_INFO("WebSocket service initialized"); }
void WebSocketService::shutdown() { LOG_INFO("WebSocket service shutdown"); }

void WebSocketService::add_client(const std::string& client_id, int socket) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    clients_[client_id] = {client_id, socket, 0, AdminRole::ADMIN, {}};
}

void WebSocketService::remove_client(const std::string& client_id) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    clients_.erase(client_id);
}

bool WebSocketService::has_client(const std::string& client_id) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    return clients_.find(client_id) != clients_.end();
}

void WebSocketService::send_to_client(const std::string& client_id, const std::string& message) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    auto it = clients_.find(client_id);
    if (it != clients_.end()) {
        messages_sent_++;
    }
}

void WebSocketService::send_to_all(const std::string& message) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    for (const auto& client : clients_) {
        messages_sent_++;
    }
}

void WebSocketService::send_to_role(AdminRole role, const std::string& message) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    for (const auto& client : clients_) {
        if (client.second.role == role) {
            messages_sent_++;
        }
    }
}

void WebSocketService::subscribe(const std::string& client_id, const std::string& channel) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    auto it = clients_.find(client_id);
    if (it != clients_.end()) {
        it->second.channels.push_back(channel);
    }
}

void WebSocketService::unsubscribe(const std::string& client_id, const std::string& channel) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    auto it = clients_.find(client_id);
    if (it != clients_.end()) {
        auto& channels = it->second.channels;
        channels.erase(std::remove(channels.begin(), channels.end(), channel), channels.end());
    }
}

void WebSocketService::broadcast_to_channel(const std::string& channel, const std::string& message) {
    std::lock_guard<std::mutex> lock(clients_mutex_);
    for (const auto& client : clients_) {
        for (const auto& c : client.second.channels) {
            if (c == channel) {
                messages_sent_++;
                break;
            }
        }
    }
}

void WebSocketService::notify(EventType event, const std::string& data) {
    send_to_all(data);
}

size_t WebSocketService::connected_clients() const {
    return clients_.size();
}

size_t WebSocketService::messages_sent() const {
    return messages_sent_.load();
}

// Notification Service
NotificationService& NotificationService::instance() {
    static NotificationService service;
    return service;
}

void NotificationService::initialize() { LOG_INFO("Notification service initialized"); }

uint64_t NotificationService::create_notification(AdminID admin_id, const std::string& title,
    const std::string& message, const std::string& notification_type) {
    return 1;
}

std::vector<Notification> NotificationService::list_notifications(AdminID admin_id, bool unread_only) {
    return {};
}

bool NotificationService::mark_as_read(uint64_t notification_id) { return true; }
bool NotificationService::mark_all_as_read(AdminID admin_id) { return true; }
bool NotificationService::delete_notification(uint64_t notification_id) { return true; }
bool NotificationService::delete_all(AdminID admin_id) { return true; }
void NotificationService::broadcast(const std::string& title, const std::string& message,
    const std::string& notification_type) {}

} // namespace admin
} // namespace tiger
