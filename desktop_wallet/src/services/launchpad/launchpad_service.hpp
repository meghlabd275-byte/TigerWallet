/**
 * TigerWallet Desktop - Launchpad Service
 *
 * Delegates to the real ProjectParty backend (http://localhost:8106) via the
 * shared APIClient singleton. This service never fabricates launches,
 * participations, claims or statuses: every value returned originates from a
 * real backend response. On any API failure the methods are fail-closed
 * (return empty results / throw) rather than inventing data.
 */

#ifndef LAUNCHPAD_SERVICE_HPP
#define LAUNCHPAD_SERVICE_HPP

#include <string>
#include <vector>
#include <sstream>
#include <optional>
#include <stdexcept>

#include "services/api_client.h"

using namespace std;

// Base URL of the real ProjectParty launchpad backend.
static constexpr const char* kLaunchpadBackendUrl = "http://localhost:8106";

struct Launch {
    string id;
    string name;
    string symbol;
    string description;
    string tokenAddress;
    double price;
    double hardCap;
    double softCap;
    double raised;
    uint64_t startTime;
    uint64_t endTime;
    string status;
};

struct Participation {
    string id;
    string launchId;
    double amount;
    double tokenAmount;
    string status;
    uint64_t participatedAt;
};

class LaunchpadException : public runtime_error {
public:
    explicit LaunchpadException(const string& message) : runtime_error(message) {}
};

class LaunchpadService {
public:
    LaunchpadService() = default;

    // Returns the real active launches from the ProjectParty backend. Empty on
    // API failure - never fabricated.
    vector<Launch> getActiveLaunches() {
        try {
            string resp = projectPartyGet("/api/v1/launchpad");
            // Backend returns {"launchpads": [ { ... }, ... ]}. Each element's
            // fields are mapped into the existing Launch struct.
            vector<Launch> result;
            for (const auto& obj : tiger::wallet::jsonArrayOfObjects(resp, "launchpads")) {
                Launch l;
                l.id = tiger::wallet::jsonStringField(obj, "id").value_or("");
                l.name = tiger::wallet::jsonStringField(obj, "name").value_or("");
                l.symbol = tiger::wallet::jsonStringField(obj, "symbol").value_or("");
                l.description = tiger::wallet::jsonStringField(obj, "description").value_or("");
                l.tokenAddress = tiger::wallet::jsonStringField(obj, "token_address").value_or("");
                l.price = tiger::wallet::jsonNumberField(obj, "token_price").value_or(0.0);
                l.hardCap = tiger::wallet::jsonNumberField(obj, "hard_cap").value_or(0.0);
                l.softCap = tiger::wallet::jsonNumberField(obj, "soft_cap").value_or(0.0);
                l.raised = tiger::wallet::jsonNumberField(obj, "total_raised").value_or(0.0);
                l.startTime = static_cast<uint64_t>(tiger::wallet::jsonNumberField(obj, "start_time").value_or(0.0));
                l.endTime = static_cast<uint64_t>(tiger::wallet::jsonNumberField(obj, "end_time").value_or(0.0));
                l.status = tiger::wallet::jsonStringField(obj, "status").value_or("");
                result.push_back(l);
            }
            return result;
        } catch (const exception&) {
            // Fail-closed: return nothing rather than inventing launches.
            return {};
        }
    }

    // Submits a real contribution to the ProjectParty backend. Returns the
    // resulting Participation exactly as the backend reports it. Throws
    // LaunchpadException on any failure - never fabricates a "CONFIRMED".
    Participation participate(const string& launchId, double amount) {
        Participation p;
        p.launchId = launchId;
        p.amount = amount;
        p.status = "";
        p.tokenAmount = 0.0;
        p.participatedAt = 0;

        ostringstream body;
        body << "{\"amount\":\"" << amount << "\"}";
        string endpoint = "/api/v1/launchpad/" + launchId + "/contribute";
        string resp;
        try {
            resp = projectPartyPost(endpoint, body.str());
        } catch (const exception& e) {
            throw LaunchpadException(string("launchpad contribute failed: ") + e.what());
        }

        // Backend nests the contribution under "contribution".
        string contribJson = resp;
        auto contribObj = tiger::wallet::jsonArrayOfObjects(resp, "contribution");
        if (!contribObj.empty()) {
            contribJson = contribObj.front();
        }
        p.id = tiger::wallet::jsonStringField(contribJson, "id").value_or("");
        p.status = tiger::wallet::jsonStringField(contribJson, "status").value_or("");
        p.tokenAmount = tiger::wallet::jsonNumberField(contribJson, "token_amount").value_or(0.0);
        p.participatedAt = static_cast<uint64_t>(tiger::wallet::jsonNumberField(contribJson, "created_at").value_or(0.0));

        if (p.id.empty() || p.status.empty()) {
            throw LaunchpadException("launchpad contribute returned no contribution");
        }
        return p;
    }

    // Performs a real token claim against the ProjectParty backend. Returns
    // true only when the backend confirms the claim; throws on failure. Never
    // returns a fabricated success.
    bool claimTokens(const string& launchId) {
        string endpoint = "/api/v1/launchpad/" + launchId + "/claim";
        try {
            projectPartyPost(endpoint, "{}");
            return true;
        } catch (const exception& e) {
            throw LaunchpadException(string("launchpad claim failed: ") + e.what());
        }
    }

private:
    // Returns a ProjectParty-backed APIClient singleton, initializing it to
    // the real launchpad backend URL on first use.
    shared_ptr<tiger::wallet::APIClient> client() {
        auto c = tiger::wallet::APIClient::getInstance();
        if (!c->isInitialized()) {
            c->initialize(kLaunchpadBackendUrl);
        }
        return c;
    }

    // Synchronous GET against the ProjectParty backend. executeRequest()
    // attaches the stored Bearer auth token (APIClient::getAuthToken()) to
    // every request as "Authorization: Bearer <token>".
    string projectPartyGet(const string& endpoint) {
        auto c = client();
        // Ensure the stored token is the one used for the auth header.
        string token = c->getAuthToken();
        if (!token.empty()) {
            c->setAuthToken(token);
        }
        string url = c->buildUrl(endpoint, nullopt);
        return c->executeRequest(tiger::wallet::HTTPMethod::GET, url, nullopt);
    }

    // Synchronous POST against the ProjectParty backend with Bearer auth.
    string projectPartyPost(const string& endpoint, const string& body) {
        auto c = client();
        string token = c->getAuthToken();
        if (!token.empty()) {
            c->setAuthToken(token);
        }
        string url = c->buildUrl(endpoint, nullopt);
        return c->executeRequest(tiger::wallet::HTTPMethod::POST, url, body);
    }
};

#endif
