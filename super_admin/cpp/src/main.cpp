/**
 * TigerWallet Super Admin C++ Core — real CLI driver.
 *
 * A thin, ultra-low-latency command-line front over the header-only
 * SuperAdminHttpClient, which forwards every command to the canonical
 * `super_admin/go` backend on 127.0.0.1:8082. No data is fabricated: a down
 * upstream surfaces as a transport error and a non-zero exit code.
 *
 * Usage:
 *   tiger_super_admin_cpp <jwt> <method> <path> [json_body]
 *   tiger_super_admin_cpp <jwt> health
 *
 * Governance records only; never moves crypto assets.
 */
#include "super_admin_domains.hpp"

#include <cstdio>
#include <cstdlib>
#include <iostream>
#include <string>
#include <string_view>

using tiger::admin::domains::DomainResult;
using tiger::admin::domains::HttpMethod;
using tiger::admin::domains::SuperAdminHttpClient;

namespace {

HttpMethod parse_method(std::string_view m) {
    if (m == "GET" || m == "get") return HttpMethod::GET;
    if (m == "POST" || m == "post") return HttpMethod::POST;
    if (m == "PUT" || m == "put") return HttpMethod::PUT;
    if (m == "DELETE" || m == "delete") return HttpMethod::DELETE;
    return HttpMethod::GET;
}

int print(const DomainResult& r) {
    if (r.transport_error()) {
        std::cerr << "transport error: " << r.error << "\n";
        return 2;
    }
    std::cout << r.body;
    if (!r.body.empty() && r.body.back() != '\n') std::cout << "\n";
    return (r.status_code >= 200 && r.status_code < 300) ? 0 : 1;
}

} // namespace

int main(int argc, char* argv[]) {
    if (argc < 3) {
        std::cerr << "usage: " << (argc > 0 ? argv[0] : "tiger_super_admin_cpp")
                  << " <jwt> <method> <path> [json_body]\n"
                  << "       " << (argc > 0 ? argv[0] : "tiger_super_admin_cpp")
                  << " <jwt> health\n";
        return 64;
    }

    const std::string jwt = argv[1];
    SuperAdminHttpClient client(jwt);

    if (std::string(argv[2]) == "health") {
        // /api/v1/admin/health is the real super_admin/go liveness route.
        return print(client.get("health"));
    }

    const HttpMethod method = parse_method(argv[2]);
    if (argc < 4) {
        std::cerr << "error: <path> is required for method " << argv[2] << "\n";
        return 64;
    }
    const std::string path = argv[3];
    const std::string body = (argc >= 5) ? argv[4] : std::string{};

    return print(client.request(method, path, body));
}
