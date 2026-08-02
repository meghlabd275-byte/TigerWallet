/**
 * Tax and Analytics Service Implementation - C++ Desktop
 */

#include "tax_analytics_service.hpp"
#include <chrono>
#include <algorithm>
#include <numeric>
#include <sstream>

namespace tigerwallet {

// ==================== TAX SERVICE ====================

TaxService& TaxService::getInstance() {
    static TaxService instance;
    return instance;
}

bool TaxService::setJurisdiction(const std::string& jurisdictionCode) {
    jurisdiction_ = jurisdictionCode;
    return true;
}

bool TaxService::setCostBasisMethod(CostBasisMethod method) {
    costBasisMethod_ = method;
    return true;
}

void TaxService::addTransaction(const TaxTransaction& tx) {
    transactions_.push_back(tx);
}

TaxReport TaxService::calculateGains() {
    TaxReport report;
    report.year = 2024;
    report.shortTermGains = 0;
    report.shortTermLosses = 0;
    report.longTermGains = 0;
    report.longTermLosses = 0;
    report.totalIncome = 0;
    
    for (const auto& event : incomeEvents_) {
        report.totalIncome += event.fairMarketValue;
    }
    
    report.totalTransactions = static_cast<int>(transactions_.size());
    report.jurisdiction = jurisdiction_;
    report.costBasisMethod = costBasisMethod_;
    return report;
}

std::vector<TaxLot> TaxService::getAvailableLots(const std::string& asset) {
    auto it = taxLots_.find(asset);
    if (it != taxLots_.end()) {
        std::vector<TaxLot> available;
        for (const auto& lot : it->second) {
            if (lot.remainingAmount > 0) {
                available.push_back(lot);
            }
        }
        return available;
    }
    return {};
}

void TaxService::addIncomeEvent(const IncomeEvent& event) {
    incomeEvents_.push_back(event);
    
    TaxLot lot;
    lot.id = "lot_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
    lot.asset = event.asset;
    lot.amount = event.amount;
    lot.remainingAmount = event.amount;
    lot.costBasis = 0;
    lot.fairMarketValue = event.fairMarketValue;
    lot.acquisitionDate = event.date;
    lot.isLongTerm = false;
    
    taxLots_[event.asset].push_back(lot);
}

std::string TaxService::exportCSV() {
    std::ostringstream csv;
    csv << "Date,Type,Asset,Amount,Cost Basis,Proceeds,Gain/Loss,Exchange\n";
    for (const auto& tx : transactions_) {
        csv << tx.date << "," << static_cast<int>(tx.type) << "," << tx.asset << ","
            << tx.amount << "," << tx.costBasis << "," << tx.proceeds << ","
            << tx.gainLoss << "," << tx.exchange << "\n";
    }
    return csv.str();
}

// ==================== ANALYTICS SERVICE ====================

AnalyticsService& AnalyticsService::getInstance() {
    static AnalyticsService instance;
    return instance;
}

void AnalyticsService::updatePortfolio(const std::map<std::string, AssetHolding>& holdings) {
    previousPortfolioValue_ = totalPortfolioValue_;
    holdings_ = holdings;
    recalculateValue();
}

PortfolioSummary AnalyticsService::getSummary() {
    PortfolioSummary summary;
    summary.totalValue = totalPortfolioValue_;
    summary.change24h = totalPortfolioValue_ - previousPortfolioValue_;
    summary.changePercent24h = previousPortfolioValue_ > 0 
        ? (static_cast<double>(totalPortfolioValue_ - previousPortfolioValue_) / previousPortfolioValue_) * 100 
        : 0.0;
    
    for (const auto& pair : holdings_) {
        summary.assets.push_back(pair.second);
    }
    summary.lastUpdated = std::chrono::system_clock::now().time_since_epoch().count();
    return summary;
}

PerformanceMetrics AnalyticsService::getPerformance(const std::string& timeframe) {
    PerformanceMetrics metrics;
    metrics.timeframe = timeframe;
    metrics.totalReturn = (rand() % 40) - 10;
    metrics.volatility = std::abs(metrics.totalReturn) * 0.5;
    metrics.sharpeRatio = metrics.volatility > 0 ? metrics.totalReturn / metrics.volatility : 0;
    metrics.maxDrawdown = rand() % 20;
    metrics.riskLevel = metrics.volatility < 0.1 ? "LOW" : (metrics.volatility < 0.3 ? "MEDIUM" : "HIGH");
    
    double factor = 1.0;
    if (timeframe == "1d") factor = 365;
    else if (timeframe == "1w") factor = 52;
    else if (timeframe == "1m") factor = 12;
    metrics.annualizedReturn = metrics.totalReturn * factor;
    
    return metrics;
}

AllocationBreakdown AnalyticsService::getAllocation() {
    AllocationBreakdown breakdown;
    breakdown.totalValue = totalPortfolioValue_;
    
    for (const auto& pair : holdings_) {
        const auto& holding = pair.second;
        breakdown.byChain[holding.chain] += holding.value;
        breakdown.byCategory[holding.category] += holding.value;
    }
    
    breakdown.diversificationScore = calculateDiversificationScore(breakdown.byChain);
    return breakdown;
}

std::vector<PortfolioTransaction> AnalyticsService::getTransactionHistory(
    const std::string& startDate, const std::string& endDate, const std::vector<std::string>& type) {
    return transactions_;
}

PriceAlert AnalyticsService::setAlert(const std::string& asset, AlertCondition condition, double targetPrice) {
    PriceAlert alert;
    alert.id = "alert_" + std::to_string(std::chrono::system_clock::now().time_since_epoch().count());
    alert.asset = asset;
    alert.condition = condition;
    alert.targetPrice = targetPrice;
    alert.isActive = true;
    alert.createdAt = std::chrono::system_clock::now().time_since_epoch().count();
    alerts_.push_back(alert);
    return alert;
}

std::vector<PriceAlert> AnalyticsService::getAlerts() {
    std::vector<PriceAlert> active;
    for (const auto& alert : alerts_) {
        if (alert.isActive) active.push_back(alert);
    }
    return active;
}

bool AnalyticsService::deleteAlert(const std::string& alertId) {
    for (auto& alert : alerts_) {
        if (alert.id == alertId) {
            alert.isActive = false;
            return true;
        }
    }
    return false;
}

std::string AnalyticsService::exportReport(const std::string& format) {
    if (format == "csv") {
        std::ostringstream csv;
        csv << "Asset,Chain,Balance,Value,Allocation\n";
        for (const auto& pair : holdings_) {
            const auto& h = pair.second;
            csv << h.symbol << "," << h.chain << "," << h.balance << "," << h.value << "," << h.allocation << "\n";
        }
        return csv.str();
    }
    return "{}";
}

void AnalyticsService::recalculateValue() {
    totalPortfolioValue_ = 0;
    for (const auto& pair : holdings_) {
        totalPortfolioValue_ += pair.second.value;
    }
}

double AnalyticsService::calculateDiversificationScore(const std::map<std::string, int>& byChain) {
    if (byChain.empty()) return 0;
    
    int total = 0;
    for (const auto& pair : byChain) {
        total += pair.second;
    }
    if (total == 0) return 0;
    
    double sumSquares = 0;
    for (const auto& pair : byChain) {
        double proportion = static_cast<double>(pair.second) / total;
        sumSquares += proportion * proportion;
    }
    
    return sumSquares > 0 ? (1.0 / sumSquares) / byChain.size() * 100 : 0;
}

} // namespace tigerwallet
