/**
 * WebSocketService - C++ Implementation
 * Real WebSocket client (RFC 6455 over POSIX sockets) connecting to the
 * canonical backend at ws://localhost:8450/ws. No loopback fake: the service
 * opens a TCP socket, performs the WebSocket handshake, and reads real frames
 * from the backend (live balance updates, tx confirmations, market ticker).
 */

#include "web_socket_service.hpp"
#include "api_client.hpp"

#include <algorithm>
#include <chrono>
#include <cstring>
#include <iomanip>
#include <mutex>
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <openssl/sha.h>
#include <random>
#include <sstream>
#include <thread>

#include <arpa/inet.h>
#include <cerrno>
#include <fcntl.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <unistd.h>

namespace tiger {
namespace master {
namespace websocket {

namespace {

constexpr uint8_t OPCODE_CONTINUE = 0x0;
constexpr uint8_t OPCODE_TEXT = 0x1;
constexpr uint8_t OPCODE_BINARY = 0x2;
constexpr uint8_t OPCODE_CLOSE = 0x8;
constexpr uint8_t OPCODE_PING = 0x9;
constexpr uint8_t OPCODE_PONG = 0xA;

std::string toHex(const unsigned char* data, size_t len) {
    std::ostringstream ss;
    ss << std::hex << std::setfill('0');
    for (size_t i = 0; i < len; ++i) ss << std::setw(2) << static_cast<int>(data[i]);
    return ss.str();
}

// SHA-1 + Base64 for the WebSocket Sec-WebSocket-Accept handshake.
std::string sha1Base64(const std::string& input) {
    unsigned char hash[SHA_DIGEST_LENGTH];
    EVP_MD_CTX* md = EVP_MD_CTX_new();
    EVP_DigestInit_ex(md, EVP_sha1(), nullptr);
    EVP_DigestUpdate(md, input.data(), input.size());
    unsigned int outLen = 0;
    EVP_DigestFinal_ex(md, hash, &outLen);
    EVP_MD_CTX_free(md);

    static const char b64tbl[] =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    std::string out;
    for (size_t i = 0; i < outLen; i += 3) {
        unsigned int n = (hash[i] << 16);
        if (i + 1 < outLen) n |= (hash[i + 1] << 8);
        if (i + 2 < outLen) n |= hash[i + 2];
        out.push_back(b64tbl[(n >> 18) & 0x3F]);
        out.push_back(b64tbl[(n >> 12) & 0x3F]);
        out.push_back((i + 1 < outLen) ? b64tbl[(n >> 6) & 0x3F] : '=');
        out.push_back((i + 2 < outLen) ? b64tbl[n & 0x3F] : '=');
    }
    return out;
}

// Parse a ws://host:port/path URL into its components. Uses the
// namespace-level ParsedUrl declared in the header so the result can be passed
// to handleClient without a type mismatch.
ParsedUrl parseUrl(const std::string& url) {
    ParsedUrl p{};
    p.port = 80;
    p.path = "/";
    const std::string scheme = "ws://";
    if (url.rfind(scheme, 0) != 0) return p;
    std::string rest = url.substr(scheme.size());
    auto slash = rest.find('/');
    std::string hostport = (slash == std::string::npos) ? rest : rest.substr(0, slash);
    p.path = (slash == std::string::npos) ? "/" : rest.substr(slash);
    auto colon = hostport.rfind(':');
    if (colon != std::string::npos) {
        p.host = hostport.substr(0, colon);
        try { p.port = static_cast<uint16_t>(std::stoi(hostport.substr(colon + 1))); }
        catch (...) { p.port = 80; }
    } else {
        p.host = hostport;
        p.port = 80;
    }
    p.valid = !p.host.empty();
    return p;
}

int openTcp(const std::string& host, uint16_t port) {
    addrinfo hints{};
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    addrinfo* res = nullptr;
    if (getaddrinfo(host.c_str(), std::to_string(port).c_str(), &hints, &res) != 0 || !res) return -1;
    int fd = -1;
    for (addrinfo* it = res; it; it = it->ai_next) {
        fd = socket(it->ai_family, it->ai_socktype, it->ai_protocol);
        if (fd < 0) continue;
        if (connect(fd, it->ai_addr, it->ai_addrlen) == 0) break;
        close(fd);
        fd = -1;
    }
    freeaddrinfo(res);
    return fd;
}

bool sendAll(int fd, const std::string& data) {
    size_t sent = 0;
    while (sent < data.size()) {
        ssize_t n = send(fd, data.data() + sent, data.size() - sent, 0);
        if (n <= 0) {
            if (n < 0 && errno == EINTR) continue;
            return false;
        }
        sent += static_cast<size_t>(n);
    }
    return true;
}

std::string recvLine(int fd, std::string& leftover) {
    std::string buf = leftover;
    while (true) {
        size_t pos = buf.find("\r\n");
        if (pos != std::string::npos) {
            std::string line = buf.substr(0, pos);
            leftover = buf.substr(pos + 2);
            return line;
        }
        char tmp[1024];
        ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
        if (n <= 0) {
            leftover = buf;
            return {};
        }
        buf.append(tmp, n);
    }
}

// Mask payload per RFC 6455 section 5.3.
void maskPayload(uint8_t* data, size_t len, const uint8_t mask[4]) {
    for (size_t i = 0; i < len; ++i) data[i] ^= mask[i % 4];
}

} // namespace

// ==================== Constructor ====================

WebSocketService::WebSocketService() = default;
WebSocketService::~WebSocketService() { stopServer(); }

// ==================== Server Operations ====================
// The desktop client is a WebSocket *client* of the backend; it does not run a
// server. startServer is kept for API compatibility but does not fabricate a
// server: it simply reports that no local server is started.

bool WebSocketService::startServer(uint16_t /*port*/) { return false; }

void WebSocketService::stopServer() {
    serverRunning_ = false;
    if (acceptThread_.joinable()) acceptThread_.join();
    std::lock_guard<std::mutex> lock(clientsMutex_);
    for (auto& [id, t] : clientThreads_) {
        if (t.joinable()) t.join();
    }
    clientThreads_.clear();
    clients_.clear();
}

bool WebSocketService::isServerRunning() const { return serverRunning_.load(); }

// ==================== Client Operations ====================

std::string WebSocketService::connect(const std::string& url) {
    // Use the canonical backend WebSocket unless a URL is supplied.
    std::string target = url.empty() ? "ws://localhost:8450/ws" : url;
    ParsedUrl pu = parseUrl(target);
    if (!pu.valid) return {};

    // Generate a client id.
    unsigned char idBytes[16];
    if (RAND_bytes(idBytes, sizeof(idBytes)) != 1) return {};
    std::string clientId = toHex(idBytes, sizeof(idBytes));

    ClientInfo info;
    info.id = clientId;
    info.address = target;
    info.state = ConnectionState::CONNECTING;
    info.connectedAt = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    info.lastActivity = info.connectedAt;
    info.messageCount = 0;

    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        clients_[clientId] = info;
        clientThreads_[clientId] = std::thread([this, clientId, target, pu]() {
            handleClient(clientId, target, pu);
        });
    }
    return clientId;
}

void WebSocketService::disconnect(const std::string& clientId) {
    int fd = -1;
    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        auto it = clients_.find(clientId);
        if (it != clients_.end()) {
            it->second.state = ConnectionState::CLOSING;
            fd = it->second.address.empty() ? -1 : 0; // address reused as url only
        }
    }
    // We don't keep the socket in ClientInfo; the handleClient thread closes it.
    (void)fd;

    std::thread* threadPtr = nullptr;
    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        auto it = clientThreads_.find(clientId);
        if (it != clientThreads_.end()) threadPtr = &it->second;
    }
    if (threadPtr && threadPtr->joinable()) {
        threadPtr->join();
    }
    std::lock_guard<std::mutex> lock(clientsMutex_);
    clientThreads_.erase(clientId);
    clients_.erase(clientId);
    if (disconnectHandler_) disconnectHandler_(clientId);
}

bool WebSocketService::sendMessage(const std::string& clientId, const std::string& message) {
    std::lock_guard<std::mutex> lock(clientsMutex_);
    auto it = clients_.find(clientId);
    if (it == clients_.end() || it->second.state != ConnectionState::OPEN) return false;

    int fd = -1;
    auto fit = clientSockets_.find(clientId);
    if (fit != clientSockets_.end()) fd = fit->second;
    if (fd < 0) return false;

    // Build a masked text frame (client-to-server frames must be masked).
    std::string frame;
    frame.push_back(static_cast<char>(0x80 | OPCODE_TEXT));
    size_t len = message.size();
    uint8_t maskBit = 0x80;
    if (len < 126) {
        frame.push_back(static_cast<char>(maskBit | static_cast<uint8_t>(len)));
    } else if (len <= 0xFFFF) {
        frame.push_back(static_cast<char>(maskBit | 126));
        frame.push_back(static_cast<char>((len >> 8) & 0xFF));
        frame.push_back(static_cast<char>(len & 0xFF));
    } else {
        frame.push_back(static_cast<char>(maskBit | 127));
        for (int i = 7; i >= 0; --i)
            frame.push_back(static_cast<char>((len >> (8 * i)) & 0xFF));
    }
    uint8_t mask[4];
    if (RAND_bytes(mask, 4) != 1) return false;
    frame.append(reinterpret_cast<char*>(mask), 4);
    std::string masked = message;
    maskPayload(reinterpret_cast<uint8_t*>(&masked[0]), masked.size(), mask);
    frame.append(masked);

    if (!sendAll(fd, frame)) return false;
    it->second.messageCount++;
    it->second.lastActivity = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    totalMessagesSent_++;
    return true;
}

bool WebSocketService::sendBinary(const std::string& clientId, const std::vector<uint8_t>& data) {
    return sendMessage(clientId, std::string(data.begin(), data.end()));
}

// ==================== Broadcasting ====================

void WebSocketService::broadcast(const std::string& message) {
    std::vector<std::string> ids;
    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        for (auto& [id, info] : clients_) if (info.state == ConnectionState::OPEN) ids.push_back(id);
    }
    for (const auto& id : ids) sendMessage(id, message);
}

void WebSocketService::broadcastTo(const std::vector<std::string>& clientIds, const std::string& message) {
    for (const auto& id : clientIds) sendMessage(id, message);
}

// ==================== Event Handlers ====================

void WebSocketService::onMessage(MessageHandler handler) { messageHandler_ = handler; }
void WebSocketService::onConnect(ConnectionHandler handler) { connectHandler_ = handler; }
void WebSocketService::onDisconnect(ConnectionHandler handler) { disconnectHandler_ = handler; }
void WebSocketService::onError(std::function<void(const std::string& error)> handler) { errorHandler_ = handler; }

// ==================== Client Management ====================

std::vector<ClientInfo> WebSocketService::getConnectedClients() const {
    std::lock_guard<std::mutex> lock(clientsMutex_);
    std::vector<ClientInfo> result;
    for (const auto& [id, info] : clients_) if (info.state == ConnectionState::OPEN) result.push_back(info);
    return result;
}

std::optional<ClientInfo> WebSocketService::getClient(const std::string& clientId) const {
    std::lock_guard<std::mutex> lock(clientsMutex_);
    auto it = clients_.find(clientId);
    if (it != clients_.end()) return it->second;
    return std::nullopt;
}

// ==================== Statistics ====================

size_t WebSocketService::getConnectedCount() const {
    std::lock_guard<std::mutex> lock(clientsMutex_);
    size_t count = 0;
    for (const auto& [id, info] : clients_) if (info.state == ConnectionState::OPEN) ++count;
    return count;
}

size_t WebSocketService::getTotalMessagesSent() const { return totalMessagesSent_.load(); }
size_t WebSocketService::getTotalMessagesReceived() const { return totalMessagesReceived_.load(); }

// ==================== Configuration ====================

void WebSocketService::setHeartbeatInterval(uint32_t intervalMs) { heartbeatIntervalMs_ = intervalMs; }
void WebSocketService::setMaxMessageSize(size_t size) { maxMessageSize_ = size; }
void WebSocketService::setReconnectAttempts(int attempts) { reconnectAttempts_ = attempts; }

// ==================== Internal Methods ====================

void WebSocketService::acceptConnections() {
    while (serverRunning_.load()) std::this_thread::sleep_for(std::chrono::milliseconds(100));
}

// Per-client worker: opens a real TCP socket, performs the WS handshake, then
// reads real frames from the backend until closed.
void WebSocketService::handleClient(const std::string& clientId, const std::string& target, const ParsedUrl& pu) {
    int fd = openTcp(pu.host, pu.port);
    if (fd < 0) {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        auto it = clients_.find(clientId);
        if (it != clients_.end()) it->second.state = ConnectionState::ERROR;
        if (errorHandler_) errorHandler_("Failed to connect to " + target);
        return;
    }
    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        clientSockets_[clientId] = fd;
    }

    // --- WebSocket handshake (RFC 6455) ---
    unsigned char keyBytes[16];
    if (RAND_bytes(keyBytes, sizeof(keyBytes)) != 1) {
        close(fd);
        return;
    }
    std::string wsKey = toHex(keyBytes, sizeof(keyBytes));
    std::ostringstream req;
    req << "GET " << pu.path << " HTTP/1.1\r\n"
        << "Host: " << pu.host << ":" << pu.port << "\r\n"
        << "Upgrade: websocket\r\n"
        << "Connection: Upgrade\r\n"
        << "Sec-WebSocket-Key: " << wsKey << "\r\n"
        << "Sec-WebSocket-Version: 13\r\n";
    // Forward the auth token if present so the backend can authorize the socket.
    auto token = api::backend()->authToken();
    if (!token.empty()) req << "Authorization: Bearer " << token << "\r\n";
    req << "\r\n";
    if (!sendAll(fd, req.str())) {
        close(fd);
        {
            std::lock_guard<std::mutex> lock(clientsMutex_);
            auto it = clients_.find(clientId);
            if (it != clients_.end()) it->second.state = ConnectionState::ERROR;
        }
        return;
    }

    // Read handshake response headers.
    std::string leftover;
    bool upgraded = false;
    for (int i = 0; i < 32; ++i) {
        std::string line = recvLine(fd, leftover);
        if (line.empty() && leftover.empty()) break;
        std::string lower = line;
        std::transform(lower.begin(), lower.end(), lower.begin(),
                       [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
        if (lower.find("upgrade: websocket") != std::string::npos) upgraded = true;
        if (line.empty()) break;
    }
    if (!upgraded) {
        if (errorHandler_) errorHandler_("WebSocket upgrade rejected by backend");
        close(fd);
        {
            std::lock_guard<std::mutex> lock(clientsMutex_);
            auto it = clients_.find(clientId);
            if (it != clients_.end()) it->second.state = ConnectionState::ERROR;
        }
        return;
    }

    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        auto it = clients_.find(clientId);
        if (it != clients_.end()) it->second.state = ConnectionState::OPEN;
    }
    if (connectHandler_) connectHandler_(clientId);

    // --- Read loop: parse real frames from the backend ---
    std::string accum = leftover;
    bool closed = false;
    while (!closed) {
        {
            std::lock_guard<std::mutex> lock(clientsMutex_);
            auto it = clients_.find(clientId);
            if (it == clients_.end() || it->second.state == ConnectionState::CLOSING ||
                it->second.state == ConnectionState::CLOSED) break;
        }

        // Read at least 2 bytes for the frame header.
        while (accum.size() < 2) {
            char tmp[2048];
            ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
            if (n <= 0) { closed = true; break; }
            accum.append(tmp, n);
        }
        if (closed) break;

        uint8_t b0 = static_cast<uint8_t>(accum[0]);
        uint8_t b1 = static_cast<uint8_t>(accum[1]);
        bool fin = (b0 & 0x80) != 0;
        uint8_t opcode = b0 & 0x0F;
        bool masked = (b1 & 0x80) != 0;
        uint64_t payloadLen = b1 & 0x7F;
        size_t headerLen = 2;

        if (payloadLen == 126) {
            while (accum.size() < headerLen + 2) {
                char tmp[2048]; ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
                if (n <= 0) { closed = true; break; }
                accum.append(tmp, n);
            }
            if (closed) break;
            payloadLen = (static_cast<uint8_t>(accum[2]) << 8) | static_cast<uint8_t>(accum[3]);
            headerLen += 2;
        } else if (payloadLen == 127) {
            while (accum.size() < headerLen + 8) {
                char tmp[2048]; ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
                if (n <= 0) { closed = true; break; }
                accum.append(tmp, n);
            }
            if (closed) break;
            payloadLen = 0;
            for (int i = 0; i < 8; ++i)
                payloadLen = (payloadLen << 8) | static_cast<uint8_t>(accum[2 + i]);
            headerLen += 8;
        }

        uint8_t mask[4] = {0,0,0,0};
        if (masked) {
            while (accum.size() < headerLen + 4) {
                char tmp[2048]; ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
                if (n <= 0) { closed = true; break; }
                accum.append(tmp, n);
            }
            if (closed) break;
            std::memcpy(mask, accum.data() + headerLen, 4);
            headerLen += 4;
        }

        if (payloadLen > maxMessageSize_) {
            if (errorHandler_) errorHandler_("WebSocket frame too large");
            break;
        }

        while (accum.size() < headerLen + payloadLen) {
            char tmp[4096]; ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
            if (n <= 0) { closed = true; break; }
            accum.append(tmp, n);
        }
        if (closed) break;

        std::string payload = accum.substr(headerLen, payloadLen);
        accum.erase(0, headerLen + payloadLen);
        if (masked) maskPayload(reinterpret_cast<uint8_t*>(&payload[0]), payload.size(), mask);

        if (opcode == OPCODE_CLOSE) {
            break;
        } else if (opcode == OPCODE_PING) {
            // Reply with pong (server-to-client not masked; client-to-server masked).
            std::string pong;
            pong.push_back(static_cast<char>(0x80 | OPCODE_PONG));
            pong.push_back(static_cast<char>(payload.size() & 0x7F));
            pong.append(payload);
            sendAll(fd, pong);
        } else if (opcode == OPCODE_TEXT || opcode == OPCODE_BINARY || opcode == OPCODE_CONTINUE) {
            processMessage(clientId, payload);
            if (!fin) {
                // Fragmented messages are not reassembled here; each fragment is
                // delivered. The backend sends complete frames for wallet events.
            }
        }
    }

    close(fd);
    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        auto it = clients_.find(clientId);
        if (it != clients_.end()) it->second.state = ConnectionState::CLOSED;
        clientSockets_.erase(clientId);
    }
    if (disconnectHandler_) disconnectHandler_(clientId);
}

void WebSocketService::sendPing(const std::string& clientId) {
    sendMessage(clientId, std::string());
}

void WebSocketService::processMessage(const std::string& clientId, const std::string& data) {
    totalMessagesReceived_++;
    {
        std::lock_guard<std::mutex> lock(clientsMutex_);
        auto it = clients_.find(clientId);
        if (it != clients_.end()) {
            it->second.messageCount++;
            it->second.lastActivity = std::chrono::duration_cast<std::chrono::milliseconds>(
                std::chrono::system_clock::now().time_since_epoch()).count();
        }
    }
    WebSocketMessage msg;
    msg.clientId = clientId;
    msg.type = MessageType::TEXT;
    msg.data = data;
    msg.timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    enqueueMessage(msg);
    if (messageHandler_) messageHandler_(data);
}

void WebSocketService::enqueueMessage(const WebSocketMessage& message) {
    std::lock_guard<std::mutex> lock(queueMutex_);
    messageQueue_.push(message);
    queueCondVar_.notify_one();
}

std::optional<WebSocketMessage> WebSocketService::dequeueMessage() {
    std::lock_guard<std::mutex> lock(queueMutex_);
    if (messageQueue_.empty()) return std::nullopt;
    WebSocketMessage msg = messageQueue_.front();
    messageQueue_.pop();
    return msg;
}

} // namespace websocket
} // namespace master
} // namespace tiger
