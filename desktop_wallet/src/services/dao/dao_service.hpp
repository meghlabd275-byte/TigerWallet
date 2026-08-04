#ifndef DAO_SERVICE_HPP
#define DAO_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <unordered_map>
#include <mutex>
#include <chrono>
#include <functional>

using namespace std;
using namespace std::chrono;

struct DAO {
    string id;
    string name;
    string description;
    string token;
    string treasuryAddress;
    int memberCount;
    double treasuryValue;
    uint64_t createdAt;
};

struct Proposal {
    string id;
    string daoId;
    string title;
    string description;
    string type;
    string status;
    double forVotes;
    double againstVotes;
    int quorum;
    uint64_t createdAt;
    uint64_t executedAt;
};

struct Vote {
    string id;
    string proposalId;
    string voter;
    string choice;
    double weight;
    uint64_t votedAt;
};

class DAOService {
private:
    mutex mutex_;
    unordered_map<string, DAO> daos_;
    unordered_map<string, vector<Proposal>> daoProposals_;
    unordered_map<string, vector<Vote>> proposalVotes_;
    
public:
    DAOService() { initializeDAOs(); }
    
    void initializeDAOs() {
        lock_guard<mutex> lock(mutex_);
        DAOs["tiger"] = {"tiger", "TigerDAO", "Tiger Foundation DAO", "TIGER", "0x123...abc", 5000, 1000000};
        DAOs["uni"] = {"uni", "Uniswap DAO", "Uniswap Governance", "UNI", "0x456...def", 150000, 50000000};
    }
    
    vector<DAO> getDAOs() {
        lock_guard<mutex> lock(mutex_);
        vector<DAO> result;
        for (auto& p : daos_) result.push_back(p.second);
        return result;
    }
    
    vector<Proposal> getProposals(string daoId) {
        lock_guard<mutex> lock(mutex_);
        return daoProposals_[daoId];
    }
    
    Proposal createProposal(string daoId, string title, string desc, string type) {
        lock_guard<mutex> lock(mutex_);
        Proposal p;
        p.id = "PROP-" + to_string(duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count());
        p.daoId = daoId;
        p.title = title;
        p.description = desc;
        p.type = type;
        p.status = "PENDING";
        p.forVotes = 0;
        p.againstVotes = 0;
        p.quorum = 1000000;
        p.createdAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        daoProposals_[daoId].push_back(p);
        return p;
    }
    
    bool vote(string proposalId, string voter, string choice, double weight) {
        lock_guard<mutex> lock(mutex_);
        Vote v;
        v.id = "VOTE-" + to_string(duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count());
        v.proposalId = proposalId;
        v.voter = voter;
        v.choice = choice;
        v.weight = weight;
        v.votedAt = duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
        proposalVotes_[proposalId].push_back(v);
        return true;
    }
};

#endif
