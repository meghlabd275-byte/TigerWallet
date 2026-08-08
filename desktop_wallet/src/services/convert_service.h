// Convert Service - C++ Desktop Implementation
// One-click conversion between crypto assets

#ifndef CONVERT_SERVICE_H
#define CONVERT_SERVICE_H

#include <string>
#include <vector>
#include <map>
#include <ctime>
#include <utility>

namespace tigerwallet {

struct ConvertToken {
    std::string symbol;
    std::string name;
    double balance;
    std::string icon;
    
    ConvertToken() : balance(0) {}
    ConvertToken(std::string s, std::string n, double b, std::string i)
        : symbol(std::move(s)), name(std::move(n)), balance(b), icon(std::move(i)) {}
};

struct ConvertPair {
    std::string from;
    std::string to;
    double rate;
    double inverseRate;
    double fee;
    bool enabled;
    
    ConvertPair() : rate(0), inverseRate(0), fee(0), enabled(true) {}
    ConvertPair(std::string f, std::string t, double r, double ir, double fe, bool en)
        : from(std::move(f)), to(std::move(t)), rate(r), inverseRate(ir), fee(fe), enabled(en) {}
};

struct ConvertOrder {
    std::string id;
    std::string userId;
    std::string fromToken;
    std::string toToken;
    double fromAmount;
    double toAmount;
    double rate;
    double fee;
    std::string status;
    time_t createTime;
    
    ConvertOrder() : fromAmount(0), toAmount(0), rate(0), fee(0), createTime(0) {}
};

class ConvertService {
private:
    std::map<std::string, std::vector<ConvertOrder>> orders_;
    std::map<std::string, std::map<std::string, double>> balances_;
    std::vector<ConvertPair> pairs_;
    std::vector<ConvertToken> tokens_;
    
public:
    ConvertService();
    ~ConvertService();
    
    // Initialize tokens and pairs
    void initialize();
    
    // Get available tokens
    std::vector<ConvertToken> getTokens();
    
    // Get conversion rate
    double getRate(const std::string& from, const std::string& to);
    
    // Get balance
    double getBalance(const std::string& userId, const std::string& symbol);
    
    // Execute conversion
    ConvertOrder convert(const std::string& userId, const std::string& from,
                         const std::string& to, double amount);
    
    // Get conversion history
    std::vector<ConvertOrder> getHistory(const std::string& userId);
    
    // Get popular pairs
    std::vector<ConvertPair> getPopularPairs();
};

} // namespace tigerwallet

#endif // CONVERT_SERVICE_H
