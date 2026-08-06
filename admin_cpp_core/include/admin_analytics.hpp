/**
 * TigerAdmin C++ Core - Analytics & Reporting
 */

#ifndef TIGER_ADMIN_ANALYTICS_HPP
#define TIGER_ADMIN_ANALYTICS_HPP

#include <string>
#include <vector>
#include <map>
#include <optional>
#include "admin_models.hpp"

namespace tiger {
namespace admin {

// ============================================================================
// Analytics Service
// ============================================================================

class AnalyticsService {
public:
    static AnalyticsService& instance();
    
    void initialize();
    
    // Dashboard stats
    PlatformStats get_dashboard_stats();
    
    // User analytics
    struct UserAnalytics {
        int64_t total_users;
        int64_t new_users_today;
        int64_t new_users_this_week;
        int64_t new_users_this_month;
        int64_t active_users_today;
        int64_t active_users_this_week;
        std::map<std::string, int64_t> users_by_country;
        std::map<std::string, int64_t> users_by_status;
    };
    
    UserAnalytics get_user_analytics(const std::string& start_date,
                                     const std::string& end_date);
    
    // Transaction analytics
    struct TransactionAnalytics {
        int64_t total_transactions;
        double total_volume;
        double avg_transaction_size;
        std::map<std::string, int64_t> transactions_by_type;
        std::map<std::string, double> volume_by_token;
        std::map<std::string, double> volume_by_chain;
    };
    
    TransactionAnalytics get_transaction_analytics(const std::string& start_date,
                                                   const std::string& end_date);
    
    // Revenue analytics
    struct RevenueAnalytics {
        double total_fees;
        double fees_today;
        double fees_this_week;
        double fees_this_month;
        std::map<std::string, double> fees_by_token;
        std::map<std::string, double> fees_by_type;
    };
    
    RevenueAnalytics get_revenue_analytics(const std::string& start_date,
                                           const std::string& end_date);
    
    // White label analytics
    struct WhiteLabelAnalytics {
        std::string white_label_id;
        int64_t total_users;
        int64_t active_users;
        double total_volume;
        double total_fees;
        double profit_share;
    };
    
    WhiteLabelAnalytics get_white_label_analytics(WhiteLabelID id,
                                                   const std::string& start_date,
                                                   const std::string& end_date);
    
    // Chart data
    struct ChartDataPoint {
        std::string date;
        double value;
    };
    
    std::vector<ChartDataPoint> get_volume_chart(const std::string& start_date,
                                                   const std::string& end_date,
                                                   const std::string& interval);
    
    std::vector<ChartDataPoint> get_user_chart(const std::string& start_date,
                                                 const std::string& end_date,
                                                 const std::string& interval);
    
    // Real-time metrics
    struct RealtimeMetrics {
        int64_t active_users;
        int64_t transactions_today;
        double volume_today;
        int64_t pending_withdrawals;
        int64_t pending_kyc;
    };
    
    RealtimeMetrics get_realtime_metrics();
    
private:
    AnalyticsService() = default;
};

// ============================================================================
// Report Service
// ============================================================================

class ReportService {
public:
    static ReportService& instance();
    
    void initialize();
    
    // Generate report
    enum class ReportFormat {
        JSON,
        CSV,
        PDF,
        EXCEL
    };
    
    std::string generate_report(const std::string& report_type,
                                const std::string& start_date,
                                const std::string& end_date,
                                ReportFormat format);
    
    // Scheduled reports
    bool create_scheduled_report(const std::string& name,
                                 const std::string& report_type,
                                 const std::string& schedule,
                                 const std::string& recipients);
    
    bool delete_scheduled_report(uint64_t id);
    
    // Export
    std::string export_data(const std::string& entity,
                           const std::map<std::string, std::string>& filters,
                           ReportFormat format);
    
private:
    ReportService() = default;
};

// ============================================================================
// SLA Service
// ============================================================================

class SLAService {
public:
    static SLAService& instance();
    
    void initialize();
    
    // SLA Policies
    struct SLAPolicy {
        uint64_t id;
        std::string name;
        std::string description;
        int response_time_hours;
        int resolution_time_hours;
        TicketPriority priority;
        bool is_active;
    };
    
    std::vector<SLAPolicy> list_sla_policies();
    std::optional<SLAPolicy> get_sla_policy(uint64_t id);
    
    bool create_sla_policy(const std::string& name,
                          const std::string& description,
                          int response_time_hours,
                          int resolution_time_hours,
                          TicketPriority priority);
    
    bool update_sla_policy(uint64_t id,
                          const std::optional<std::string>& name,
                          const std::optional<std::string>& description,
                          const std::optional<int>& response_time_hours,
                          const std::optional<int>& resolution_time_hours);
    
    bool delete_sla_policy(uint64_t id);
    
    // SLA Reports
    struct SLAReport {
        std::string period_start;
        std::string period_end;
        int64_t total_tickets;
        int64_t met_sla;
        int64_t breached_sla;
        double compliance_rate;
        double avg_response_time;
        double avg_resolution_time;
    };
    
    SLAReport get_sla_report(const std::string& start_date,
                             const std::string& end_date);
    
private:
    SLAService() = default;
};

} // namespace admin
} // namespace tiger

#endif // TIGER_ADMIN_ANALYTICS_HPP
