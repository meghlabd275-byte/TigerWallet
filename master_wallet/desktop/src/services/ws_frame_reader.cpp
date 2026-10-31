#include "ws_frame_reader.hpp"

#include <cerrno>
#include <sys/socket.h>
#include <sys/types.h>

namespace tiger {
namespace master {
namespace websocket {

namespace {

// Pull at least `want` bytes into `accum`. Returns false when the peer closed
// or the socket failed; in that case `accum` holds whatever did arrive.
bool fillTo(std::string& accum, size_t want, int fd) {
    while (accum.size() < want) {
        if (fd < 0) return false;
        char tmp[4096];
        ssize_t n = recv(fd, tmp, sizeof(tmp), 0);
        if (n <= 0) {
            if (n < 0 && errno == EINTR) continue;
            return false;
        }
        accum.append(tmp, static_cast<size_t>(n));
    }
    return true;
}

WsFrameResult fail(const std::string& message) {
    WsFrameResult r;
    r.status = WsFrameStatus::Error;
    r.error = message;
    return r;
}

}  // namespace

WsFrameResult readWsFrame(std::string& accum, int fd, uint64_t maxPayload) {
    WsFrameResult result;

    if (!fillTo(accum, 2, fd)) {
        result.status = WsFrameStatus::Closed;
        return result;
    }

    const uint8_t b0 = static_cast<uint8_t>(accum[0]);
    const uint8_t b1 = static_cast<uint8_t>(accum[1]);
    result.fin = (b0 & 0x80) != 0;
    result.opcode = b0 & 0x0F;
    const bool masked = (b1 & 0x80) != 0;
    uint64_t payloadLen = b1 & 0x7F;
    size_t headerLen = 2;

    if (masked) {
        return fail("protocol error: server frame must not be masked");
    }

    if (payloadLen == 126) {
        if (!fillTo(accum, headerLen + 2, fd)) {
            result.status = WsFrameStatus::Closed;
            return result;
        }
        payloadLen = (static_cast<uint64_t>(static_cast<uint8_t>(accum[2])) << 8) |
                     static_cast<uint8_t>(accum[3]);
        headerLen += 2;
    } else if (payloadLen == 127) {
        if (!fillTo(accum, headerLen + 8, fd)) {
            result.status = WsFrameStatus::Closed;
            return result;
        }
        payloadLen = 0;
        for (int i = 0; i < 8; ++i) {
            payloadLen = (payloadLen << 8) | static_cast<uint8_t>(accum[2 + i]);
        }
        headerLen += 8;
    }

    if (payloadLen > maxPayload) {
        return fail("WebSocket frame too large");
    }

    // payloadLen is bounded by maxPayload here, so the addition cannot wrap.
    if (!fillTo(accum, headerLen + static_cast<size_t>(payloadLen), fd)) {
        result.status = WsFrameStatus::Closed;
        return result;
    }

    result.payload = accum.substr(headerLen, static_cast<size_t>(payloadLen));
    accum.erase(0, headerLen + static_cast<size_t>(payloadLen));
    return result;
}

std::string buildPongFrame(const std::string& payload) {
    std::string frame;
    frame.push_back(static_cast<char>(0x80 | OPCODE_PONG));  // FIN, unmasked

    // RFC 6455 section 5.2 allows a 7-, 16- or 64-bit length. Writing only the
    // low 7 bits truncated the payload for anything over 125 bytes.
    const uint64_t len = payload.size();
    if (len <= 125) {
        frame.push_back(static_cast<char>(len));
    } else if (len <= 0xFFFF) {
        frame.push_back(static_cast<char>(126));
        frame.push_back(static_cast<char>((len >> 8) & 0xFF));
        frame.push_back(static_cast<char>(len & 0xFF));
    } else {
        frame.push_back(static_cast<char>(127));
        for (int shift = 56; shift >= 0; shift -= 8) {
            frame.push_back(static_cast<char>((len >> shift) & 0xFF));
        }
    }

    frame.append(payload);
    return frame;
}

}  // namespace websocket
}  // namespace master
}  // namespace tiger
