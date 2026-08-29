/**
 * MasterWallet Desktop — real console driver.
 *
 * Commands are routed to MasterWalletService methods which delegate
 * to the canonical backend (:8450). No client-side key material,
 * no fabricated balances, no simulation. If the backend is down or
 * the user is not authenticated, commands fail loudly and honestly.
 */

#include "master_wallet_service.hpp"
#include "api_client.hpp"
#include "theme.hpp"

#include <cstdlib>
#include <cstring>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

#include <arpa/inet.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>

using tiger::master::api::APIClient;
using tiger::master::api::backend;
using tiger::master::MasterWalletService;
using tiger::master::ui::ThemeManager;

namespace {

std::vector<std::string> split(const std::string& s) {
    std::istringstream iss(s);
    std::vector<std::string> out;
    for (std::string tok; iss >> tok;) out.push_back(tok);
    return out;
}

void printHelp() {
    std::cout
        << "MasterWallet console — commands (all hit the real backend):\n"
        << "  help                              this list\n"
        << "  health                            GET /health\n"
        << "  chains                            list chains\n"
        << "  tokens                            list tokens\n"
        << "  wallets                           list master wallets\n"
        << "  wallet <id>                       show wallet\n"
        << "  balance <wallet> [chain]          show balance\n"
        << "  tx <wallet> <to> <amount> <chain> create+sign transaction\n"
        << "  txs <wallet>                      list transactions\n"
        << "  users <wallet>                    list users\n"
        << "  fees <wallet>                     list fees\n"
        << "  policies <wallet>                 list policies\n"
        << "  auto <wallet>                     list auto-sign rules\n"
        << "  treasury <wallet>                 treasury summary\n"
        << "  analytics <wallet>                volume/tx/users\n"
        << "  audit <wallet>                    audit log\n"
        << "  notifications <wallet>            list notifications\n"
        << "  webhooks <wallet>                 list webhooks\n"
        << "  theme                             show current theme CSS\n"
        << "  gui                               serve the React GUI (dist/) on 127.0.0.1\n"
        << "  exit                              quit\n";
}

// ---------------------------------------------------------------------------
// gui: minimal loopback-only static file server for the built React GUI.
// It serves desktop/dist (built by `npm run build`) and injects the backend
// base URL into index.html so the GUI always talks to the same backend this
// console is configured with. Binds 127.0.0.1 only — never exposed externally.
// ---------------------------------------------------------------------------

std::string mimeType(const std::string& path) {
    auto ends = [&](const char* ext) {
        const size_t n = std::strlen(ext);
        return path.size() >= n && path.compare(path.size() - n, n, ext) == 0;
    };
    if (ends(".html")) return "text/html; charset=utf-8";
    if (ends(".js"))   return "application/javascript; charset=utf-8";
    if (ends(".css"))  return "text/css; charset=utf-8";
    if (ends(".json")) return "application/json";
    if (ends(".svg"))  return "image/svg+xml";
    if (ends(".png"))  return "image/png";
    if (ends(".ico"))  return "image/x-icon";
    if (ends(".woff2")) return "font/woff2";
    return "application/octet-stream";
}

bool readFile(const std::string& path, std::string& out) {
    std::ifstream f(path, std::ios::binary);
    if (!f) return false;
    std::ostringstream ss;
    ss << f.rdbuf();
    out = ss.str();
    return true;
}

int cmdGui(const std::string& baseUrl) {
    std::string dir = "dist";
    if (const char* env = std::getenv("MASTER_WALLET_GUI_DIR")) {
        if (env && *env) dir = env;
    }
    int port = 8452;
    if (const char* env = std::getenv("MASTER_WALLET_GUI_PORT")) {
        if (env && *env) port = std::atoi(env);
    }

    std::string probe;
    if (!readFile(dir + "/index.html", probe)) {
        std::cerr << "gui: " << dir << "/index.html not found — run "
                  << "`npm install && npm run build` in master_wallet/desktop first\n";
        return 1;
    }

    int srv = ::socket(AF_INET, SOCK_STREAM, 0);
    if (srv < 0) { std::cerr << "gui: socket failed\n"; return 1; }
    int one = 1;
    ::setsockopt(srv, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));
    sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK); // loopback only
    addr.sin_port = htons(static_cast<uint16_t>(port));
    if (::bind(srv, reinterpret_cast<sockaddr*>(&addr), sizeof(addr)) < 0 ||
        ::listen(srv, 8) < 0) {
        std::cerr << "gui: cannot bind 127.0.0.1:" << port << "\n";
        ::close(srv);
        return 1;
    }
    std::cout << "gui: serving " << dir << " at http://127.0.0.1:" << port
              << " (backend " << baseUrl << "). Ctrl+C to stop.\n";

    // Inject the backend URL once into the cached index.html.
    const std::string inject =
        "<script>window.__MASTER_API_URL__=\"" + baseUrl + "\";</script>";
    const std::string marker = "<script type=\"module\"";
    auto pos = probe.find(marker);
    if (pos != std::string::npos) probe.insert(pos, inject);

    for (;;) {
        int fd = ::accept(srv, nullptr, nullptr);
        if (fd < 0) continue;
        char buf[4096];
        ssize_t n = ::recv(fd, buf, sizeof(buf) - 1, 0);
        if (n <= 0) { ::close(fd); continue; }
        buf[n] = '\0';
        // Parse "GET /path HTTP/1.1" — anything else gets 405.
        std::istringstream req(buf);
        std::string method, target, version;
        req >> method >> target >> version;
        std::string body, status = "200 OK", mime;
        if (method != "GET" && method != "HEAD") {
            status = "405 Method Not Allowed";
            body.clear();
            mime = "text/plain";
        } else {
            std::string path = target;
            auto q = path.find('?');
            if (q != std::string::npos) path.resize(q);
            if (path.empty() || path == "/") path = "/index.html";
            // Reject traversal outright.
            if (path.find("..") != std::string::npos) {
                status = "400 Bad Request";
                body.clear();
                mime = "text/plain";
            } else if (path == "/index.html") {
                body = probe; // injected variant
                mime = "text/html; charset=utf-8";
            } else if (!readFile(dir + path, body)) {
                // SPA fallback: unknown paths serve the app shell.
                body = probe;
                mime = "text/html; charset=utf-8";
            } else {
                mime = mimeType(path);
            }
        }
        std::ostringstream resp;
        resp << "HTTP/1.1 " << status << "\r\n"
             << "Content-Type: " << mime << "\r\n"
             << "Content-Length: " << (method == "HEAD" ? 0 : body.size()) << "\r\n"
             << "Cache-Control: no-store\r\n"
             << "Connection: close\r\n\r\n";
        const std::string head = resp.str();
        ::send(fd, head.data(), head.size(), 0);
        if (method != "HEAD" && !body.empty())
            ::send(fd, body.data(), body.size(), 0);
        ::close(fd);
    }
    return 0;
}

int cmdHealth() {
    std::cout << backend()->get("/health") << "\n";
    return 0;
}

int cmdChains() {
    std::cout << backend()->get("/api/v1/chains") << "\n";
    return 0;
}

int cmdTokens() {
    std::cout << backend()->get("/api/v1/tokens") << "\n";
    return 0;
}

int cmdWallets() {
    std::cout << backend()->get("/api/v1/master-wallet") << "\n";
    return 0;
}

int cmdWallet(const std::string& id) {
    auto r = MasterWalletService::getInstance().getWallet(id);
    if (!r) { std::cerr << "not found\n"; return 1; }
    std::cout << r->address << " chains=" << r->supportedChains.size() << "\n";
    return 0;
}

int cmdBalance(const std::string& id, const std::string& chain) {
    auto r = MasterWalletService::getInstance().getBalance(
        id, chain.empty() ? 1 : std::stoul(chain), "");
    std::cout << r.balance << " " << r.symbol << "\n";
    return r.success ? 0 : 1;
}

int cmdTx(const std::string& wid, const std::string& to,
          const std::string& amount, const std::string& chain) {
    tiger::master::TransactionRequest req;
    req.fromWallet = wid;
    req.toAddress = to;
    req.amount = amount;
    req.chainId = std::stoul(chain);
    auto r = MasterWalletService::getInstance().signAndBroadcast(req);
    std::cout << r.txHash << "\n";
    return r.success ? 0 : 1;
}

int cmdTxs(const std::string& wid) {
    auto r = MasterWalletService::getInstance().getTransactions(wid);
    for (const auto& t : r) std::cout << t.hash << " " << t.status << "\n";
    return 0;
}

int cmdUsers(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getUsers(wid) << "\n";
    return 0;
}

int cmdFees(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getFees(wid) << "\n";
    return 0;
}

int cmdPolicies(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getPolicies(wid) << "\n";
    return 0;
}

int cmdAuto(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getAutoSignRules(wid) << "\n";
    return 0;
}

int cmdTreasury(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getTreasury(wid) << "\n";
    return 0;
}

int cmdAnalytics(const std::string& wid) {
    auto& s = MasterWalletService::getInstance();
    std::cout << s.getAnalyticsVolume(wid) << "\n"
              << s.getAnalyticsTransactions(wid) << "\n"
              << s.getAnalyticsWallets(wid) << "\n";
    return 0;
}

int cmdAudit(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getAudit(wid) << "\n";
    return 0;
}

int cmdNotifications(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getNotifications(wid) << "\n";
    return 0;
}

int cmdWebhooks(const std::string& wid) {
    std::cout << MasterWalletService::getInstance().getWebhooks(wid) << "\n";
    return 0;
}

} // namespace

int main() {
    std::string baseUrl = "http://localhost:8450";
    if (const char* env = std::getenv("MASTER_WALLET_API_URL")) {
        if (env && *env) baseUrl = env;
    }
    try {
        APIClient::instance()->initialize(baseUrl);
    } catch (const std::exception& e) {
        std::cerr << "init: " << e.what() << "\n";
        return 1;
    }

    auto& theme = ThemeManager::getInstance();
    theme.loadFromFile("theme.json");
    std::cout << "MasterWallet console — backend " << baseUrl
              << ", theme " << (theme.isDark() ? "dark" : "light")
              << ". Type 'help'.\n";

    for (std::string line; std::cout << "> " && std::getline(std::cin, line);) {
        if (line.empty()) continue;
        auto args = split(line);
        const auto& cmd = args[0];
        try {
            if (cmd == "help") printHelp();
            else if (cmd == "exit") break;
            else if (cmd == "health") cmdHealth();
            else if (cmd == "chains") cmdChains();
            else if (cmd == "tokens") cmdTokens();
            else if (cmd == "wallets") cmdWallets();
            else if (cmd == "wallet" && args.size() > 1) cmdWallet(args[1]);
            else if (cmd == "balance" && args.size() > 1)
                cmdBalance(args[1], args.size() > 2 ? args[2] : "");
            else if (cmd == "tx" && args.size() > 4)
                cmdTx(args[1], args[2], args[3], args[4]);
            else if (cmd == "txs" && args.size() > 1) cmdTxs(args[1]);
            else if (cmd == "users" && args.size() > 1) cmdUsers(args[1]);
            else if (cmd == "fees" && args.size() > 1) cmdFees(args[1]);
            else if (cmd == "policies" && args.size() > 1) cmdPolicies(args[1]);
            else if (cmd == "auto" && args.size() > 1) cmdAuto(args[1]);
            else if (cmd == "treasury" && args.size() > 1) cmdTreasury(args[1]);
            else if (cmd == "analytics" && args.size() > 1) cmdAnalytics(args[1]);
            else if (cmd == "audit" && args.size() > 1) cmdAudit(args[1]);
            else if (cmd == "notifications" && args.size() > 1)
                cmdNotifications(args[1]);
            else if (cmd == "webhooks" && args.size() > 1) cmdWebhooks(args[1]);
            else if (cmd == "theme")
                std::cout << theme.getCssVariables() << "\n";
            else if (cmd == "gui") cmdGui(baseUrl);
            else
                std::cerr << "unknown or missing args; try 'help'\n";
        } catch (const std::exception& e) {
            std::cerr << "error: " << e.what() << "\n";
        }
    }

    theme.saveToFile("theme.json");
    backend()->shutdown();
    return 0;
}
