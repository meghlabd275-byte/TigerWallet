// Uniswap-V2-style DEX connector.
//
// This connector is deliberately thin: it encodes calldata, calls the node and
// decodes what comes back. It keeps no price table and no pool cache, because a
// stale cached reserve would be indistinguishable from a real one at the call
// site. When the node cannot answer, the call reports failure.

#include "dex_connector.hpp"

#include <array>
#include <atomic>
#include <cctype>
#include <cstdlib>
#include <mutex>
#include <optional>
#include <utility>

#include "rpc.hpp"
#include "transaction.hpp"
#include "wallet.hpp"

namespace tigerwallet {
namespace dex {
namespace {

std::string lower_hex(const std::string& in) {
    std::string s = in;
    for (char& c : s) c = static_cast<char>(std::tolower(static_cast<unsigned char>(c)));
    if (!s.empty() && s.rfind("0x", 0) != 0) s = "0x" + s;
    return s;
}

bool is_zero_address(const std::string& address) {
    std::string hex;
    for (char c : address) {
        if (c == '0' || c == 'x' || c == 'X') continue;
        hex.push_back(c);
    }
    return hex.empty();
}

}  // namespace

// ---------------------------------------------------------------------------
// V2 connector
// ---------------------------------------------------------------------------

class V2Connector final : public DexConnector {
public:
    V2Connector(DexProtocol protocol, std::string rpc_url, uint64_t chain_id,
                std::string router, std::string factory, std::string private_key)
        : protocol_(protocol),
          name_(to_string(protocol)),
          rpc_url_(std::move(rpc_url)),
          chain_id_(chain_id),
          router_(lower_hex(router)),
          factory_(lower_hex(factory)),
          private_key_hex_(std::move(private_key)) {
        if (!private_key_hex_.empty()) {
            auto account = derive_account();
            if (account) {
                wallet_ = *account;
                address_ = wallet_->address;
            }
        }
    }

    bool connect() override {
        std::lock_guard<std::mutex> lock(mutex_);
        RpcClient client(rpc_url_);
        const auto chain = client.ChainId();
        if (!chain.ok) return false;
        // The configured contracts are chain-specific; refuse to operate against
        // a node that is not the chain the operator declared.
        if (chain.value != chain_id_) return false;
        connected_ = true;
        return true;
    }

    void disconnect() override {
        std::lock_guard<std::mutex> lock(mutex_);
        connected_ = false;
    }

    bool is_connected() const override { return connected_; }

    bool get_token_info(const std::string& address, Token& out) override {
        out = Token{};
        out.address = lower_hex(address);

        RpcClient client(rpc_url_);
        std::string word;
        if (!encode_uint256("0", word)) return false;

        const std::string symbol_call =
            "0x" + function_selector("symbol()") + word;
        const auto symbol_raw = client.EthCall(out.address, symbol_call);
        if (!symbol_raw.ok) return false;
        out.symbol = decode_string(symbol_raw.value);
        if (out.symbol.empty()) return false;

        const std::string name_call = "0x" + function_selector("name()") + word;
        const auto name_raw = client.EthCall(out.address, name_call);
        if (name_raw.ok) out.name = decode_string(name_raw.value);

        const std::string decimals_call = "0x" + function_selector("decimals()") + word;
        const auto decimals_raw = client.EthCall(out.address, decimals_call);
        if (!decimals_raw.ok) return false;
        const std::string decimals = decode_uint256(decimals_raw.value);
        if (decimals.empty()) return false;
        const unsigned long parsed = std::stoul(decimals);
        if (parsed > 255) return false;
        out.decimals = static_cast<uint8_t>(parsed);
        return true;
    }

    bool get_pool(const std::string& token_a, const std::string& token_b,
                  Pool& out) override {
        out = Pool{};

        RpcClient client(rpc_url_);
        std::string word;
        if (!encode_uint256("0", word)) return false;

        const std::string pair_call = "0x" + function_selector("getPair(address,address)") +
                                      encode_address(token_a) + encode_address(token_b);
        const auto pair_raw = client.EthCall(factory_, pair_call);
        if (!pair_raw.ok) return false;

        const std::string pair_word = decode_uint256(pair_raw.value);
        if (pair_word.empty()) return false;
        std::string pair_address = "0x" + pair_raw.value.substr(24, 40);
        if (is_zero_address(pair_address)) return false;
        out.address = pair_address;

        const std::string reserves_call = "0x" + function_selector("getReserves()") + word;
        const auto reserves_raw = client.EthCall(out.address, reserves_call);
        if (!reserves_raw.ok) return false;

        // getReserves returns (uint112, uint112, uint32) as three words.
        std::string r0, r1;
        if (!word_from(reserves_raw.value, 0, r0)) return false;
        if (!word_from(reserves_raw.value, 1, r1)) return false;
        out.reserve_a = r0;
        out.reserve_b = r1;

        Token a, b;
        if (get_token_info(token_a, a)) out.token_a = a;
        if (get_token_info(token_b, b)) out.token_b = b;
        return true;
    }

    bool get_quote(const std::string& from_token, const std::string& to_token,
                   const std::string& amount, Quote& out) override {
        out = Quote{};
        std::string word;
        if (!encode_uint256(amount, word)) return false;

        RpcClient client(rpc_url_);
        const std::string path =
            encode_address(from_token) + encode_address(to_token);
        // getAmountsOut(uint256,address[]) -> offset(0x40), amount, offset(0x40),
        // length(2), token0, token1
        const std::string head = "0x" + function_selector("getAmountsOut(uint256,address[])") +
                                 word +
                                 std::string(64, '0').replace(62, 2, "40") +
                                 std::string(64, '0').replace(63, 1, "2") +
                                 path.substr(0, 64) + path.substr(64, 64);
        const auto raw = client.EthCall(router_, head);
        if (!raw.ok) return false;

        std::vector<std::string> amounts;
        if (!decode_amounts_array(raw.value, amounts) || amounts.size() < 2) return false;

        out.from_token = lower_hex(from_token);
        out.to_token = lower_hex(to_token);
        out.from_amount = amounts.front();
        out.to_amount = amounts.back();
        out.path = {out.from_token, out.to_token};
        out.price = decimal_mul_div(out.to_amount, "1000000000000000000", out.from_amount, out.price)
                        ? out.price
                        : "0";
        return true;
    }

    SwapResult execute_swap(const std::string& from_token,
                            const std::string& to_token,
                            const std::string& amount,
                            const std::string& min_out) override {
        SwapResult result;
        result.from_token = lower_hex(from_token);
        result.to_token = lower_hex(to_token);
        result.from_amount = amount;

        if (!wallet_) {
            result.error_message = "no signing key configured";
            return result;
        }

        // The output is measured from the balance delta, so the balance before
        // the swap is read first.
        std::string balance_before;
        if (!get_balance(to_token, address_, balance_before)) {
            result.error_message = "cannot read recipient balance";
            return result;
        }

        RpcClient client(rpc_url_);
        std::string deadline;
        {
            const auto block = client.BlockNumber();
            if (!block.ok) {
                result.error_message = "cannot read block number";
                return result;
            }
            // A short deadline keeps a stale quote from being mined much later.
            deadline = std::to_string(block.value + 1200);
        }

        std::string amount_word, min_word, deadline_word;
        if (!encode_uint256(amount, amount_word) ||
            !encode_uint256(min_out, min_word) ||
            !encode_uint256(deadline, deadline_word)) {
            result.error_message = "invalid amount";
            return result;
        }

        const std::string path = encode_address(from_token) + encode_address(to_token);
        const std::string data = "0x" + function_selector(
            "swapExactTokensForTokens(uint256,uint256,address[],address,uint256)") +
            amount_word + min_word +
            std::string(64, '0').replace(62, 2, "a0") +   // offset to path
            encode_address(address_) +                    // recipient
            deadline_word +
            std::string(64, '0').replace(63, 1, "2") +    // path length
            path.substr(0, 64) + path.substr(64, 64);

        // Approve the router for the input token, then swap.
        const std::string approve_data = "0x" +
            function_selector("approve(address,uint256)") + encode_address(router_) + amount_word;

        Transaction approve_tx;
        approve_tx.chain_id = chain_id_;
        approve_tx.to = result.from_token;
        approve_tx.data_hex = approve_data;
        approve_tx.value_wei = "0";
        approve_tx.type = TxType::EIP1559;

        if (!PopulateTransactionFromNode(rpc_url_, address_, approve_tx)) {
            result.error_message = "cannot populate approval transaction";
            return result;
        }
        const auto approve_sent = Wallet::SignAndSend(*wallet_, password_, approve_tx, rpc_url_);
        if (!approve_sent.ok) {
            result.error_message = "approval failed: " + approve_sent.error;
            return result;
        }

        Transaction swap_tx;
        swap_tx.chain_id = chain_id_;
        swap_tx.to = router_;
        swap_tx.data_hex = data;
        swap_tx.value_wei = "0";
        swap_tx.type = TxType::EIP1559;

        if (!PopulateTransactionFromNode(rpc_url_, address_, swap_tx)) {
            result.error_message = "cannot populate swap transaction";
            return result;
        }
        const auto sent = Wallet::SignAndSend(*wallet_, password_, swap_tx, rpc_url_);
        if (!sent.ok) {
            result.error_message = sent.error;
            return result;
        }

        std::string balance_after;
        if (!get_balance(to_token, address_, balance_after)) {
            result.error_message = "cannot read recipient balance after swap";
            return result;
        }

        std::string received;
        if (!decimal_sub(balance_after, balance_before, received)) {
            result.error_message = "balance delta is negative";
            return result;
        }

        result.success = true;
        result.tx_hash = sent.tx_hash;
        result.to_amount = received;
        return result;
    }

    bool get_balance(const std::string& token, const std::string& owner,
                     std::string& out) override {
        RpcClient client(rpc_url_);
        const std::string data = "0x" + function_selector("balanceOf(address)") +
                                 encode_address(owner);
        const auto raw = client.EthCall(lower_hex(token), data);
        if (!raw.ok) return false;
        const std::string value = decode_uint256(raw.value);
        if (value.empty()) return false;
        out = value;
        return true;
    }

    DexProtocol protocol() const override { return protocol_; }
    uint64_t chain_id() const override { return chain_id_; }
    const std::string& name() const override { return name_; }

private:
    static bool word_from(const std::string& hex_data, size_t index, std::string& out) {
        std::string hex = hex_data;
        if (hex.rfind("0x", 0) == 0) hex = hex.substr(2);
        if (hex.size() < (index + 1) * 64) return false;
        const std::string word = hex.substr(index * 64, 64);
        out = decode_uint256(word);
        return !out.empty();
    }

    std::optional<WalletAccount> derive_account() {
        std::string hex = private_key_hex_;
        if (hex.rfind("0x", 0) == 0) hex = hex.substr(2);
        if (hex.size() != 64) return std::nullopt;
        std::array<uint8_t, 32> key{};
        for (size_t i = 0; i < 32; ++i) {
            key[i] = static_cast<uint8_t>(std::strtol(hex.substr(i * 2, 2).c_str(), nullptr, 16));
        }
        password_ = "dex-connector";  // ephemeral in-memory account
        return Wallet::FromPrivateKey(key, password_, "dex-connector");
    }

    DexProtocol protocol_;
    std::string name_;
    std::string rpc_url_;
    uint64_t chain_id_;
    std::string router_;
    std::string factory_;
    std::string private_key_hex_;
    std::string password_;
    std::optional<WalletAccount> wallet_;
    std::string address_;
    std::atomic<bool> connected_{false};
    mutable std::mutex mutex_;
};

std::unique_ptr<DexConnector> make_v2_connector(DexProtocol protocol,
                                                std::string rpc_url,
                                                uint64_t chain_id,
                                                std::string router,
                                                std::string factory,
                                                std::string private_key) {
    return std::make_unique<V2Connector>(protocol, std::move(rpc_url), chain_id,
                                         std::move(router), std::move(factory),
                                         std::move(private_key));
}

}  // namespace dex
}  // namespace tigerwallet
