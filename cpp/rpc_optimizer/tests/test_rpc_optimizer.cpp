// Tests for the RPC optimizer.
//
// The interesting behaviour is the transport and the JSON-RPC reply handling,
// so these run against a real HTTP server started by the test itself on a
// loopback port. Nothing is mocked: the optimizer opens a socket, sends a
// request and parses what comes back.
//
// Also covered: the metrics that used to be constants (avg 50, min 10, max 500)
// and the cache, which must not serve a stale answer for a submission.

#include "rpc_optimizer.h"

#include <arpa/inet.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>

#include <atomic>
#include <cassert>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <thread>
#include <vector>

using namespace tiger::rpc;

namespace {

// A minimal HTTP server that answers every request with a fixed body. It is a
// real listener on a real port, which is what makes the transport path testable
// without reaching the internet.
class StubServer {
public:
    StubServer(const std::string& response_body, long status = 200)
        : body_(response_body), status_(status) {}

    bool start() {
        listen_fd_ = ::socket(AF_INET, SOCK_STREAM, 0);
        if (listen_fd_ < 0) return false;

        int one = 1;
        ::setsockopt(listen_fd_, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));

        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
        addr.sin_port = 0;  // Let the kernel choose.
        if (::bind(listen_fd_, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) < 0) {
            return false;
        }
        if (::listen(listen_fd_, 8) < 0) return false;

        socklen_t len = sizeof(addr);
        if (::getsockname(listen_fd_, reinterpret_cast<sockaddr*>(&addr), &len) < 0) {
            return false;
        }
        port_ = ntohs(addr.sin_port);

        thread_ = std::thread([this] { serve(); });
        return true;
    }

    void stop() {
        if (!running_.exchange(false)) return;
        // Unblock accept() with one connection.
        const int fd = ::socket(AF_INET, SOCK_STREAM, 0);
        if (fd >= 0) {
            sockaddr_in addr{};
            addr.sin_family = AF_INET;
            addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
            addr.sin_port = htons(port_);
            ::connect(fd, reinterpret_cast<sockaddr*>(&addr), sizeof(addr));
            ::close(fd);
        }
        if (thread_.joinable()) thread_.join();
        if (listen_fd_ >= 0) {
            ::close(listen_fd_);
            listen_fd_ = -1;
        }
    }

    ~StubServer() { stop(); }

    std::string url() const { return "http://127.0.0.1:" + std::to_string(port_) + "/"; }

private:
    void serve() {
        while (running_) {
            const int client = ::accept(listen_fd_, nullptr, nullptr);
            if (client < 0) break;
            if (!running_) {
                ::close(client);
                break;
            }

            std::string request;
            char buffer[4096];
            while (true) {
                const ssize_t n = ::recv(client, buffer, sizeof(buffer), 0);
                if (n <= 0) break;
                request.append(buffer, static_cast<size_t>(n));
                const size_t header_end = request.find("\r\n\r\n");
                if (header_end == std::string::npos) continue;
                size_t content_length = 0;
                const size_t cl = request.find("Content-Length:");
                if (cl != std::string::npos) {
                    content_length = std::strtoul(request.c_str() + cl + 15, nullptr, 10);
                }
                if (request.size() >= header_end + 4 + content_length) break;
            }
            last_request_ = request;
            request_count_++;

            const std::string status_text = (status_ == 200) ? "OK" : "Error";
            std::string response =
                "HTTP/1.1 " + std::to_string(status_) + " " + status_text + "\r\n";
            response += "Content-Type: application/json\r\n";
            response += "Content-Length: " + std::to_string(body_.size()) + "\r\n";
            response += "Connection: close\r\n\r\n";
            response += body_;
            ::send(client, response.data(), response.size(), 0);
            ::close(client);
        }
    }

    std::string body_;
    long status_;
    int listen_fd_ = -1;
    uint16_t port_ = 0;
    std::atomic<bool> running_{true};
    std::atomic<int> request_count_{0};
    std::string last_request_;
    std::thread thread_;

public:
    int request_count() const { return request_count_.load(); }
    std::string last_request() const { return last_request_; }
};

RPCRequest MakeRequest(RPCMethod method, const std::string& chain,
                       const std::vector<std::string>& params = {}) {
    RPCRequest request;
    request.id = "1";
    request.method = method;
    request.params = params;
    request.chain = chain;
    request.timeout_ms = 5000;
    return request;
}

}  // namespace

int main() {
    // --- A successful call ---------------------------------------------------

    {
        StubServer server(R"({"jsonrpc":"2.0","id":"1","result":"0x10"})");
        assert(server.start());

        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "stub", server.url(), 0.0);

        const auto response = optimizer.execute(
            MakeRequest(RPCMethod::ETH_BLOCK_NUMBER, "ethereum"));

        assert(response.has_value());
        assert(response->success);
        assert(response->result == "0x10");
        assert(response->provider == "stub");

        // The request that reached the server must be a JSON-RPC call for the
        // right method. This is what proves the envelope is built correctly.
        const std::string seen = server.last_request();
        assert(seen.find("eth_blockNumber") != std::string::npos);
        assert(seen.find("\"jsonrpc\":\"2.0\"") != std::string::npos);

        // The metrics are measured. The old code returned 50.0, 10.0 and 500.0
        // no matter what happened.
        const OptimizerMetrics metrics = optimizer.get_metrics();
        assert(metrics.total_requests == 1);
        assert(metrics.successful_requests == 1);
        assert(metrics.failed_requests == 0);
        assert(metrics.total_bytes_sent > 0);
        assert(metrics.total_bytes_received > 0);
        assert(metrics.min_latency_ms <= metrics.avg_latency_ms);
        assert(metrics.avg_latency_ms <= metrics.max_latency_ms);
        assert(!(metrics.avg_latency_ms == 50.0 && metrics.min_latency_ms == 10.0 &&
                 metrics.max_latency_ms == 500.0));
    }

    // --- A JSON-RPC error arrives with HTTP 200 ------------------------------

    {
        StubServer server(
            R"({"jsonrpc":"2.0","id":"1","error":{"code":-32602,"message":"invalid argument"}})");
        assert(server.start());

        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "stub", server.url(), 0.0);

        const auto response =
            optimizer.execute(MakeRequest(RPCMethod::ETH_CALL, "ethereum"));
        assert(response.has_value());
        // The transport succeeded, but the call did not. Reporting success here
        // would hand the caller an error object as if it were a result.
        assert(!response->success);
        assert(!response->error.empty());

        const OptimizerMetrics metrics = optimizer.get_metrics();
        assert(metrics.failed_requests == 1);
        assert(metrics.successful_requests == 0);
    }

    // --- A null result is a success, not a failure ---------------------------

    {
        StubServer server(R"({"jsonrpc":"2.0","id":"1","result":null})");
        assert(server.start());

        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "stub", server.url(), 0.0);

        const auto response = optimizer.execute(
            MakeRequest(RPCMethod::ETH_GET_TRANSACTION_RECEIPT, "ethereum"));
        assert(response.has_value());
        assert(response->success);
        assert(response->result.empty());
    }

    // --- An object result is passed through ----------------------------------

    {
        StubServer server(
            R"({"jsonrpc":"2.0","id":"1","result":{"status":"0x1","gasUsed":"0x5208"}})");
        assert(server.start());

        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "stub", server.url(), 0.0);

        const auto response = optimizer.execute(
            MakeRequest(RPCMethod::ETH_GET_TRANSACTION_RECEIPT, "ethereum"));
        assert(response.has_value());
        assert(response->success);
        assert(response->result.find("\"status\":\"0x1\"") != std::string::npos);
    }

    // --- An HTTP error is a failure ------------------------------------------

    {
        StubServer server("bad gateway", 502);
        assert(server.start());

        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "stub", server.url(), 0.0);

        const auto response =
            optimizer.execute(MakeRequest(RPCMethod::ETH_BLOCK_NUMBER, "ethereum"));
        assert(response.has_value());
        assert(!response->success);
        assert(response->error.find("502") != std::string::npos);
    }

    // --- The cache serves reads and never a submission -----------------------

    {
        StubServer server(R"({"jsonrpc":"2.0","id":"1","result":"0x1"})");
        assert(server.start());

        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "stub", server.url(), 0.0);

        const RPCRequest read = MakeRequest(RPCMethod::ETH_BLOCK_NUMBER, "ethereum");

        const auto first = optimizer.execute(read);
        assert(first.has_value() && first->success);
        const int after_first = server.request_count();

        const auto second = optimizer.execute(read);
        assert(second.has_value() && second->success);
        // The second read is answered from the cache, so the server is not
        // contacted again.
        assert(server.request_count() == after_first);
        assert(second->provider == "CACHE");

        const OptimizerMetrics cached = optimizer.get_metrics();
        assert(cached.cache_hits == 1);

        // A submission must never be cached. If it were, a retry would return
        // the first answer and the transaction would never be rebroadcast.
        const RPCRequest send =
            MakeRequest(RPCMethod::ETH_SEND_RAW_TRANSACTION, "ethereum", {"0xdeadbeef"});
        const auto send_first = optimizer.execute(send);
        assert(send_first.has_value() && send_first->success);
        const int after_send = server.request_count();

        const auto send_second = optimizer.execute(send);
        assert(send_second.has_value() && send_second->success);
        assert(server.request_count() == after_send + 1);
        assert(send_second->provider != "CACHE");
    }

    // --- No provider means no call -------------------------------------------

    {
        RPCOptimizer optimizer;
        const auto response =
            optimizer.execute(MakeRequest(RPCMethod::ETH_BLOCK_NUMBER, "unknown-chain"));
        assert(!response.has_value());
    }

    // --- A dead endpoint is reported, not invented ---------------------------

    {
        RPCOptimizer optimizer;
        // Port 1 on loopback has no listener.
        optimizer.add_provider("ethereum", "dead", "http://127.0.0.1:1/", 0.0);

        const auto response =
            optimizer.execute(MakeRequest(RPCMethod::ETH_BLOCK_NUMBER, "ethereum"));
        assert(response.has_value());
        assert(!response->success);
        assert(!response->error.empty());
    }

    // --- Provider selection and removal --------------------------------------

    {
        RPCOptimizer optimizer;
        optimizer.add_provider("ethereum", "one", "http://127.0.0.1:9001/", 0.0);
        optimizer.add_provider("ethereum", "two", "http://127.0.0.1:9002/", 0.0);

        auto providers = optimizer.get_providers("ethereum");
        assert(providers.size() == 2);

        optimizer.remove_provider("ethereum", "one");
        providers = optimizer.get_providers("ethereum");
        assert(providers.size() == 1);
        assert(providers[0].name == "two");

        const auto best = optimizer.get_best_provider("ethereum");
        assert(best.has_value());
        assert(best->name == "two");

        optimizer.reset_metrics();
        const OptimizerMetrics metrics = optimizer.get_metrics();
        assert(metrics.total_requests == 0);
        assert(metrics.avg_latency_ms == 0.0);
    }

    std::printf("rpc_optimizer: all checks passed\n");
    return 0;
}
