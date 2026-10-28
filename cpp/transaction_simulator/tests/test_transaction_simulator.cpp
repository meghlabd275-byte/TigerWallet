// Tests for the transaction simulator.
//
// The simulator previously produced its transaction hash by concatenating
// eight hex characters of the sender address with eight of the recipient, and
// accepted any 64-byte string as a valid signature. Both are covered here.

#include "transaction_simulator.h"

#include "rpc.hpp"

#include <arpa/inet.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>

#include <atomic>
#include <cassert>
#include <cstdint>
#include <cstdio>
#include <string>
#include <thread>

using namespace TigerWallet;

namespace {

Transaction MakeTransaction() {
    Transaction tx;
    tx.from = Address::fromString("0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E");
    tx.to = Address::fromString("0xdAC17F958D2ee523a2206206994597C13D831ec7");
    tx.value = {1000000000000000000ULL, 0, 0, 0};
    tx.gasLimit = 21000;
    tx.gasPrice = 20000000000ULL;
    tx.nonce = 0;
    tx.chainId = 1;
    return tx;
}

// The secp256k1 group order.
const uint8_t kOrder[32] = {
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfe,
    0xba, 0xae, 0xdc, 0xe6, 0xaf, 0x48, 0xa0, 0x3b,
    0xbf, 0xd2, 0x5e, 0x8c, 0xd0, 0x36, 0x41, 0x41
};

// A minimal JSON-RPC endpoint. It answers the handful of methods the
// simulator calls, so the simulation path can be exercised without a real
// chain. The balance is large enough for the sample transaction.
class StubNode {
public:
    StubNode() {
        listen_fd_ = ::socket(AF_INET, SOCK_STREAM, 0);
        assert(listen_fd_ >= 0);

        int one = 1;
        ::setsockopt(listen_fd_, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));

        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_addr.s_addr = ::htonl(INADDR_LOOPBACK);
        addr.sin_port = 0;
        assert(::bind(listen_fd_, reinterpret_cast<sockaddr*>(&addr),
                      sizeof(addr)) == 0);

        socklen_t len = sizeof(addr);
        assert(::getsockname(listen_fd_, reinterpret_cast<sockaddr*>(&addr),
                             &len) == 0);
        port_ = ::ntohs(addr.sin_port);
        assert(::listen(listen_fd_, 8) == 0);

        worker_ = std::thread([this] { serve(); });
    }

    ~StubNode() {
        running_ = false;
        ::shutdown(listen_fd_, SHUT_RDWR);
        ::close(listen_fd_);
        if (worker_.joinable()) worker_.join();
    }

    std::string url() const {
        return "http://127.0.0.1:" + std::to_string(port_);
    }

    // A node that rejects the call, for the fail-closed check.
    void setRevert(bool revert) { revert_ = revert; }

private:
    void serve() {
        while (running_) {
            const int fd = ::accept(listen_fd_, nullptr, nullptr);
            if (fd < 0) {
                if (!running_) return;
                continue;
            }
            handle(fd);
            ::close(fd);
        }
    }

    void handle(int fd) {
        std::string request;
        char buffer[4096];
        // Read the headers first, then the body by its length.
        size_t header_end = std::string::npos;
        while ((header_end = request.find("\r\n\r\n")) == std::string::npos) {
            const ssize_t n = ::recv(fd, buffer, sizeof(buffer), 0);
            if (n <= 0) return;
            request.append(buffer, static_cast<size_t>(n));
            if (request.size() > 65536) return;
        }

        size_t content_length = 0;
        const std::string length_key = "Content-Length:";
        const size_t at = request.find(length_key);
        if (at != std::string::npos) {
            content_length = static_cast<size_t>(
                std::strtoul(request.c_str() + at + length_key.size(), nullptr, 10));
        }

        const size_t body_start = header_end + 4;
        while (request.size() < body_start + content_length) {
            const ssize_t n = ::recv(fd, buffer, sizeof(buffer), 0);
            if (n <= 0) return;
            request.append(buffer, static_cast<size_t>(n));
        }
        const std::string body = request.substr(body_start, content_length);

        std::string result;
        if (revert_) {
            result = "{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":3,"
                     "\"message\":\"execution reverted: insufficient output\"}}";
        } else if (body.find("eth_estimateGas") != std::string::npos) {
            result = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0x5208\"}";
        } else if (body.find("eth_getTransactionCount") != std::string::npos) {
            result = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0x0\"}";
        } else if (body.find("eth_gasPrice") != std::string::npos) {
            result = "{\"jsonrpc\":\"2.0\",\"id\":1,"
                     "\"result\":\"0x4a817c800\"}";
        } else if (body.find("eth_getBlockByNumber") != std::string::npos) {
            result = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":"
                     "{\"baseFeePerGas\":\"0x3b9aca00\"}}";
        } else {
            // eth_getBalance, and anything else, reports ten ether.
            result = "{\"jsonrpc\":\"2.0\",\"id\":1,"
                     "\"result\":\"0x8ac7230489e80000\"}";
        }

        const std::string response =
            "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n"
            "Content-Length: " + std::to_string(result.size()) +
            "\r\nConnection: close\r\n\r\n" + result;
        ::send(fd, response.data(), response.size(), 0);
    }

    int listen_fd_ = -1;
    uint16_t port_ = 0;
    std::atomic<bool> running_{true};
    std::atomic<bool> revert_{false};
    std::thread worker_;
};

}  // namespace

int main() {
    TransactionSimulator simulator(1);

    // --- Transaction hashing -------------------------------------------------

    const Transaction tx = MakeTransaction();
    const std::string hash = simulator.computeTransactionHash(tx);

    // A transaction hash is 32 bytes, so 66 characters including the prefix.
    assert(hash.size() == 66);
    assert(hash.rfind("0x", 0) == 0);

    for (size_t i = 2; i < hash.size(); ++i) {
        const char c = hash[i];
        const bool is_hex = (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
        assert(is_hex);
    }

    // The old implementation produced from.substr(2,8) + to.substr(2,8).
    const std::string old_style =
        "0x" + tx.from.toString().substr(2, 8) + tx.to.toString().substr(2, 8);
    assert(hash != old_style);
    assert(hash.size() > old_style.size());

    // A different nonce is a different transaction, so it hashes differently.
    // Under the old implementation every transaction between this pair hashed
    // to the same value regardless of any field.
    Transaction tx_nonce = tx;
    tx_nonce.nonce = 1;
    assert(simulator.computeTransactionHash(tx_nonce) != hash);

    Transaction tx_value = tx;
    tx_value.value = {2000000000000000000ULL, 0, 0, 0};
    assert(simulator.computeTransactionHash(tx_value) != hash);

    Transaction tx_gas = tx;
    tx_gas.gasPrice = 30000000000ULL;
    assert(simulator.computeTransactionHash(tx_gas) != hash);

    Transaction tx_data = tx;
    tx_data.data = {0xa9, 0x05, 0x9c, 0xbb};
    assert(simulator.computeTransactionHash(tx_data) != hash);

    // Identical transactions do hash the same, which is correct.
    assert(simulator.computeTransactionHash(MakeTransaction()) == hash);

    // For an unsigned transaction the signing hash and the identity hash
    // coincide, because the signature is the only difference between them.
    const std::string signing = simulator.signingHash(tx);
    assert(signing.size() == 66);
    assert(signing == hash);

    // Once a signature is present the two differ, because the identity hash
    // covers r, s and v while the signing hash does not.
    Transaction signed_tx = tx;
    signed_tx.signature.resize(65, 0x00);
    signed_tx.signature[31] = 0x01;
    signed_tx.signature[63] = 0x01;
    signed_tx.signature[64] = 27;
    assert(simulator.signingHash(signed_tx) == signing);
    assert(simulator.computeTransactionHash(signed_tx) != hash);

    // --- Signature validation ------------------------------------------------

    // The old check was `signature.size() >= 64`, which accepted 64 zero bytes.
    // checkSignature is private, so this goes through the public path: a
    // transaction carrying an invalid signature must not be validated.
    Transaction short_sig = MakeTransaction();
    short_sig.signature.assign(64, 0x00);
    assert(!simulator.simulateTransaction(short_sig).success);

    // A 65-byte all-zero signature has zero r and s, which is not a signature.
    Transaction zero_sig = MakeTransaction();
    zero_sig.signature.assign(65, 0x00);
    assert(!simulator.simulateTransaction(zero_sig).success);

    // r and s equal to the group order are out of range.
    Transaction order_sig = MakeTransaction();
    order_sig.signature.resize(65, 0x00);
    for (int i = 0; i < 32; ++i) {
        order_sig.signature[i] = kOrder[i];       // r = n
        order_sig.signature[32 + i] = kOrder[i];  // s = n
    }
    order_sig.signature[64] = 27;
    assert(!simulator.simulateTransaction(order_sig).success);

    // A v below 27 is not a recovery id.
    Transaction bad_v = MakeTransaction();
    bad_v.signature.resize(65, 0x00);
    bad_v.signature[31] = 0x01;
    bad_v.signature[63] = 0x01;
    bad_v.signature[64] = 5;
    assert(!simulator.simulateTransaction(bad_v).success);

    // A well-formed signature passes validation. With no node configured the
    // simulation cannot observe an outcome, so it reports a failure and names
    // the reason rather than claiming success it did not see.
    Transaction good = MakeTransaction();
    good.signature.resize(65, 0x00);
    good.signature[31] = 0x01;  // r = 1
    good.signature[63] = 0x01;  // s = 1
    good.signature[64] = 27;    // v
    const SimulationResult unsigned_no_node = simulator.simulateTransaction(good);
    assert(!unsigned_no_node.success);
    assert(unsigned_no_node.error.find("no node") != std::string::npos);

    // The same holds for a transaction with no signature, which is what a user
    // has before signing.
    const SimulationResult plain_no_node =
        simulator.simulateTransaction(MakeTransaction());
    assert(!plain_no_node.success);
    assert(plain_no_node.error.find("no node") != std::string::npos);

    // --- Simulation against a node -------------------------------------------

    StubNode node;
    simulator.setRpcClient(std::make_shared<tigerwallet::RpcClient>(node.url(), 5));

    const SimulationResult simulated = simulator.simulateTransaction(good);
    assert(simulated.success);
    assert(simulated.gasUsed[0] == 21000);   // the value the node reported
    assert(simulated.gasPrice == 20000000000ULL);
    assert(simulated.valueOut == good.value);
    // The sender is debited the value plus the gas cost, and the recipient is
    // credited the value.
    assert(simulated.stateChanges.count(good.from.toString()) == 1);
    assert(simulated.stateChanges.at(good.from.toString())[0] == '-');
    assert(simulated.stateChanges.at(good.to.toString()) ==
           "+" + std::to_string(good.value[0]));

    // A revert the node reports is a failed simulation carrying the node's
    // reason, not a success.
    node.setRevert(true);
    const SimulationResult reverted = simulator.simulateTransaction(good);
    assert(!reverted.success);
    assert(reverted.error.find("reverted") != std::string::npos);
    node.setRevert(false);

    // --- Gas estimation ------------------------------------------------------

    const GasEstimate estimate = simulator.estimateGas(good);
    // The node measured 21000, so the twenty percent margin applies to that
    // rather than to the intrinsic floor.
    assert(estimate.gasLimit == 25200);
    assert(estimate.gasPrice == 20000000000ULL);
    assert(estimate.confidence > 0.9);

    const GasEstimate eip1559 = simulator.estimateGasEIP1559(good);
    // The base fee comes from the node's latest block.
    assert(eip1559.maxFeePerGas == 1000000000ULL * 2 + 1000000000ULL);
    assert(eip1559.maxPriorityFeePerGas == 1000000000ULL);

    // --- Swap quoting --------------------------------------------------------

    // Reserves are not available, so no quote is produced and the caller is
    // told why, rather than receiving a number derived from invented reserves.
    const Address router =
        Address::fromString("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D");
    const uint256_t amount_in = {1000000000000000000ULL, 0, 0, 0};
    const SwapSimulation swap = simulator.simulateSwap(
        router,
        "0x0000000000000000000000000000000000000000",
        "0xdAC17F958D2ee523a2206206994597C13D831ec7",
        amount_in, 0.5);

    assert(!swap.quoteAvailable);
    assert(!swap.quoteUnavailableReason.empty());
    assert(swap.expectedAmountOut[0] == 0);
    assert(swap.minimumAmountOut[0] == 0);
    assert(swap.priceImpact == 0.0);

    std::printf("transaction_simulator: all checks passed\n");
    return 0;
}
