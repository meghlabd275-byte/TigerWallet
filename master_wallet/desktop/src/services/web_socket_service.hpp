/**
 * WebSocketService - C++ Implementation
 * Real-time communication for Master Wallet Desktop
 * Features: WebSocket server/client, Event broadcasting, Heartbeat
 * Ultra-low latency with epoll/kqueue
 */

#ifndef WEBSOCKET_SERVICE_HPP
#define WEBSOCKET_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <set>
#include <functional>
#include <memory>
#include <thread>
#include <mutex>
#include <atomic>
#include <queue>
#include <optional>
#include <chrono>

namespace tiger {
namespace master {
namespace websocket {

// Types
using MessageHandler = std::function<void(const std::string& message)>;
using ConnectionHandler = std::function<void(const std::string& clientId)>;
using Message = std::string;

enum class MessageType {
    TEXT,
    BINARY,
    PING,
    PONG,
    CLOSE
};

enum class ConnectionState {
    CONNECTING,
    OPEN,
    CLOSING,
    CLOSED,
    ERROR
};

struct ClientInfo {
    std::string id;
    std::string address;
    ConnectionState state;
    uint64_t connectedAt;
    uint64_t lastActivity;
    uint64_t messageCount;
};

struct WebSocketMessage {
    std::string clientId;
    MessageType type;
    std::string data;
    uint64_t timestamp;
};

class WebSocketService {
public:
    static WebSocketService& getInstance();
    
    // Server operations
    bool startServer(uint16_t port);
    void stopServer();
    bool isServerRunning() const;
    
    // Client operations
    std::string connect(const std::string& url);
    void disconnect(const std::string& clientId);
    bool sendMessage(const std::string& clientId, const std::string& message);
    bool sendBinary(const std::string& clientId, const std::vector<uint8_t>& data);
    
    // Broadcasting
    void broadcast(const std::string& message);
    void broadcastTo(const std::vector<std::string>& clientIds, const std::string& message);
    
    // Event handlers
    void onMessage(MessageHandler handler);
    void onConnect(ConnectionHandler handler);
    void onDisconnect(ConnectionHandler handler);
    void onError(std::function<void(const std::string& error)> handler);
    
    // Client management
    std::vector<ClientInfo> getConnectedClients() const;
    std::optional<ClientInfo> getClient(const std::string& clientId) const;
    
    // Statistics
    size_t getConnectedCount() const;
    size_t getTotalMessagesSent() const;
    size_t getTotalMessagesReceived() const;
    
    // Configuration
    void setHeartbeatInterval(uint32_t intervalMs);
    void setMaxMessageSize(size_t size);
    void setReconnectAttempts(int attempts);

private:
    WebSocketService();
    ~WebSocketService();
    WebSocketService(const WebSocketService&) = delete;
    WebSocketService& operator=(const WebSocketService&) = delete;
    
    // Internal methods
    void acceptConnections();
    void handleClient(const std::string& clientId);
    void sendPing(const std::string& clientId);
    void processMessage(const std::string& clientId, const std::string& data);
    
    // Message queue
    void enqueueMessage(const WebSocketMessage& message);
    std::optional<WebSocketMessage> dequeueMessage();
    
    // Event handlers
    MessageHandler messageHandler_;
    ConnectionHandler connectHandler_;
    ConnectionHandler disconnectHandler_;
    std::function<void(const std::string& error)> errorHandler_;
    
    // Server state
    std::atomic<bool> serverRunning_{false};
    uint16_t serverPort_ = 0;
    std::thread acceptThread_;
    std::map<std::string, std::thread> clientThreads_;
    
    // Client state
    std::map<std::string, ClientInfo> clients_;
    mutable std::mutex clientsMutex_;
    
    // Message queues
    std::queue<WebSocketMessage> messageQueue_;
    mutable std::mutex queueMutex_;
    std::condition_variable queueCondVar_;
    
    // Statistics
    std::atomic<size_t> totalMessagesSent_{0};
    std::atomic<size_t> totalMessagesReceived_{0};
    
    // Configuration
    uint32_t heartbeatIntervalMs_ = 30000; // 30 seconds
    size_t maxMessageSize_ = 1024 * 1024; // 1MB
    int reconnectAttempts_ = 3;
    
    // Thread safety
    mutable std::mutex mutex_;
};

// Inline implementation
inline WebSocketService& WebSocketService::getInstance() {
    static WebSocketService instance;
    return instance;
}

} // namespace websocket
} // namespace master
} // namespace tiger

#endif // WEBSOCKET_SERVICE_HPP
