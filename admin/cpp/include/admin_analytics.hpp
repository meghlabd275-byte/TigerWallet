/**
 * TigerAdmin C++ Core - Analytics Header
 */
#pragma once

#include "admin_security.hpp"
#include "admin_blockchain.hpp"

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace tiger {
namespace admin {

struct PlatformStats {
    int64_t total_users = 0;
    int64_t active_users = 0;
    double total_volume = 0.0;
    int64_t total_transactions = 0;
    double total_fees = 0.0;
    int64_t active_bots = 0;
    int64_t total_bots = 0;
};

class AnalyticsService {
public:
    struct ChartDataPoint {
        std::string timestamp;
        double value = 0.0;
    };

    struct UserAnalytics {
        int64_t new_users = 0;
        int64_t active_users = 0;
        int64_t verified_users = 0;
        int64_t kyc_approved = 0;
        int64_t suspended = 0;
        int64_t banned = 0;
        std::vector<ChartDataPoint> signup_chart;
        std::vector<ChartDataPoint> active_chart;
    };

    struct TransactionAnalytics {
        int64_t total_count = 0;
        double total_volume = 0.0;
        double total_fees = 0.0;
        std::vector<ChartDataPoint> volume_chart;
        std::vector<ChartDataPoint> count_chart;
        std::map<std::string, int64_t> by_type;
    };

    struct RevenueAnalytics {
        double total_revenue = 0.0;
        double trading_fees = 0.0;
        double withdrawal_fees = 0.0;
        double other_revenue = 0.0;
        std::vector<ChartDataPoint> revenue_chart;
        std::map<std::string, double> by_source;
    };

    struct WhiteLabelAnalytics {
        std::string company_name;
        int64_t total_users = 0;
        int64_t active_users = 0;
        double total_volume = 0.0;
        double total_revenue = 0.0;
        double platform_share = 0.0;
    };

    struct RealtimeMetrics {
        int64_t active_users = 0;
        int64_t transactions_per_minute = 0;
        double volume_per_minute = 0.0;
        int64_t pending_withdrawals = 0;
        int64_t open_tickets = 0;
    };

    static AnalyticsService& instance();

    void initialize();

    PlatformStats get_dashboard_stats();
    UserAnalytics get_user_analytics(const std::string& start_date,
                                     const std::string& end_date);
    TransactionAnalytics get_transaction_analytics(const std::string& start_date,
                                                   const std::string& end_date);
    RevenueAnalytics get_revenue_analytics(const std::string& start_date,
                                           const std::string& end_date);
    WhiteLabelAnalytics get_white_label_analytics(WhiteLabelID id,
                                                  const std::string& start_date,
                                                  const std::string& end_date);

    std::vector<ChartDataPoint> get_volume_chart(const std::string& start_date,
                                                 const std::string& end_date,
                                                 const std::string& interval);
    std::vector<ChartDataPoint> get_user_chart(const std::string& start_date,
                                               const std::string& end_date,
                                               const std::string& interval);

    RealtimeMetrics get_realtime_metrics();
};

enum class ReportFormat { JSON = 0, CSV = 1, PDF = 2, XLSX = 3 };

class ReportService {
public:
    static ReportService& instance();

    void initialize();

    std::string generate_report(const std::string& report_type,
                                const std::string& start_date,
                                const std::string& end_date,
                                ReportFormat format);
    bool create_scheduled_report(const std::string& name,
                                 const std::string& report_type,
                                 const std::string& schedule,
                                 const std::string& recipients);
    bool delete_scheduled_report(uint64_t id);
    std::string export_data(const std::string& entity,
                            const std::map<std::string, std::string>& filters,
                            ReportFormat format);
};

class SLAService {
public:
    struct SLAPolicy {
        uint64_t id = 0;
        std::string name;
        std::string description;
        int response_time_hours = 0;
        int resolution_time_hours = 0;
        TicketPriority priority = TicketPriority::MEDIUM;
        bool is_active = true;
    };

    struct SLAReport {
        std::string start_date;
        std::string end_date;
        int total_tickets = 0;
        int met_response = 0;
        int met_resolution = 0;
        double response_rate = 0.0;
        double resolution_rate = 0.0;
        double average_response_time = 0.0;
    };

    static SLAService& instance();

    void initialize();

    std::vector<SLAPolicy> list_sla_policies();
    std::optional<SLAPolicy> get_sla_policy(uint64_t id);
    bool create_sla_policy(const std::string& name, const std::string& description,
                           int response_time_hours, int resolution_time_hours,
                           TicketPriority priority);
    bool update_sla_policy(uint64_t id, const std::optional<std::string>& name,
                           const std::optional<std::string>& description,
                           const std::optional<int>& response_time_hours,
                           const std::optional<int>& resolution_time_hours);
    bool delete_sla_policy(uint64_t id);
    SLAReport get_sla_report(const std::string& start_date,
                             const std::string& end_date);
};

} // namespace admin
} // namespace tiger
