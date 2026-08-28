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
#include <iostream>
#include <sstream>
#include <string>
#include <vector>

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
        << "  exit                              quit\n";
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
