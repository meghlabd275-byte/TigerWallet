#ifndef MASTER_WALLET_TAX_ANALYTICS_SERVICE_HPP
#define MASTER_WALLET_TAX_ANALYTICS_SERVICE_HPP

#include <string>
#include <vector>
#include <map>
#include <memory>
#include <functional>
#include <optional>
#include <chrono>
#include <mutex>

namespace tiger {
namespace master {
namespace tax {

// Forward declarations
class TaxAnalyticsService;
class TaxCalculator;
class CostBasisTracker;

/**
 * Transaction - Tax-relevant transaction
 */
struct TaxTransaction {
    std::string txId;
    std::string walletAddress;
    std::string hash;
    std::string type;  // buy, sell, transfer, reward, staking, swap
    std::string asset;
    uint64_t quantity;
    double priceUSD;
    double feeUSD;
    std::string chainId;
    std::string timestamp;
    std::string counterpart;
    std::string notes;
};

/**
 * CostBasisEntry - Cost basis record
 */
struct CostBasisEntry {
    std::string id;
    std::string asset;
    uint64_t quantity;
    double costPerUnit;
    double totalCost;
    std::string acquisitionDate;
    std::string lotId;
};

/**
 * CapitalGainLoss - Capital gains/losses calculation
 */
struct CapitalGainLoss {
    std::string asset;
    double proceeds;
    double costBasis;
    double gainLoss;
    std::string term;  // short_term, long_term
    std::string disposalDate;
    std::string lotId;
};

/**
 * TaxReport - Tax report summary
 */
struct TaxReport {
    std::string reportId;
    std::string walletAddress;
    int32_t taxYear;
    double totalProceeds;
    double totalCostBasis;
    double totalGainLoss;
    double shortTermGainLoss;
    double longTermGainLoss;
    double income;
    double stakingRewards;
    double interestIncome;
    double totalTaxableIncome;
    std::vector<CapitalGainLoss> transactions;
    std::map<std::string, double> gainsByAsset;
    std::map<std::string, double> incomeByType;
    std::chrono::system_clock::time_point generatedAt;
};

/**
 * TaxConfiguration - Tax settings
 */
struct TaxConfiguration {
    std::string method;  // FIFO, LIFO, HIFO, SPECIFIC_IDENTIFICATION
    std::string jurisdiction;  // US, UK, EU, etc.
    double shortTermRate;
    double longTermRate;
    double incomeTaxRate;
    bool includeStakingRewards;
    bool includeDeFiIncome;
    bool includeNFTs;
    bool applyWashSaleRules;
    std::vector<std::string> ignoredAssets;
};

/**
 * TaxAnalyticsService - Tax reporting and analytics
 */
class TaxAnalyticsService {
public:
    TaxAnalyticsService(const std::string& masterWalletId);
    ~TaxAnalyticsService();
    
    // Service lifecycle
    bool initialize();
    void shutdown();
    
    // Transaction tracking
    bool addTransaction(const TaxTransaction& transaction);
    bool addTransactions(const std::vector<TaxTransaction>& transactions);
    std::vector<TaxTransaction> getTransactions(
        const std::string& walletAddress,
        const std::string& startDate = "",
        const std::string& endDate = "",
        const std::string& type = ""
    );
    
    // Cost basis tracking
    std::string addCostBasis(const CostBasisEntry& entry);
    std::vector<CostBasisEntry> getCostBasis(
        const std::string& walletAddress,
        const std::string& asset
    );
    
    // Capital gains calculation
    std::vector<CapitalGainLoss> calculateGainsLosses(
        const std::string& walletAddress,
        const std::string& asset,
        int32_t taxYear
    );
    
    std::vector<CapitalGainLoss> calculateAllGainsLosses(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    // Tax reports
    std::string generateTaxReport(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    std::optional<TaxReport> getTaxReport(
        const std::string& reportId
    );
    
    std::vector<TaxReport> getTaxReports(
        const std::string& walletAddress,
        int32_t year = 0
    );
    
    // Configuration
    bool setConfiguration(const TaxConfiguration& config);
    TaxConfiguration getConfiguration();
    
    // Income tracking
    double calculateTotalIncome(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    double calculateStakingRewards(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    double calculateInterestIncome(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    double calculateDeFiIncome(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    // Export
    std::string exportToCSV(const std::string& reportId);
    std::string exportToJSON(const std::string& reportId);
    std::string exportToPDF(const std::string& reportId);
    
    // Statistics
    struct TaxStats {
        uint64_t totalTransactions;
        uint64_t totalReports;
        double totalGains;
        double totalLosses;
        double totalIncome;
        std::map<std::string, double> gainsByYear;
    };
    
    TaxStats getStats() const;

private:
    std::string masterWalletId_;
    std::map<std::string, std::vector<TaxTransaction>> transactions_;  // wallet -> transactions
    std::map<std::string, std::vector<CostBasisEntry>> costBasis_;  // wallet -> cost basis
    std::map<std::string, TaxReport> reports_;
    TaxConfiguration config_;
    
    mutable std::mutex dataMutex_;
    std::map<std::string, std::string> transactionCache_;
    
    // Private methods
    double calculateCostBasis(
        const std::string& walletAddress,
        const std::string& asset,
        uint64_t quantity,
        const std::string& method
    );
    
    std::vector<CostBasisEntry> getMatchingLots(
        const std::string& walletAddress,
        const std::string& asset,
        uint64_t quantity,
        const std::string& method
    );
    
    bool updateCostBasis(
        const std::string& walletAddress,
        const std::string& asset,
        uint64_t soldQuantity,
        const std::string& lotId
    );
    
    bool addIncome(
        const std::string& walletAddress,
        const std::string& incomeType,
        double amount,
        const std::string& date
    );
    
    std::map<std::string, double> getIncomeByType(
        const std::string& walletAddress,
        int32_t taxYear
    );
    
    std::string generateReportId();
    std::string formatDate(const std::chrono::system_clock::time_point& tp);
    
    double getCurrentPrice(const std::string& asset);
    double getHistoricalPrice(const std::string& asset, const std::string& date);
};

} // namespace tax
} // namespace master
} // namespace tiger

#endif // MASTER_WALLET_TAX_ANALYTICS_SERVICE_HPP
