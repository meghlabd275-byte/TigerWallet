#ifndef RWA_SERVICE_HPP
#define RWA_SERVICE_HPP
#include <string>
#include <vector>
#include <unordered_map>
#include <mutex>
using namespace std;
struct RWA { string id, name, type, assetAddress; double price, totalSupply; };
struct RWAOrder { string id, rwaId, type; double amount, price; string status; };
class RWAService {
    unordered_map<string, RWA> rwas_;
    mutex mutex_;
public:
    RWAService() { rwas_["r1"] = {"r1", "Real Estate NYC", "REAL_ESTATE", "0xABC", 1000000, 100}; }
    vector<RWA> getRWAs() { lock_guard<mutex> l(mutex_); vector<RWA> r; for(auto&p:rwas_) r.push_back(p.second); return r; }
    RWAOrder buyRWA(string id, double amt) { return {"ORD-1", id, "BUY", amt, 1000000, "FILLED"}; }
};
#endif
