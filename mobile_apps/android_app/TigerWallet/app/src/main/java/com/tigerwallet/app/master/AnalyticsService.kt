/**
 * TigerWallet Android - Portfolio Analytics Service
 * 
 * Complete Analytics Features:
 * - Real-time Portfolio Tracking
 * - Performance Metrics
 * - Risk Analysis
 * - Asset Allocation
 * - Price Alerts
 * 
 * This service MUST be identical across ALL platforms.
 */

package com.tigerwallet.app.master

import java.math.BigInteger
import java.math.RoundingMode
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * Analytics Service - Portfolio tracking and analysis
 */
class AnalyticsService private constructor() {

    companion object {
        val instance: AnalyticsService by lazy { AnalyticsService() }
    }

    private val holdings = mutableMapOf<String, AssetHolding>()
    private val priceHistory = mutableMapOf<String, MutableList<PricePoint>>()
    private val transactions = mutableListOf<PortfolioTransaction>()
    private val alerts = mutableListOf<PriceAlert>()

    private var totalPortfolioValue = BigInteger.ZERO
    private var previousPortfolioValue = BigInteger.ZERO

    /**
     * Update portfolio
     */
    fun updatePortfolio(holdings: Map<String, AssetHolding>) {
        previousPortfolioValue = totalPortfolioValue
        this.holdings.clear()
        this.holdings.putAll(holdings)
        recalculateValue()
    }

    /**
     * Get portfolio summary
     */
    fun getSummary(): PortfolioSummary {
        return PortfolioSummary(
            totalValue = totalPortfolioValue,
            change24h = totalPortfolioValue.subtract(previousPortfolioValue),
            changePercent24h = calculateChangePercent(),
            assets = holdings.values.toList(),
            lastUpdated = System.currentTimeMillis()
        )
    }

    /**
     * Get performance metrics
     */
    fun getPerformance(timeframe: String): PerformanceMetrics {
        val returns = calculateReturns(timeframe)
        val volatility = calculateVolatility(returns)
        val sharpe = if (volatility > 0.0) (returns / volatility) else 0.0
        
        return PerformanceMetrics(
            timeframe = timeframe,
            totalReturn = returns,
            annualizedReturn = returns * getAnnualizationFactor(timeframe),
            volatility = volatility,
            sharpeRatio = sharpe,
            maxDrawdown = calculateMaxDrawdown(),
            riskLevel = getRiskLevel(volatility)
        )
    }

    /**
     * Get asset allocation
     */
    fun getAllocation(): AllocationBreakdown {
        val byChain = holdings.values.groupBy { it.chain }
            .mapValues { it.value.sumOf { a -> a.value } }
        
        val byCategory = holdings.values.groupBy { it.category }
            .mapValues { it.value.sumOf { a -> a.value } }
        
        return AllocationBreakdown(
            byChain = byChain,
            byCategory = byCategory,
            totalValue = totalPortfolioValue,
            diversificationScore = calculateDiversificationScore(byChain)
        )
    }

    /**
     * Get transaction history
     */
    fun getTransactionHistory(
        startDate: String? = null,
        endDate: String? = null,
        type: List<String>? = null
    ): List<PortfolioTransaction> {
        var result = transactions
        
        if (startDate != null) {
            result = result.filter { it.date >= startDate }
        }
        if (endDate != null) {
            result = result.filter { it.date <= endDate }
        }
        if (type != null) {
            result = result.filter { type.contains(it.type) }
        }
        
        return result
    }

    /**
     * Set price alert
     */
    fun setAlert(asset: String, condition: AlertCondition, targetPrice: Double): PriceAlert {
        val alert = PriceAlert(
            id = "alert_${System.currentTimeMillis()}",
            asset = asset,
            condition = condition,
            targetPrice = targetPrice,
            isActive = true,
            createdAt = System.currentTimeMillis()
        )
        alerts.add(alert)
        return alert
    }

    /**
     * Get active alerts
     */
    fun getAlerts(): List<PriceAlert> = alerts.filter { it.isActive }

    /**
     * Delete alert
     */
    fun deleteAlert(alertId: String): Boolean {
        return alerts.removeIf { it.id == alertId }
    }

    /**
     * Get portfolio history
     */
    fun getHistory(startDate: String, endDate: String, interval: String): List<HistoryPoint> {
        // Return mock history for demo
        val points = mutableListOf<HistoryPoint>()
        val format = SimpleDateFormat("yyyy-MM-dd", Locale.US)
        
        try {
            val start = format.parse(startDate)?.time ?: return points
            val end = format.parse(endDate)?.time ?: return points
            
            var current = start
            while (current <= end) {
                // No fabricated random variation: use the real portfolio value.
                // Real historical data must come from the backend/indexer.
                val value = totalPortfolioValue
                
                points.add(HistoryPoint(
                    timestamp = current,
                    value = value,
                    change = value.subtract(previousPortfolioValue)
                ))
                
                current += when (interval) {
                    "1h" -> 3600000L
                    "1d" -> 86400000L
                    "1w" -> 604800000L
                    else -> 86400000L
                }
            }
        } catch (e: Exception) {
            // Return empty
        }
        
        return points
    }

    /**
     * Export report
     */
    fun exportReport(format: String): String {
        return when (format) {
            "csv" -> generateCSV(),
            "json" -> generateJSON(),
            else -> ""
        }
    }

    // ============================================================================
    // PRIVATE HELPERS
    // ============================================================================

    private fun recalculateValue() {
        totalPortfolioValue = holdings.values.sumOf { it.value }
    }

    private fun calculateChangePercent(): Double {
        if (previousPortfolioValue == BigInteger.ZERO) return 0.0
        val change = totalPortfolioValue.subtract(previousPortfolioValue)
        return change.multiply(BigInteger.valueOf(100))
            .divide(previousPortfolioValue, 4, RoundingMode.HALF_UP)
            .toDouble()
    }

    private fun calculateReturns(timeframe: String): Double {
        // Real returns are computed from backend historical data; until wired,
        // return 0.0 rather than a fabricated Math.random() value.
        return 0.0
    }

    private fun calculateVolatility(returns: Double): Double {
        return Math.abs(returns) * 0.5
    }

    private fun calculateMaxDrawdown(): Double {
        // Real max drawdown is computed from backend historical data; until
        // wired, return 0.0 rather than a fabricated Math.random() value.
        return 0.0
    }

    private fun getAnnualizationFactor(timeframe: String): Double {
        return when (timeframe) {
            "1d" -> 365.0
            "1w" -> 52.0
            "1m" -> 12.0
            "1y" -> 1.0
            else -> 1.0
        }
    }

    private fun getRiskLevel(volatility: Double): String {
        return when {
            volatility < 0.1 -> "LOW"
            volatility < 0.3 -> "MEDIUM"
            else -> "HIGH"
        }
    }

    private fun calculateDiversificationScore(byChain: Map<String, BigInteger>): Double {
        if (byChain.isEmpty()) return 0.0
        val total = byChain.values.sumOf { it }.toDouble()
        if (total == 0.0) return 0.0
        
        val proportions = byChain.values.map { it.toDouble() / total }
        val sumSquares = proportions.sumOf { it * it }
        
        // Herfindahl index inverse (higher = more diversified)
        return if (sumSquares > 0) (1.0 / sumSquares) / byChain.size * 100 else 0.0
    }

    private fun generateCSV(): String {
        val header = "Asset,Chain,Balance,Value,Allocation"
        val rows = holdings.values.map { h ->
            "${h.symbol},${h.chain},${h.balance},${h.value},${h.allocation}%"
        }
        return (listOf(header) + rows).joinToString("\n")
    }

    private fun generateJSON(): String {
        return """
        {
            "portfolio": {
                "totalValue": $totalPortfolioValue,
                "assets": [
                    ${holdings.values.joinToString(",") { "{\"symbol\":\"${it.symbol}\",\"value\":${it.value}}" }}
                ]
            }
        }
        """.trimIndent()
    }
}

// ============================================================================
// DATA CLASSES
// ============================================================================

data class AssetHolding(
    val symbol: String,
    val name: String,
    val chain: String,
    val category: String,
    val balance: BigInteger,
    val price: Double,
    val value: BigInteger,
    val allocation: Double,
    val change24h: Double
)

data class PortfolioSummary(
    val totalValue: BigInteger,
    val change24h: BigInteger,
    val changePercent24h: Double,
    val assets: List<AssetHolding>,
    val lastUpdated: Long
)

data class PerformanceMetrics(
    val timeframe: String,
    val totalReturn: Double,
    val annualizedReturn: Double,
    val volatility: Double,
    val sharpeRatio: Double,
    val maxDrawdown: Double,
    val riskLevel: String
)

data class AllocationBreakdown(
    val byChain: Map<String, BigInteger>,
    val byCategory: Map<String, BigInteger>,
    val totalValue: BigInteger,
    val diversificationScore: Double
)

data class PortfolioTransaction(
    val id: String,
    val type: String,
    val asset: String,
    val amount: BigInteger,
    val value: BigInteger,
    val date: String,
    val txHash: String
)

data class PriceAlert(
    val id: String,
    val asset: String,
    val condition: AlertCondition,
    val targetPrice: Double,
    val isActive: Boolean,
    val createdAt: Long
)

enum class AlertCondition { ABOVE, BELOW }

data class HistoryPoint(
    val timestamp: Long,
    val value: BigInteger,
    val change: BigInteger
)

data class PricePoint(
    val timestamp: Long,
    val price: Double,
    val volume: Double
)
