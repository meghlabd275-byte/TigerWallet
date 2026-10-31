/**
 * Tests for the RFC 6455 frame reader.
 *
 * Every case drives readWsFrame with a byte buffer (fd = -1), so the tests
 * exercise the real parser without a socket.
 */

#include "ws_frame_reader.hpp"

#include <cstdint>
#include <iostream>
#include <string>

using namespace tiger::master::websocket;

namespace {

int failures = 0;

void check(bool cond, const std::string& what) {
    if (!cond) {
        std::cerr << "FAIL: " << what << "\n";
        ++failures;
    }
}

std::string frame(uint8_t b0, const std::string& payload) {
    std::string f;
    f.push_back(static_cast<char>(b0));
    const uint64_t len = payload.size();
    if (len <= 125) {
        f.push_back(static_cast<char>(len));
    } else if (len <= 0xFFFF) {
        f.push_back(static_cast<char>(126));
        f.push_back(static_cast<char>((len >> 8) & 0xFF));
        f.push_back(static_cast<char>(len & 0xFF));
    } else {
        f.push_back(static_cast<char>(127));
        for (int shift = 56; shift >= 0; shift -= 8) {
            f.push_back(static_cast<char>((len >> shift) & 0xFF));
        }
    }
    f.append(payload);
    return f;
}

const uint64_t kMax = 1024 * 1024;

// --- Short frames ----------------------------------------------------------

void testSmallTextFrame() {
    std::string accum = frame(0x80 | OPCODE_TEXT, "hello");
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Ok, "small frame parses");
    check(r.fin, "small frame is final");
    check(r.opcode == OPCODE_TEXT, "small frame opcode");
    check(r.payload == "hello", "small frame payload");
    check(accum.empty(), "buffer drained");
}

void testEmptyPayloadFrame() {
    std::string accum = frame(0x80 | OPCODE_TEXT, "");
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Ok, "empty frame parses");
    check(r.payload.empty(), "empty payload");
}

// --- Extended lengths ------------------------------------------------------

void testSixteenBitLength() {
    // 126..65535 uses the 16-bit form. 300 bytes crosses the 125 boundary.
    const std::string body(300, 'x');
    std::string accum = frame(0x80 | OPCODE_BINARY, body);
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Ok, "16-bit length parses");
    check(r.payload.size() == 300, "16-bit length payload size");
    check(r.payload == body, "16-bit length payload intact");
}

void testSixteenBitBoundary() {
    // Exactly 126 bytes must use the 16-bit form; the reader has to follow the
    // declared width rather than assuming the 7-bit one.
    const std::string body(126, 'y');
    std::string accum = frame(0x80 | OPCODE_BINARY, body);
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Ok, "126-byte frame parses");
    check(r.payload == body, "126-byte payload intact");
}

void testSixtyFourBitLength() {
    const std::string body(70000, 'z');  // forces the 64-bit form
    std::string accum = frame(0x80 | OPCODE_BINARY, body);
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Ok, "64-bit length parses");
    check(r.payload.size() == 70000, "64-bit length payload size");
    check(r.payload == body, "64-bit length payload intact");
}

// --- Rejections ------------------------------------------------------------

void testRejectsMaskedServerFrame() {
    // RFC 6455 5.1: servers must not mask. The old parser consumed four payload
    // bytes as a mask key and unmasked with them, silently corrupting the body.
    std::string f;
    f.push_back(static_cast<char>(0x80 | OPCODE_TEXT));
    f.push_back(static_cast<char>(0x80 | 5));  // mask bit set, length 5
    f.append("\x01\x02\x03\x04", 4);           // mask key
    f.append("hello", 5);
    std::string accum = f;
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Error, "masked server frame is rejected");
    check(r.error.find("masked") != std::string::npos, "error names the mask rule");
}

void testRejectsOversizedFrame() {
    const std::string body(2000, 'a');
    std::string accum = frame(0x80 | OPCODE_BINARY, body);
    WsFrameResult r = readWsFrame(accum, -1, 1024);
    check(r.status == WsFrameStatus::Error, "oversized frame is rejected");
    check(r.error.find("too large") != std::string::npos, "error names the size rule");
}

void testClosedOnTruncatedHeader() {
    std::string accum(1, static_cast<char>(0x80));
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Closed, "truncated header reports closed");
}

void testClosedOnTruncatedPayload() {
    std::string f = frame(0x80 | OPCODE_BINARY, "0123456789");
    std::string accum = f.substr(0, f.size() - 3);
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Closed, "truncated payload reports closed");
}

// --- Pipelining and control frames ----------------------------------------

void testConsumesOnlyOneFrame() {
    // Two frames in the buffer: the reader must leave the second for the next
    // call, which is what makes pipelined frames work.
    std::string accum = frame(0x80 | OPCODE_TEXT, "first") +
                        frame(0x80 | OPCODE_TEXT, "second");
    WsFrameResult r1 = readWsFrame(accum, -1, kMax);
    check(r1.payload == "first", "first frame parsed");
    WsFrameResult r2 = readWsFrame(accum, -1, kMax);
    check(r2.payload == "second", "second frame parsed from the same buffer");
    check(accum.empty(), "buffer drained after both frames");
}

void testNonFinalFrame() {
    std::string accum = frame(OPCODE_TEXT, "part");  // FIN clear
    WsFrameResult r = readWsFrame(accum, -1, kMax);
    check(r.status == WsFrameStatus::Ok, "non-final frame parses");
    check(!r.fin, "FIN is reported as clear");
    check(r.opcode == OPCODE_TEXT, "opcode preserved");
}

void testPingAndCloseOpcodes() {
    std::string ping = frame(0x80 | OPCODE_PING, "hb");
    WsFrameResult r = readWsFrame(ping, -1, kMax);
    check(r.opcode == OPCODE_PING, "ping opcode");
    check(r.payload == "hb", "ping payload");

    std::string cls = frame(0x80 | OPCODE_CLOSE, "");
    WsFrameResult rc = readWsFrame(cls, -1, kMax);
    check(rc.opcode == OPCODE_CLOSE, "close opcode");
}

// --- Pong construction -----------------------------------------------------

void testPongEchoesShortPayload() {
    std::string p = buildPongFrame("hi");
    check(static_cast<uint8_t>(p[0]) == (0x80 | OPCODE_PONG), "pong opcode byte");
    check(static_cast<uint8_t>(p[1]) == 2, "pong length byte");
    check(p.substr(2) == "hi", "pong echoes the payload");
    check((static_cast<uint8_t>(p[1]) & 0x80) == 0, "pong is not masked");
}

void testPongUsesSixteenBitLength() {
    // 200 bytes needs the 16-bit form. The old code wrote `size() & 0x7F`,
    // which is 72 for 200, so the peer received a truncated control frame.
    const std::string body(200, 'p');
    std::string p = buildPongFrame(body);
    check(static_cast<uint8_t>(p[1]) == 126, "pong uses the 16-bit length marker");
    const uint64_t declared = (static_cast<uint64_t>(static_cast<uint8_t>(p[2])) << 8) |
                              static_cast<uint8_t>(p[3]);
    check(declared == 200, "pong declares the real length");
    check(p.substr(4) == body, "pong echoes the full payload");
}

void testPongUsesSixtyFourBitLength() {
    const std::string body(70000, 'q');
    std::string p = buildPongFrame(body);
    check(static_cast<uint8_t>(p[1]) == 127, "pong uses the 64-bit length marker");
    uint64_t declared = 0;
    for (int i = 0; i < 8; ++i) {
        declared = (declared << 8) | static_cast<uint8_t>(p[2 + i]);
    }
    check(declared == 70000, "pong declares the real 64-bit length");
    check(p.substr(10) == body, "pong echoes the full payload");
}

void testPongBoundary125() {
    const std::string body(125, 'r');
    std::string p = buildPongFrame(body);
    check(static_cast<uint8_t>(p[1]) == 125, "125 bytes fits the 7-bit form");
    check(p.size() == 2 + 125, "pong frame size");
}

void testPongBoundary126() {
    const std::string body(126, 's');
    std::string p = buildPongFrame(body);
    check(static_cast<uint8_t>(p[1]) == 126, "126 bytes needs the 16-bit form");
    check(p.size() == 4 + 126, "pong frame size");
}

}  // namespace

int main() {
    testSmallTextFrame();
    testEmptyPayloadFrame();
    testSixteenBitLength();
    testSixteenBitBoundary();
    testSixtyFourBitLength();
    testRejectsMaskedServerFrame();
    testRejectsOversizedFrame();
    testClosedOnTruncatedHeader();
    testClosedOnTruncatedPayload();
    testConsumesOnlyOneFrame();
    testNonFinalFrame();
    testPingAndCloseOpcodes();
    testPongEchoesShortPayload();
    testPongUsesSixteenBitLength();
    testPongUsesSixtyFourBitLength();
    testPongBoundary125();
    testPongBoundary126();

    if (failures == 0) {
        std::cout << "all ws_frame_reader tests passed\n";
        return 0;
    }
    std::cerr << failures << " check(s) failed\n";
    return 1;
}
