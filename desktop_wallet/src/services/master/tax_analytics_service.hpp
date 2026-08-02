/**
 * Tax and Analytics Services - C++ Desktop Implementation
 * Identical across ALL platforms
 */

#ifndef TAX_ANALYTICS_SERVICE_HPP
#define TAX_ANALYTICS_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <cstdint>

namespace tigerwallet {

// ==================== TAX SERVICE ====================

enum class CostBasisMethod { FIFO, LIFO, HIFO };
enum class TransactionType { BUY, SELL, TRANSFER, SWAP, STAKE, UNSTAKE, MINT, BURN };

struct TaxTransaction {
    std::string id;
    TransactionType type;
    std::string date;
    std::string asset;
    int amount;
    int price;
    int costBasis;
    int proceeds;
    int gainLoss;
    std::string exchange;
    std::string txHash;
};

struct TaxLot {
    std::string id;
    std::string asset;
    int amount;
    int remainingAmount;
    int costBasis;
    int fairMarketValue;
    std::string acquisitionDate;
    bool isLongTerm;
};

struct IncomeEvent {
    std::string id;
    std::string type;
    std::string asset;
    int amount;
    int fairMarketValue;
    std::string date;
};

struct TaxReport {
    int year;
    int shortTermGains;
    int shortTermLosses;
    int longTermGains;
    int longTermLosses;
    int totalIncome;
    int totalTransactions;
    std::string jurisdiction;
    CostBasisMethod costBasisMethod;
};

class TaxService {
public:
    static TaxService& getInstance();
    
    bool setJurisdiction(const std::string& jurisdictionCode);
    bool setCostBasisMethod(CostBasisMethod method);
    void addTransaction(const TaxTransaction& tx);
    TaxReport calculateGains();
    std::vector<TaxLot> getAvailableLots(const std::string& asset);
    void addIncomeEvent(const IncomeEvent& event);
    std::string exportCSV();

private:
    TaxService() : jurisdiction_("US"), costBasisMethod_(CostBasisMethod::FIFO) {}
    TaxService(const TaxService&) = delete;
    TaxService& operator=(const TaxService&) = delete;
    
    std::vector<TaxTransaction> transactions_;
    std::map<std::string, std::vector<TaxLot>> taxLots_;
    std::vector<IncomeEvent> incomeEvents_;
    std::string jurisdiction_;
    CostBasisMethod costBasisMethod_;
};

// ==================== ANALYTICS SERVICE ====================

enum class AlertCondition { ABOVE, BELOW };

struct AssetHolding {
    std::string symbol;
    std::string name;
    std::string chain;
    std::string category;
    int balance;
    double price;
    int value;
    double allocation;
    double change24h;
};

struct PortfolioSummary {
    int totalValue;
    int change24h;
    double changePercent24h;
    std::vector<AssetHolding> assets;
    int64_t lastUpdated;
};

struct PerformanceMetrics {
    std::string timeframe;
    double totalReturn;
    double annualizedReturn;
    double volatility;
    double sharpeRatio;
    double maxDrawdown;
    std::string riskLevel;
};

struct AllocationBreakdown {
    std::map<std::string, int> byChain;
    std::map<std::string, int> byCategory;
    int totalValue;
    double diversificationScore;
};

struct PortfolioTransaction {
    std::string id;
    std::string type;
    std::string asset;
    int amount;
    int value;
    std::string date;
    std::string txHash;
};

struct PriceAlert {
    std::string id;
    std::string asset;
    AlertCondition condition;
    double targetPrice;
    bool isActive;
    int64_t createdAt;
};

class AnalyticsService {
public:
    static AnalyticsService& getInstance();
    
    void updatePortfolio(const std::map<std::string, AssetHolding>& holdings);
    PortfolioSummary getSummary();
    PerformanceMetrics getPerformance(const std::string& timeframe);
    AllocationBreakdown getAllocation();
    std::vector<PortfolioTransaction> getTransactionHistory(
        const std::string& startDate = "",
        const std::string& endDate = "",
        const std::vector<std::string>& type = {});
    PriceAlert setAlert(const std::string& asset, AlertCondition condition, double targetPrice);
    std::vector<PriceAlert> getAlerts();
    bool deleteAlert(const std::string& alertId);
    std::string exportReport(const std::string& format);

private:
    AnalyticsService() : totalPortfolioValue_(0), previousPortfolioValue_(0) {}
    AnalyticsService(const AnalyticsService&) = delete;
    AnalyticsService& operator=(const AnalyticsService&) = delete;
    
    std::map<std::string, AssetHolding> holdings_;
    std::vector<PortfolioTransaction> transactions_;
    std::vector<PriceAlert> alerts_;
    int totalPortfolioValue_;
    int previousPortfolioValue_;
    
    void recalculateValue();
    double calculateDiversificationScore(const std::map<std::string, int>& byChain);
};

} // namespace tigerwallet

#endif // TAX_ANALYTICS_SERVICE_HPP
