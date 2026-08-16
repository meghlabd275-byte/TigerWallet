/**
 * TigerAdmin C++ Core - WebSocket Header
 */
#pragma once

#include "admin_security.hpp"

#include <string>
#include <vector>
#include <map>
#include <algorithm>
#include <mutex>
#include <atomic>
#include <cstdint>

namespace tiger {
namespace admin {

enum class EventType {
    TRANSACTION_UPDATE = 0,
    WITHDRAWAL_UPDATE = 1,
    KYC_UPDATE = 2,
    USER_UPDATE = 3,
    TICKET_UPDATE = 4,
    SYSTEM_ALERT = 5,
    PRICE_UPDATE = 6,
    ORDER_UPDATE = 7
};

class WebSocketService {
public:
    struct Client {
        std::string id;
        int socket = -1;
        int64_t connected_at = 0;
        AdminRole role = AdminRole::ADMIN;
        std::vector<std::string> channels;
    };

    static WebSocketService& instance();

    void initialize();
    void shutdown();

    void add_client(const std::string& client_id, int socket);
    void remove_client(const std::string& client_id);
    bool has_client(const std::string& client_id);

    void send_to_client(const std::string& client_id, const std::string& message);
    void send_to_all(const std::string& message);
    void send_to_role(AdminRole role, const std::string& message);

    void subscribe(const std::string& client_id, const std::string& channel);
    void unsubscribe(const std::string& client_id, const std::string& channel);
    void broadcast_to_channel(const std::string& channel, const std::string& message);

    void notify(EventType event, const std::string& data);

    size_t connected_clients() const;
    size_t messages_sent() const;

private:
    std::map<std::string, Client> clients_;
    mutable std::mutex clients_mutex_;
    std::atomic<size_t> messages_sent_{0};
};

struct Notification {
    uint64_t id = 0;
    AdminID admin_id = 0;
    std::string title;
    std::string message;
    std::string notification_type;
    bool is_read = false;
    int64_t created_at = 0;
};

class NotificationService {
public:
    static NotificationService& instance();

    void initialize();

    uint64_t create_notification(AdminID admin_id, const std::string& title,
                                 const std::string& message,
                                 const std::string& notification_type);
    std::vector<Notification> list_notifications(AdminID admin_id, bool unread_only);

    bool mark_as_read(uint64_t notification_id);
    bool mark_all_as_read(AdminID admin_id);
    bool delete_notification(uint64_t notification_id);
    bool delete_all(AdminID admin_id);

    void broadcast(const std::string& title, const std::string& message,
                   const std::string& notification_type);
};

} // namespace admin
} // namespace tiger
