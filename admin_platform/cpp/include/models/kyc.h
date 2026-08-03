#pragma once

#include <string>
#include <vector>
#include <optional>
#include <chrono>
#include <nlohmann/json.hpp>

namespace tiger::models {

using json = nlohmann::json;

// KYC Record model
struct KYCRecord {
    std::string id;
    std::string user_id;
    int level;
    std::string document_type;
    std::optional<std::string> document_number;
    std::optional<std::string> document_front;
    std::optional<std::string> document_back;
    std::optional<std::string> selfie_image;
    std::optional<std::string> first_name;
    std::optional<std::string> last_name;
    std::optional<std::string> date_of_birth;
    std::optional<std::string> country;
    std::optional<std::string> address;
    std::string status;
    std::optional<std::string> reject_reason;
    std::optional<std::string> reviewed_by;
    std::optional<std::chrono::system_clock::time_point> reviewed_at;
    std::optional<std::string> notes;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    
    json to_json() const;
    static KYCRecord from_json(const json& j);
    
    bool is_approved() const;
    bool is_pending() const;
    bool is_rejected() const;
};

} // namespace tiger::models
