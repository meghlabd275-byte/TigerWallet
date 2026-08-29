/**
 * TigerWallet Tax Integration
 * 
 * Provides:
 * - Transaction categorization
 * - Capital gains calculation (FIFO, LIFO, HIFO)
 * - Tax report generation
 * - Multi-jurisdiction support
 * - Cost basis tracking
 * 
 * @author TigerWallet Team
 */

#ifndef TIGERWALLET_TAX_INTEGRATION_HPP
#define TIGERWALLET_TAX_INTEGRATION_HPP

#include <iostream>
#include <algorithm>
#include <string>
#include <vector>
#include <map>
#include <memory>
#include <variant>
#include <optional>
#include <functional>
#include <mutex>
#include <thread>
#include <chrono>
#include <ctime>
#include <cmath>

namespace tiger {

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

using TransactionID = std::string;
using WalletAddress = std::string;
using TokenSymbol = std::string;
using Jurisdiction = std::string; // US, UK, DE, JP, etc.

// Transaction Type
enum class TransactionType {
    BUY,
    SELL,
    TRANSFER_IN,
    TRANSFER_OUT,
    REWARD,
    STAKE_REWARD,
    AIRDROP,
    FEE,
    GIFT,
    DONATION,
    MINING
};

// Transaction
struct Transaction {
    TransactionID id;
    WalletAddress walletAddress;
    TokenSymbol token;
    TransactionType type;
    double amount;
    double priceAtTime; // USD value at transaction time
    double fee;
    std::string timestamp; // ISO 8601
    std::string txHash;
    std::string exchange; // For tax reporting
    std::string notes;
    bool isTaxable;
    bool isReported;
};

// Cost Basis Lot
struct CostBasisLot {
    std::string lotId;
    TokenSymbol token;
    double quantity;
    double costPerUnit;
    double totalCost;
    std::string acquisitionDate;
    std::string acquisitionTxHash;
    bool isLongTerm; // Held > 1 year
};

// Capital Gain/Loss
struct CapitalGain {
    std::string id;
    TransactionID sellTransactionId;
    std::string lotId;
    TokenSymbol token;
    double proceeds; // Sale proceeds
    double costBasis; // What was paid for the asset
    double gain; // proceeds - costBasis
    double feeAllocated;
    bool isLongTerm;
    std::string fiscalYear;
    std::string calculationMethod; // FIFO, LIFO, HIFO
};

// Tax Report
struct TaxReport {
    std::string id;
    Jurisdiction jurisdiction;
    std::string taxYear;
    double totalProceeds;
    double totalCostBasis;
    double totalGain;
    double totalLoss;
    double netGainLoss;
    double shortTermGain;
    double longTermGain;
    double shortTermLoss;
    double longTermLoss;
    double totalFees;
    std::vector<CapitalGain> gains;
    std::map<TokenSymbol, double> holdingsByToken;
    std::string generatedAt;
    std::string format; // PDF, CSV, JSON
};

// Tax Jurisdiction Rules
struct TaxJurisdiction {
    Jurisdiction code;
    std::string name;
    std::string currency;
    double shortTermTaxRate; // < 1 year
    double longTermTaxRate; // > 1 year
    bool hasWashSaleRule;
    int washSalePeriodDays;
    bool allowsIFO;
    bool allowsCryptoToCrypto; // Taxable event
    double deMinimisThreshold; // Below this, no tax
    std::vector<std::string> exemptTokens;
};

// Calculation Method
enum class CostBasisMethod {
    FIFO, // First In, First Out
    LIFO, // Last In, First Out
    HIFO, // Highest In, First Out
    SPECIFIC_IDENTIFICATION
};

// =============================================================================
// COST BASIS CALCULATOR
// =============================================================================

class CostBasisCalculator {
public:
    CostBasisCalculator(CostBasisMethod method = CostBasisMethod::FIFO) 
        : method_(method) {}
    
    // Set calculation method
    void setMethod(CostBasisMethod method) {
        method_ = method;
    }
    
    // Add acquisition (buy) transaction to lots
    void addAcquisition(const CostBasisLot& lot) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        // Check if we have existing lots for this token
        auto& lots = tokenLots_[lot.token];
        
        // Add new lot
        lots.push_back(lot);
    }
    
    // Calculate cost basis for a sale
    std::optional<CapitalGain> calculateGain(
        const Transaction& sellTransaction,
        double quantitySold
    ) {
        return calculateGainWithLots(sellTransaction, quantitySold, {});
    }

    // Calculate cost basis with user-designated lots (required for
    // SPECIFIC_IDENTIFICATION: lots are consumed in exactly the order given).
    std::optional<CapitalGain> calculateGainWithLots(
        const Transaction& sellTransaction,
        double quantitySold,
        const std::vector<std::string>& specifiedLotIds
    ) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = tokenLots_.find(sellTransaction.token);
        if (it == tokenLots_.end() || it->second.empty()) {
            return std::nullopt;
        }
        
        auto& lots = it->second;
        double remainingToSell = quantitySold;
        double totalCostBasis = 0;
        std::vector<std::string> lotIdsUsed;
        
        // Sort lots based on method
        std::vector<CostBasisLot*> sortedLots;
        for (auto& lot : lots) {
            sortedLots.push_back(&lot);
        }
        
        // Sort based on method
        switch (method_) {
            case CostBasisMethod::FIFO:
                // Already sorted by acquisition date
                break;
            case CostBasisMethod::LIFO:
                std::reverse(sortedLots.begin(), sortedLots.end());
                break;
            case CostBasisMethod::HIFO:
                std::sort(sortedLots.begin(), sortedLots.end(),
                    [](const CostBasisLot* a, const CostBasisLot* b) {
                        return a->costPerUnit > b->costPerUnit;
                    });
                break;
            case CostBasisMethod::SPECIFIC_IDENTIFICATION: {
                // Specific identification requires the user to designate which
                // lots to sell. Without designated lots we cannot compute an
                // honest basis — fail closed rather than silently computing a
                // FIFO gain under the wrong method.
                if (specifiedLotIds.empty()) {
                    return std::nullopt;
                }
                std::vector<CostBasisLot*> designated;
                for (const auto& wantedId : specifiedLotIds) {
                    for (auto* lot : sortedLots) {
                        if (lot->lotId == wantedId) {
                            designated.push_back(lot);
                            break;
                        }
                    }
                }
                sortedLots = designated;
                break;
            }
        }
        
        // Calculate cost basis
        for (auto* lot : sortedLots) {
            if (remainingToSell <= 0) break;
            
            double quantityFromThisLot = std::min(remainingToSell, lot->quantity);
            totalCostBasis += quantityFromThisLot * lot->costPerUnit;
            lotIdsUsed.push_back(lot->lotId);
            remainingToSell -= quantityFromThisLot;
            
            // Reduce lot quantity
            lot->quantity -= quantityFromThisLot;
        }
        
        if (remainingToSell > 0) {
            // Not enough lots to cover sale
            return std::nullopt;
        }
        
        // Calculate proceeds
        double proceeds = quantitySold * sellTransaction.priceAtTime;
        
        // Calculate gain
        double gain = proceeds - totalCostBasis;
        
        // Determine if long-term
        bool isLongTerm = isLongTermGain(sortedLots[0]->acquisitionDate);
        
        // Create capital gain record
        CapitalGain capGain;
        capGain.id = generateId();
        capGain.sellTransactionId = sellTransaction.id;
        capGain.lotId = lotIdsUsed[0]; // Primary lot
        capGain.token = sellTransaction.token;
        capGain.proceeds = proceeds;
        capGain.costBasis = totalCostBasis;
        capGain.gain = gain;
        capGain.feeAllocated = sellTransaction.fee * (quantitySold / sellTransaction.amount);
        capGain.isLongTerm = isLongTerm;
        capGain.fiscalYear = getFiscalYear(sellTransaction.timestamp);
        capGain.calculationMethod = methodToString();
        
        return capGain;
    }
    
    // Get current holdings
    std::map<TokenSymbol, double> getHoldings() {
        std::lock_guard<std::mutex> lock(mutex_);
        
        std::map<TokenSymbol, double> holdings;
        
        for (const auto& [token, lots] : tokenLots_) {
            double total = 0;
            for (const auto& lot : lots) {
                total += lot.quantity;
            }
            if (total > 0) {
                holdings[token] = total;
            }
        }
        
        return holdings;
    }
    
    // Get lot details for a token
    std::vector<CostBasisLot> getLotsForToken(const TokenSymbol& token) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = tokenLots_.find(token);
        if (it != tokenLots_.end()) {
            return it->second;
        }
        
        return {};
    }
    
private:
    CostBasisMethod method_;
    std::mutex mutex_;
    std::map<TokenSymbol, std::vector<CostBasisLot>> tokenLots_;
    
    bool isLongTermGain(const std::string& acquisitionDate) {
        // Parse date and check if > 1 year
        // Simplified - check year difference
        int acqYear = std::stoi(acquisitionDate.substr(0, 4));
        int currYear = 2026;
        
        return (currYear - acqYear) >= 1;
    }
    
    std::string getFiscalYear(const std::string& timestamp) {
        // Extract year from timestamp
        return timestamp.substr(0, 4);
    }
    
    std::string methodToString() {
        switch (method_) {
            case CostBasisMethod::FIFO: return "FIFO";
            case CostBasisMethod::LIFO: return "LIFO";
            case CostBasisMethod::HIFO: return "HIFO";
            case CostBasisMethod::SPECIFIC_IDENTIFICATION: return "Specific Identification";
            default: return "Unknown";
        }
    }
    
    std::string generateId() {
        return "gain_" + std::to_string(time(nullptr));
    }
};

// =============================================================================
// TAX REPORT GENERATOR
// =============================================================================

class TaxReportGenerator {
public:
    TaxReportGenerator() : calculator_(CostBasisMethod::FIFO) {}
    
    // Set jurisdiction
    void setJurisdiction(const Jurisdiction& jurisdiction) {
        jurisdiction_ = jurisdiction;
        loadJurisdictionRules();
    }
    
    // Set cost basis method
    void setCostBasisMethod(CostBasisMethod method) {
        calculator_.setMethod(method);
    }
    
    // Process all transactions
    void processTransactions(const std::vector<Transaction>& transactions) {
        // Separate buys and sells
        for (const auto& tx : transactions) {
            if (tx.type == TransactionType::BUY || 
                tx.type == TransactionType::TRANSFER_IN ||
                tx.type == TransactionType::REWARD ||
                tx.type == TransactionType::STAKE_REWARD ||
                tx.type == TransactionType::AIRDROP) {
                
                // Add to cost basis lots
                CostBasisLot lot;
                lot.lotId = generateId();
                lot.token = tx.token;
                lot.quantity = tx.amount;
                lot.costPerUnit = tx.priceAtTime;
                lot.totalCost = tx.amount * tx.priceAtTime;
                lot.acquisitionDate = tx.timestamp;
                lot.acquisitionTxHash = tx.txHash;
                
                calculator_.addAcquisition(lot);
                
                // Record income for rewards
                if (tx.type == TransactionType::REWARD || 
                    tx.type == TransactionType::STAKE_REWARD ||
                    tx.type == TransactionType::AIRDROP) {
                    income_[tx.id] = tx;
                }
            }
        }
        
        // Process sells
        for (const auto& tx : transactions) {
            if (tx.type == TransactionType::SELL) {
                auto gain = calculator_.calculateGain(tx, tx.amount);
                if (gain.has_value()) {
                    capitalGains_.push_back(gain.value());
                }
            }
        }
    }
    
    // Generate tax report
    TaxReport generateReport(const std::string& taxYear) {
        TaxReport report;
        report.id = generateId();
        report.jurisdiction = jurisdiction_;
        report.taxYear = taxYear;
        
        // Calculate totals
        double totalProceeds = 0;
        double totalCostBasis = 0;
        double totalGain = 0;
        double totalLoss = 0;
        double shortTermGain = 0;
        double longTermGain = 0;
        double shortTermLoss = 0;
        double longTermLoss = 0;
        double totalFees = 0;
        
        for (const auto& gain : capitalGains_) {
            totalProceeds += gain.proceeds;
            totalCostBasis += gain.costBasis;
            totalFees += gain.feeAllocated;
            
            if (gain.gain >= 0) {
                totalGain += gain.gain;
                if (gain.isLongTerm) {
                    longTermGain += gain.gain;
                } else {
                    shortTermGain += gain.gain;
                }
            } else {
                totalLoss += -gain.gain;
                if (gain.isLongTerm) {
                    longTermLoss += -gain.gain;
                } else {
                    shortTermLoss += -gain.gain;
                }
            }
        }
        
        report.totalProceeds = totalProceeds;
        report.totalCostBasis = totalCostBasis;
        report.totalGain = totalGain;
        report.totalLoss = totalLoss;
        report.netGainLoss = totalGain - totalLoss;
        report.shortTermGain = shortTermGain;
        report.longTermGain = longTermGain;
        report.shortTermLoss = shortTermLoss;
        report.longTermLoss = longTermLoss;
        report.totalFees = totalFees;
        report.gains = capitalGains_;
        report.holdingsByToken = calculator_.getHoldings();
        report.generatedAt = getCurrentTimestamp();
        
        return report;
    }
    
    // Export to CSV
    std::string exportToCSV(const TaxReport& report) {
        std::string csv;
        
        // Header
        csv += "Tax Year,Jurisdiction,Token,Proceeds,Cost Basis,Gain/Loss,Type,Date,Lot ID,Calculation Method\n";
        
        // Data rows
        for (const auto& gain : report.gains) {
            csv += report.taxYear + ",";
            csv += report.jurisdiction + ",";
            csv += gain.token + ",";
            csv += std::to_string(gain.proceeds) + ",";
            csv += std::to_string(gain.costBasis) + ",";
            csv += std::to_string(gain.gain) + ",";
            csv += (gain.isLongTerm ? "Long-term," : "Short-term,");
            csv += gain.fiscalYear + ",";
            csv += gain.lotId + ",";
            csv += gain.calculationMethod + "\n";
        }
        
        // Summary
        csv += "\nSummary\n";
        csv += "Total Proceeds," + std::to_string(report.totalProceeds) + "\n";
        csv += "Total Cost Basis," + std::to_string(report.totalCostBasis) + "\n";
        csv += "Net Gain/Loss," + std::to_string(report.netGainLoss) + "\n";
        csv += "Short-term Gain," + std::to_string(report.shortTermGain) + "\n";
        csv += "Long-term Gain," + std::to_string(report.longTermGain) + "\n";
        csv += "Short-term Loss," + std::to_string(report.shortTermLoss) + "\n";
        csv += "Long-term Loss," + std::to_string(report.longTermLoss) + "\n";
        
        return csv;
    }
    
    // Export to JSON
    std::string exportToJSON(const TaxReport& report) {
        // Simplified JSON output
        std::string json = "{\n";
        json += "  \"report_id\": \"" + report.id + "\",\n";
        json += "  \"jurisdiction\": \"" + report.jurisdiction + "\",\n";
        json += "  \"tax_year\": \"" + report.taxYear + "\",\n";
        json += "  \"total_proceeds\": " + std::to_string(report.totalProceeds) + ",\n";
        json += "  \"total_cost_basis\": " + std::to_string(report.totalCostBasis) + ",\n";
        json += "  \"net_gain_loss\": " + std::to_string(report.netGainLoss) + ",\n";
        json += "  \"gains\": " + std::to_string(report.gains.size()) + "\n";
        json += "}";
        
        return json;
    }
    
    // Get income transactions
    const std::map<TransactionID, Transaction>& getIncome() const {
        return income_;
    }
    
    // Get capital gains
    const std::vector<CapitalGain>& getCapitalGains() const {
        return capitalGains_;
    }
    
private:
    Jurisdiction jurisdiction_;
    TaxJurisdiction jurisdictionRules_;
    CostBasisCalculator calculator_;
    std::map<TransactionID, Transaction> income_;
    std::vector<CapitalGain> capitalGains_;
    
    void loadJurisdictionRules() {
        // Load tax rules for jurisdiction
        jurisdictionRules_ = getJurisdictionRules(jurisdiction_);
    }
    
    TaxJurisdiction getJurisdictionRules(const Jurisdiction& code) {
        // Default US rules
        TaxJurisdiction rules;
        rules.code = code;
        rules.name = "United States";
        rules.currency = "USD";
        rules.shortTermTaxRate = 0.37; // Max federal bracket
        rules.longTermTaxRate = 0.20;
        rules.hasWashSaleRule = true;
        rules.washSalePeriodDays = 30;
        rules.allowsCryptoToCrypto = true;
        rules.deMinimisThreshold = 0;
        
        if (code == "UK") {
            rules.name = "United Kingdom";
            rules.currency = "GBP";
            rules.shortTermTaxRate = 0.20;
            rules.longTermTaxRate = 0.20;
            rules.hasWashSaleRule = false;
        } else if (code == "DE") {
            rules.name = "Germany";
            rules.currency = "EUR";
            rules.shortTermTaxRate = 0.45;
            rules.longTermTaxRate = 0; // 1 year holding = tax free
            rules.hasWashSaleRule = false;
            rules.allowsCryptoToCrypto = false;
        }
        
        return rules;
    }
    
    std::string generateId() {
        return "tax_" + std::to_string(time(nullptr));
    }
    
    std::string getCurrentTimestamp() {
        time_t now = time(nullptr);
        char buf[32];
        strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", gmtime(&now));
        return std::string(buf);
    }
};

// =============================================================================
// TRANSACTION CATEGORIZER
// =============================================================================

class TransactionCategorizer {
public:
    TransactionCategorizer() {
        // Initialize default categories
        categories_ = {
            {"income", {"reward", "staking", "airdrop", "mining", "interest"}},
            {"expenses", {"fee", "gas", "swap", "trade"}},
            {"transfers", {"send", "receive", "transfer"}},
            {"gifts", {"gift", "donation"}},
            {"personal", {"nft_purchase", "nft_sale"}}
        };
        
        // Initialize exchange mappings
        exchangeMapping_ = {
            {"binance.com", "Binance"},
            {"coinbase.com", "Coinbase"},
            {"kraken.com", "Kraken"},
            {"uniswap.org", "Uniswap"},
            {"opensea.io", "OpenSea"}
        };
    }
    
    // Auto-categorize transaction
    TransactionType categorize(const Transaction& tx) {
        // Check transaction type based on amount direction and exchange
        if (tx.amount > 0) {
            // Check if from exchange
            for (const auto& [domain, exchange] : exchangeMapping_) {
                if (tx.exchange.find(domain) != std::string::npos) {
                    return TransactionType::BUY;
                }
            }
            
            // Check notes/tags
            if (tx.notes.find("reward") != std::string::npos ||
                tx.notes.find("staking") != std::string::npos) {
                return TransactionType::STAKE_REWARD;
            }
            
            if (tx.notes.find("airdrop") != std::string::npos) {
                return TransactionType::AIRDROP;
            }
            
            return TransactionType::TRANSFER_IN;
        } else {
            return TransactionType::SELL;
        }
    }
    
    // Check if transaction is taxable
    bool isTaxable(const Transaction& tx) {
        // In most jurisdictions, these are taxable
        auto type = categorize(tx);
        
        return type == TransactionType::SELL ||
               type == TransactionType::STAKE_REWARD ||
               type == TransactionType::AIRDROP ||
               type == TransactionType::REWARD;
    }
    
    // Categorize exchange
    std::string categorizeExchange(const std::string& txHash) {
        // In production, query blockchain to determine exchange
        // For now, return unknown
        return "Unknown";
    }
    
    // Set custom category rules
    void addCategoryRule(const std::string& category, const std::string& keyword) {
        categories_[category].push_back(keyword);
    }
    
private:
    std::map<std::string, std::vector<std::string>> categories_;
    std::map<std::string, std::string> exchangeMapping_;
};

// =============================================================================
// TAX MANAGER (Master Orchestrator)
// =============================================================================

class TaxManager {
public:
    TaxManager() 
        : reportGenerator_(),
          categorizer_() {}
    
    // Initialize
    bool initialize() {
        return true;
    }
    
    // Add transaction
    void addTransaction(const Transaction& transaction) {
        transactions_.push_back(transaction);
        
        // Auto-categorize
        auto type = categorizer_.categorize(transaction);
        
        // If buy/acquisition, add to cost basis
        if (type == TransactionType::BUY || type == TransactionType::TRANSFER_IN) {
            CostBasisLot lot;
            lot.lotId = generateId();
            lot.token = transaction.token;
            lot.quantity = transaction.amount;
            lot.costPerUnit = transaction.priceAtTime;
            lot.totalCost = transaction.amount * transaction.priceAtTime;
            lot.acquisitionDate = transaction.timestamp;
            lot.acquisitionTxHash = transaction.txHash;
            
            reportGenerator_.setCostBasisMethod(costBasisMethod_);
            // Note: In production, we'd need to add acquisition to calculator
        }
    }
    
    // Add transactions in batch
    void addTransactions(const std::vector<Transaction>& transactions) {
        for (const auto& tx : transactions) {
            addTransaction(tx);
        }
    }
    
    // Set jurisdiction
    void setJurisdiction(const Jurisdiction& jurisdiction) {
        jurisdiction_ = jurisdiction;
        reportGenerator_.setJurisdiction(jurisdiction);
    }
    
    // Set cost basis method
    void setCostBasisMethod(CostBasisMethod method) {
        costBasisMethod_ = method;
    }
    
    // Generate report
    TaxReport generateReport(const std::string& taxYear) {
        reportGenerator_.processTransactions(transactions_);
        return reportGenerator_.generateReport(taxYear);
    }
    
    // Export report
    std::string exportReport(const TaxReport& report, const std::string& format) {
        if (format == "csv") {
            return reportGenerator_.exportToCSV(report);
        } else if (format == "json") {
            return reportGenerator_.exportToJSON(report);
        }
        
        return "";
    }
    
    // Get transactions
    const std::vector<Transaction>& getTransactions() const {
        return transactions_;
    }
    
    // Filter transactions
    std::vector<Transaction> filterTransactions(
        std::function<bool(const Transaction&)> predicate
    ) {
        std::vector<Transaction> result;
        
        for (const auto& tx : transactions_) {
            if (predicate(tx)) {
                result.push_back(tx);
            }
        }
        
        return result;
    }
    
    // Calculate unrealized gains
    std::map<TokenSymbol, double> calculateUnrealizedGains(
        const std::map<TokenSymbol, double>& currentPrices
    ) {
        std::map<TokenSymbol, double> unrealized;
        
        // Calculate for each token
        // Note: Need access to cost basis calculator for this
        
        return unrealized;
    }
    
private:
    std::vector<Transaction> transactions_;
    Jurisdiction jurisdiction_;
    CostBasisMethod costBasisMethod_ = CostBasisMethod::FIFO;
    TaxReportGenerator reportGenerator_;
    TransactionCategorizer categorizer_;
    
    std::string generateId() {
        return "tx_" + std::to_string(time(nullptr));
    }
};

} // namespace tiger

#endif // TIGERWALLET_TAX_INTEGRATION_HPP
