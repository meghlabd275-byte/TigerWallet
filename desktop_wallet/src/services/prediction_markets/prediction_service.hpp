#ifndef PREDICTION_SERVICE_HPP
#define PREDICTION_SERVICE_HPP

#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
#include <chrono>

using namespace std;
using namespace std::chrono;

struct PredictionMarket {
    string id;
    string question;
    vector<string> outcomes;
    double totalVolume;
    uint64_t endTime;
    string status;
    string result;
};

struct Bet {
    string id;
    string marketId;
    string outcome;
    double amount;
    double potentialWin;
    string status;
    uint64_t createdAt;
};

class PredictionService {
private:
    mutex mutex_;
    unordered_map<string, PredictionMarket> markets_;
    unordered_map<string, vector<Bet>> bets_;
    
public:
    PredictionService() { initializeMarkets(); }
    
    void initializeMarkets() {
        lock_guard<mutex> lock(mutex_);
        markets_["btc50k"] = {"btc50k", "Will BTC reach $50k by end of year?", {"YES", "NO"}, 500000, 0, "ACTIVE", ""};
    }
    
    vector<PredictionMarket> getMarkets() {
        lock_guard<mutex> lock(mutex_);
        vector<PredictionMarket> result;
        for (auto& p : markets_) result.push_back(p.second);
        return result;
    }
    
    Bet placeBet(string marketId, string outcome, double amount) {
        lock_guard<mutex> lock(mutex_);
        Bet b;
        b.id = "BET-" + to_string(duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count());
        b.marketId = marketId;
        b.outcome = outcome;
        b.amount = amount;
        b.potentialWin = amount * 2;
        b.status = "PENDING";
        b.createdAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        bets_[marketId].push_back(b);
        return b;
    }
};

#endif
