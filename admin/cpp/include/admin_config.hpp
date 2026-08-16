/**
 * TigerAdmin C++ Core - Configuration Header
 */
#pragma once

#include <string>

namespace tiger {
namespace admin {

struct Config {
    std::string host = "0.0.0.0";
    // The C++ service listens here; it proxies all /api/v1 domain calls to the
    // admin/go backend on 9093 (see admin_domain.hpp upstream_port()). Using a
    // distinct port avoids the service proxying to itself.
    int port = 9094;
    int worker_threads = 4;
    std::string log_file;
    std::string db_host = "localhost";
    int db_port = 5432;
    std::string db_name = "tigeradmin";
    std::string db_user = "postgres";
    std::string db_password;
    std::string jwt_secret;

    static Config load() {
        Config c;
        // In production these would be read from env/config files
        return c;
    }
};

} // namespace admin
} // namespace tiger
