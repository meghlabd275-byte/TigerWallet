/**
 * RFC 6455 frame reader, split out of web_socket_service.cpp so the framing
 * rules can be exercised directly by tests.
 *
 * The desktop client only ever reads server-to-client frames, so this module
 * does not need OpenSSL or libcurl and can be compiled and tested on its own.
 */

#ifndef TIGER_MASTER_WEBSOCKET_FRAME_READER_HPP
#define TIGER_MASTER_WEBSOCKET_FRAME_READER_HPP

#include <cstddef>
#include <cstdint>
#include <string>

namespace tiger {
namespace master {
namespace websocket {

// RFC 6455 section 5.2 opcodes.
constexpr uint8_t OPCODE_CONTINUE = 0x0;
constexpr uint8_t OPCODE_TEXT = 0x1;
constexpr uint8_t OPCODE_BINARY = 0x2;
constexpr uint8_t OPCODE_CLOSE = 0x8;
constexpr uint8_t OPCODE_PING = 0x9;
constexpr uint8_t OPCODE_PONG = 0xA;

enum class WsFrameStatus {
    Ok,
    Closed,  // the peer closed the connection, or the socket errored
    Error,   // a protocol violation; `error` explains it
};

struct WsFrameResult {
    WsFrameStatus status = WsFrameStatus::Ok;
    bool fin = false;
    uint8_t opcode = 0;
    std::string payload;
    std::string error;
};

/**
 * Read one frame from `fd`.
 *
 * Bytes already buffered in `accum` are consumed first; anything left over is
 * kept there for the next call. `maxPayload` bounds both the declared frame
 * length and the resulting payload.
 *
 * The reader rejects masked frames: RFC 6455 section 5.1 says a server must not
 * mask what it sends, and honouring a mask bit made the parser swallow four
 * payload bytes as a key and hand the caller a corrupted body.
 *
 * `fd` may be -1 when the frame is already fully present in `accum`, which is
 * how the tests drive it.
 */
WsFrameResult readWsFrame(std::string& accum, int fd, uint64_t maxPayload);

/// Build an unmasked pong frame echoing `payload`.
std::string buildPongFrame(const std::string& payload);

}  // namespace websocket
}  // namespace master
}  // namespace tiger

#endif  // TIGER_MASTER_WEBSOCKET_FRAME_READER_HPP
