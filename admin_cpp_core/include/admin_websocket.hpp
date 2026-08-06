/**
 * TigerAdmin C++ Core - WebSocket Handler
 */

#ifndef TIGER_ADMIN_WEBSOCKET_HPP
#define TIGER_ADMIN_WEBSOCKET_HPP

#include <string>
#include <vector>
#include <map>
#include <functional>
#include <memory>
#include <mutex>
#include <atomic>
#include <thread>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// WebSocket Service
// ============================================================================

class WebSocketService {
public:
    static WebSocketService& instance();
    
    void initialize();
    void shutdown();
    
    // Client management
    void add_client(const std::string& client_id, int socket);
    void remove_client(const std::string& client_id);
    bool has_client(const std::string& client_id);
    
    // Send messages
    void send_to_client(const std::string& client_id, const std::string& message);
    void send_to_all(const std::string& message);
    void send_to_role(AdminRole role, const std::string& message);
    
    // Subscribe/Unsubscribe
    void subscribe(const std::string& client_id, const std::string& channel);
    void unsubscribe(const std::string& client_id, const std::string& channel);
    
    // Broadcast channels
    void broadcast_to_channel(const std::string& channel, 
                              const std::string& message);
    
    // Event types
    enum class EventType {
        USER_CREATED,
        USER_UPDATED,
        USER_BANNED,
        KYC_APPROVED,
        KYC_REJECTED,
        TRANSACTION_FLAGGED,
        WITHDRAWAL_PENDING,
        WITHDRAWAL_COMPLETED,
        TICKET_CREATED,
        TICKET_ASSIGNED,
        TICKET_RESOLVED,
        SYSTEM_ALERT,
        PRICE_ALERT
    };
    
    // Event notification
    void notify(EventType event, const std::string& data);
    
    // Stats
    size_t connected_clients() const;
    size_t messages_sent() const;
    
private:
    WebSocketService() = default;
    ~WebSocketService() = default;
    
    struct Client {
        std::string id;
        int socket;
        AdminID admin_id = 0;
        AdminRole role;
        std::vector<std::string> channels;
    };
    
    std::mutex clients_mutex_;
    std::map<std::string, Client> clients_;
    std::atomic<size_t> messages_sent_{0};
    
    void process_message(const std::string& client_id, 
                        const std::string& message);
};

// ============================================================================
// Notification Service
// ============================================================================

class NotificationService {
public:
    static NotificationService& instance();
    
    void initialize();
    
    // Create notification
    uint64_t create_notification(AdminID admin_id,
                                const std::string& title,
                                const std::string& message,
                                const std::string& notification_type);
    
    // List notifications
    std::vector<Notification> list_notifications(AdminID admin_id,
                                                  bool unread_only = false);
    
    // Mark as read
    bool mark_as_read(uint64_t notification_id);
    bool mark_all_as_read(AdminID admin_id);
    
    // Delete
    bool delete_notification(uint64_t notification_id);
    bool delete_all(AdminID admin_id);
    
    // Broadcast
    void broadcast(const std::string& title,
                  const std::string& message,
                  const std::string& notification_type);
    
private:
    NotificationService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_WEBSOCKET_HPP
