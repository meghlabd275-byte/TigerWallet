/**
 * TigerWallet Admin - Billing Handler (C++ Ultra-Low Latency)
 */

#pragma once

#include <string>
#include <vector>
#include <memory>
#include <chrono>
#include <optional>
#include "admin_handler.hpp"

namespace tiger {
namespace admin {

struct SubscriptionPlan {
    uint64_t id;
    std::string name;
    double price;
    std::string period;
    std::vector<std::string> features;
    uint32_t max_users;
    bool is_active;
};

struct Subscription {
    uint64_t id;
    uint64_t plan_id;
    std::string plan_name;
    std::string status;
    std::chrono::system_clock::time_point current_period_start;
    std::chrono::system_clock::time_point current_period_end;
    uint32_t users;
    uint64_t api_calls;
};

struct Invoice {
    uint64_t id;
    std::string invoice_number;
    uint64_t subscription_id;
    double amount;
    std::string status;
    std::chrono::system_clock::time_point created_at;
};

struct PaymentMethod {
    uint64_t id;
    std::string type;
    std::string last4;
    std::string expiry_month;
    std::string expiry_year;
    bool is_default;
};

class BillingHandler : public AdminHandler {
public:
    BillingHandler();
    ~BillingHandler() override = default;
    void Initialize(ConnectionPool* pool) override;

    std::vector<SubscriptionPlan> GetPlans();
    std::optional<SubscriptionPlan> GetPlanById(uint64_t id);
    SubscriptionPlan CreatePlan(const std::string& name, double price, const std::string& period,
                               const std::vector<std::string>& features, uint32_t max_users);
    std::optional<Subscription> GetSubscription();
    Subscription CreateSubscription(uint64_t plan_id);
    bool CancelSubscription();
    std::vector<Invoice> GetInvoices(int page = 1, int limit = 20);
    std::vector<PaymentMethod> GetPaymentMethods();
    bool AddPaymentMethod(const std::string& type, const std::string& last4,
                         const std::string& expiry_month, const std::string& expiry_year);
    bool SetDefaultPaymentMethod(uint64_t id);

private:
    ConnectionPool* pool_;
    void PrepareStatements();
    SubscriptionPlan ParsePlanRow(const Row& row);
    Subscription ParseSubscriptionRow(const Row& row);
    Invoice ParseInvoiceRow(const Row& row);
    PaymentMethod ParsePaymentMethodRow(const Row& row);
};

inline std::vector<SubscriptionPlan> BillingHandler::GetPlans() {
    std::vector<SubscriptionPlan> result;
    auto stmt = pool_->Prepare("SELECT * FROM subscription_plans WHERE is_active = true ORDER BY price ASC");
    auto rows = stmt->Execute();
    while (rows->Next()) result.push_back(ParsePlanRow(*rows));
    return result;
}

inline std::optional<Subscription> BillingHandler::GetSubscription() {
    auto stmt = pool_->Prepare(
        "SELECT s.id, s.plan_id, p.name, s.status, s.current_period_start, s.current_period_end, "
        "s.users, s.api_calls FROM subscriptions s "
        "JOIN subscription_plans p ON s.plan_id = p.id WHERE s.status = 'active' LIMIT 1"
    );
    auto rows = stmt->Execute();
    if (rows->Next()) return ParseSubscriptionRow(*rows);
    return std::nullopt;
}

inline Subscription BillingHandler::CreateSubscription(uint64_t plan_id) {
    auto now = std::chrono::system_clock::now();
    auto period_end = now + std::chrono::days(30);
    auto stmt = pool_->Prepare(
        "INSERT INTO subscriptions (plan_id, status, current_period_start, current_period_end, users, api_calls, created_at) "
        "VALUES ($1, 'active', $2, $3, 0, 0, NOW()) RETURNING *"
    );
    auto rows = stmt->Execute(plan_id, now, period_end);
    if (rows->Next()) return ParseSubscriptionRow(*rows);
    return {};
}

inline bool BillingHandler::CancelSubscription() {
    auto stmt = pool_->Prepare("UPDATE subscriptions SET status = 'cancelled' WHERE status = 'active'");
    auto result = stmt->Execute();
    return result->AffectedRows() > 0;
}

inline std::vector<Invoice> BillingHandler::GetInvoices(int page, int limit) {
    std::vector<Invoice> result;
    auto offset = (page - 1) * limit;
    auto stmt = pool_->Prepare("SELECT * FROM invoices ORDER BY created_at DESC LIMIT $1 OFFSET $2");
    auto rows = stmt->Execute(limit, offset);
    while (rows->Next()) result.push_back(ParseInvoiceRow(*rows));
    return result;
}

inline std::vector<PaymentMethod> BillingHandler::GetPaymentMethods() {
    std::vector<PaymentMethod> result;
    auto stmt = pool_->Prepare("SELECT * FROM payment_methods ORDER BY is_default DESC, created_at DESC");
    auto rows = stmt->Execute();
    while (rows->Next()) result.push_back(ParsePaymentMethodRow(*rows));
    return result;
}

inline bool BillingHandler::AddPaymentMethod(const std::string& type, const std::string& last4,
                                            const std::string& expiry_month, const std::string& expiry_year) {
    auto stmt = pool_->Prepare(
        "INSERT INTO payment_methods (type, last4, expiry_month, expiry_year, is_default, created_at) "
        "VALUES ($1, $2, $3, $4, false, NOW())"
    );
    auto result = stmt->Execute(type, last4, expiry_month, expiry_year);
    return result->AffectedRows() > 0;
}

} // namespace admin
} // namespace tiger
