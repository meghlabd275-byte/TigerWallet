/**
 * TigerWallet TradingView Charts Integration
 * 
 * Provides advanced charting with:
 * - Real-time price data
 * - Technical indicators
 * - Drawing tools
 * - Multiple timeframes
 * - Cross-chain support
 * 
 * @author TigerWallet Team
 */

#ifndef TIGERWALLET_TRADINGVIEW_HPP
#define TIGERWALLET_TRADINGVIEW_HPP

#include <iostream>
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
#include <curl/curl.h>
#include <random>
#include <cctype>
#include <nlohmann/json.hpp>

namespace tiger {

using json = nlohmann::json;

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

using Symbol = std::string;
using Interval = std::string; // 1m, 5m, 15m, 1h, 4h, 1D, 1W, 1M

// OHLCV Data
struct Candle {
    uint64_t time;       // Unix timestamp (seconds)
    double open;
    double high;
    double low;
    double close;
    double volume;
};

// Technical Indicator
enum class IndicatorType {
    SMA,        // Simple Moving Average
    EMA,        // Exponential Moving Average
    RSI,        // Relative Strength Index
    MACD,       // Moving Average Convergence Divergence
    BOLLINGER,  // Bollinger Bands
    VWAP,       // Volume Weighted Average Price
    ATR,        // Average True Range
    STOCH,      // Stochastic Oscillator
    ADX,        // Average Directional Index
    CCI,        // Commodity Channel Index
    WILLR,      // Williams %R
    OBV,        // On Balance Volume
    VOLUME      // Volume Profile
};

// Indicator Config
struct IndicatorConfig {
    IndicatorType type;
    std::string name;
    std::map<std::string, double> params; // period, overbought, oversold, etc.
    std::string color;
    int lineWidth;
    bool enabled;
};

// Chart Config
struct ChartConfig {
    Symbol symbol;
    Interval interval;
    std::string theme; // "light", "dark"
    std::string style; // "candles", "line", "area"
    bool showVolume;
    bool showGrid;
    bool showCrosshair;
    bool showLegend;
    int width;
    int height;
    std::string timezone;
    std::vector<IndicatorConfig> indicators_; // active indicators
};

// Drawing Tool
enum class DrawingType {
    TRENDLINE,
    HORIZONTAL_LINE,
    VERTICAL_LINE,
    FIBONACCI_RETRACEMENT,
    FIBONACCI_EXTENSION,
    CHANNEL,
    RAY,
    POLYGON,
    TEXT,
    SHAPE,
    PITCHFORK
};

struct Drawing {
    std::string id;
    DrawingType type;
    std::string name;
    std::vector<std::pair<double, double>> points; // price/time coordinates
    std::string color;
    int lineWidth;
    std::string lineStyle; // solid, dotted, dashed
    bool isVisible;
    bool lock_;
    bool isLocked;
};

// Chart Widget
class TradingViewChart {
public:
    TradingViewChart(const ChartConfig& config) : config_(config) {
        containerId_ = "tv_chart_" + generateId();
        widgets_[containerId_] = this;
    }
    
    // Initialize chart
    bool initialize() {
        std::cout << "Initializing TradingView Chart for " << config_.symbol << std::endl;
        
        // Load historical data
        loadHistoricalData();
        
        return true;
    }
    
    // Get HTML embed code
    std::string getEmbedCode() {
        std::string code = R"(
<div id=")" + containerId_ + R"("></div>
<script>
    new TradingView.widget({
        "width": )" + std::to_string(config_.width) + R"(
        "height": )" + std::to_string(config_.height) + R"(
        "symbol": ")" + config_.symbol + R"(",
        "interval": ")" + config_.interval + R"(",
        "timezone": ")" + config_.timezone + R"(
        "theme": ")" + config_.theme + R"(
        "style": ")" + config_.style + R"(
        "locale": "en",
        "toolbar_bg": ")" + (config_.theme == "dark" ? "#1a1a2e" : "#f5f7fa") + R"(
        "enable_publishing": false,
        "hide_side_toolbar": false,
        "allow_symbol_change": true,
        "container_id": ")" + containerId_ + R"("
    });
</script>
)";
        return code;
    }
    
    // Set symbol
    void setSymbol(const Symbol& symbol) {
        config_.symbol = symbol;
        loadHistoricalData();
    }
    
    // Set interval
    void setInterval(const Interval& interval) {
        config_.interval = interval;
        loadHistoricalData();
    }
    
    // Set theme
    void setTheme(const std::string& theme) {
        config_.theme = theme;
    }
    
    // Add indicator
    std::string addIndicator(const IndicatorConfig& indicator) {
        config_.indicators_.push_back(indicator);
        return indicator.name;
    }
    
    // Remove indicator
    void removeIndicator(const std::string& name) {
        config_.indicators_.erase(
            std::remove_if(config_.indicators_.begin(), config_.indicators_.end(),
                [&name](const IndicatorConfig& ind) { return ind.name == name; }),
            config_.indicators_.end()
        );
    }
    
    // Add drawing
    std::string addDrawing(const Drawing& drawing) {
        std::string id = generateId();
        drawings_[id] = drawing;
        return id;
    }
    
    // Remove drawing
    void removeDrawing(const std::string& id) {
        drawings_.erase(id);
    }
    
    // Update drawing
    void updateDrawing(const Drawing& drawing) {
        drawings_[drawing.id] = drawing;
    }
    
    // Get historical data
    const std::vector<Candle>& getHistoricalData() const {
        return historicalData_;
    }
    
    // Get current candle
    std::optional<Candle> getCurrentCandle() const {
        if (!historicalData_.empty()) {
            return historicalData_.back();
        }
        return std::nullopt;
    }
    
    // Subscribe to real-time updates
    void subscribeToUpdates(std::function<void(const Candle&)> callback) {
        updateCallbacks_.push_back(callback);
    }
    
    // Unsubscribe from updates
    void unsubscribe() {
        updateCallbacks_.clear();
    }
    
    // Get container ID
    std::string getContainerId() const { return containerId_; }
    
    // Get config
    const ChartConfig& getConfig() const { return config_; }
    
private:
    ChartConfig config_;
    std::string containerId_;
    std::vector<Candle> historicalData_;
    std::map<std::string, Drawing> drawings_;
    std::vector<std::function<void(const Candle&)>> updateCallbacks_;
    
    // Load historical OHLCV from a real exchange API (Binance public klines).
    // Fail-closed: returns an empty vector on any fetch/parse failure.
    void loadHistoricalData() {
        historicalData_ = fetchKlines(config_.symbol, config_.interval, 500);
        for (const auto& candle : historicalData_) {
            for (const auto& callback : updateCallbacks_) {
                callback(candle);
            }
        }
    }

    static size_t curlWriteCb(char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* buf = static_cast<std::string*>(userdata);
        buf->append(ptr, size * nmemb);
        return size * nmemb;
    }

    static std::string binanceSymbol(const std::string& sym) {
        std::string out;
        for (char c : sym) if (c != '/' && c != '-' && c != '_') out += static_cast<char>(toupper(c));
        return out;
    }

    static std::string binanceInterval(const std::string& iv) {
        if (iv == "1m" || iv == "5m" || iv == "15m" || iv == "30m" ||
            iv == "1h" || iv == "4h" || iv == "1D" || iv == "1W" || iv == "1M") return iv;
        if (iv == "1") return "1m";
        if (iv == "60") return "1h";
        if (iv == "240") return "4h";
        if (iv == "D" || iv == "1d") return "1D";
        return "1h";
    }

    // Real Binance klines fetch via libcurl. Returns empty on any error.
    std::vector<Candle> fetchKlines(const Symbol& symbol, const Interval& interval, int limit) {
        std::vector<Candle> candles;
        std::string url = "https://api.binance.com/api/v3/klines?symbol=" + binanceSymbol(symbol) +
                          "&interval=" + binanceInterval(interval) +
                          "&limit=" + std::to_string(limit > 1000 ? 1000 : limit);
        CURL* curl = curl_easy_init();
        if (!curl) return candles;
        std::string resp;
        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, curlWriteCb);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &resp);
        curl_easy_setopt(curl, CURLOPT_TIMEOUT, 10L);
        curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 5L);
        CURLcode rc = curl_easy_perform(curl);
        long http = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http);
        curl_easy_cleanup(curl);
        if (rc != CURLE_OK || http != 200) return candles;
        try {
            json j = json::parse(resp);
            if (!j.is_array()) return candles;
            for (const auto& k : j) {
                if (!k.is_array() || k.size() < 6) continue;
                Candle c;
                c.time = k[0].get<uint64_t>() / 1000;
                c.open = std::stod(k[1].get<std::string>());
                c.high = std::stod(k[2].get<std::string>());
                c.low = std::stod(k[3].get<std::string>());
                c.close = std::stod(k[4].get<std::string>());
                c.volume = std::stod(k[5].get<std::string>());
                candles.push_back(c);
            }
        } catch (...) {
            return candles;
        }
        return candles;
    }

    // CSPRNG-based hex ID (replaces rand()-based generation).
    std::string generateId() {
        const char* hex = "0123456789abcdef";
        std::random_device rd;
        std::string id;
        for (int i = 0; i < 16; i++) {
            id += hex[rd() % 16];
        }
        return id;
    }
    
    static std::map<std::string, TradingViewChart*> widgets_;
    friend class ChartManager;
};

// =============================================================================
// TECHNICAL INDICATORS CALCULATOR
// =============================================================================

class TechnicalIndicators {
public:
    // Calculate SMA
    static std::vector<double> calculateSMA(const std::vector<Candle>& candles, int period) {
        std::vector<double> sma;
        
        if (candles.size() < period) return sma;
        
        for (size_t i = period - 1; i < candles.size(); i++) {
            double sum = 0;
            for (size_t j = i - period + 1; j <= i; j++) {
                sum += candles[j].close;
            }
            sma.push_back(sum / period);
        }
        
        return sma;
    }
    
    // Calculate EMA
    static std::vector<double> calculateEMA(const std::vector<Candle>& candles, int period) {
        std::vector<double> ema;
        
        if (candles.size() < period) return ema;
        
        double multiplier = 2.0 / (period + 1);
        
        // First EMA is SMA
        double sum = 0;
        for (int i = 0; i < period; i++) {
            sum += candles[i].close;
        }
        ema.push_back(sum / period);
        
        // Calculate rest
        for (size_t i = period; i < candles.size(); i++) {
            double prevEma = ema.back();
            double newEma = (candles[i].close - prevEma) * multiplier + prevEma;
            ema.push_back(newEma);
        }
        
        return ema;
    }
    
    // Calculate RSI
    static std::vector<double> calculateRSI(const std::vector<Candle>& candles, int period) {
        std::vector<double> rsi;
        
        if (candles.size() < period + 1) return rsi;
        
        // Calculate price changes
        std::vector<double> changes;
        for (size_t i = 1; i < candles.size(); i++) {
            changes.push_back(candles[i].close - candles[i-1].close);
        }
        
        // Calculate gains and losses
        double avgGain = 0, avgLoss = 0;
        
        // First average
        for (int i = 0; i < period; i++) {
            if (changes[i] > 0) avgGain += changes[i];
            else avgLoss -= changes[i];
        }
        avgGain /= period;
        avgLoss /= period;
        
        // First RSI
        if (avgLoss == 0) {
            rsi.push_back(100);
        } else {
            double rs = avgGain / avgLoss;
            rsi.push_back(100 - (100 / (1 + rs)));
        }
        
        // Calculate subsequent RSI using smoothed averages
        for (size_t i = period; i < changes.size(); i++) {
            double gain = changes[i] > 0 ? changes[i] : 0;
            double loss = changes[i] < 0 ? -changes[i] : 0;
            
            avgGain = (avgGain * (period - 1) + gain) / period;
            avgLoss = (avgLoss * (period - 1) + loss) / period;
            
            if (avgLoss == 0) {
                rsi.push_back(100);
            } else {
                double rs = avgGain / avgLoss;
                rsi.push_back(100 - (100 / (1 + rs)));
            }
        }
        
        return rsi;
    }
    
    // Calculate MACD
    static std::map<std::string, std::vector<double>> calculateMACD(
        const std::vector<Candle>& candles,
        int fastPeriod = 12,
        int slowPeriod = 26,
        int signalPeriod = 9
    ) {
        std::map<std::string, std::vector<double>> result;
        
        // Calculate fast and slow EMA
        std::vector<double> fastEMA = calculateEMA(candles, fastPeriod);
        std::vector<double> slowEMA = calculateEMA(candles, slowPeriod);
        
        // Align arrays
        size_t startIdx = slowPeriod - fastPeriod;
        
        // Calculate MACD line
        std::vector<double> macdLine;
        for (size_t i = startIdx; i < fastEMA.size(); i++) {
            macdLine.push_back(fastEMA[i] - slowEMA[i - startIdx]);
        }
        
        // Calculate signal line
        std::vector<double> signalLine = calculateEMAFromValues(macdLine, signalPeriod);
        
        // Calculate histogram
        std::vector<double> histogram;
        size_t signalStart = signalLine.size() - macdLine.size();
        for (size_t i = 0; i < macdLine.size(); i++) {
            if (i >= signalStart) {
                histogram.push_back(macdLine[i] - signalLine[i - signalStart]);
            } else {
                histogram.push_back(0);
            }
        }
        
        result["macd"] = macdLine;
        result["signal"] = signalLine;
        result["histogram"] = histogram;
        
        return result;
    }
    
    // Calculate Bollinger Bands
    static std::map<std::string, std::vector<double>> calculateBollingerBands(
        const std::vector<Candle>& candles,
        int period = 20,
        double stdDevMultiplier = 2.0
    ) {
        std::map<std::string, std::vector<double>> result;
        
        std::vector<double> sma = calculateSMA(candles, period);
        
        // Calculate standard deviation
        std::vector<double> upper, lower;
        
        for (size_t i = period - 1; i < candles.size(); i++) {
            double sum = 0;
            for (size_t j = i - period + 1; j <= i; j++) {
                double diff = candles[j].close - sma[i - period + 1];
                sum += diff * diff;
            }
            double stdDev = std::sqrt(sum / period);
            
            double middle = sma[i - period + 1];
            upper.push_back(middle + (stdDevMultiplier * stdDev));
            lower.push_back(middle - (stdDevMultiplier * stdDev));
        }
        
        result["middle"] = sma;
        result["upper"] = upper;
        result["lower"] = lower;
        
        return result;
    }
    
    // Calculate VWAP
    static std::vector<double> calculateVWAP(const std::vector<Candle>& candles) {
        std::vector<double> vwap;
        
        double cumulativeTPV = 0;
        double cumulativeVolume = 0;
        
        for (const auto& candle : candles) {
            double typicalPrice = (candle.high + candle.low + candle.close) / 3;
            cumulativeTPV += typicalPrice * candle.volume;
            cumulativeVolume += candle.volume;
            
            if (cumulativeVolume > 0) {
                vwap.push_back(cumulativeTPV / cumulativeVolume);
            }
        }
        
        return vwap;
    }
    
    // Calculate Volume Profile
    static std::map<double, double> calculateVolumeProfile(
        const std::vector<Candle>& candles,
        int bins = 20
    ) {
        std::map<double, double> profile;
        
        // Find price range
        double minPrice = candles[0].low;
        double maxPrice = candles[0].high;
        
        for (const auto& candle : candles) {
            if (candle.low < minPrice) minPrice = candle.low;
            if (candle.high > maxPrice) maxPrice = candle.high;
        }
        
        double binSize = (maxPrice - minPrice) / bins;
        
        // Initialize bins
        for (int i = 0; i < bins; i++) {
            profile[minPrice + (i * binSize) + (binSize / 2)] = 0;
        }
        
        // Fill profile
        for (const auto& candle : candles) {
            int startBin = (int)((candle.low - minPrice) / binSize);
            int endBin = (int)((candle.high - minPrice) / binSize);
            
            for (int i = startBin; i <= endBin && i < bins; i++) {
                double priceLevel = minPrice + (i * binSize) + (binSize / 2);
                profile[priceLevel] += candle.volume;
            }
        }
        
        return profile;
    }
    
private:
    static std::vector<double> calculateEMAFromValues(const std::vector<double>& values, int period) {
        std::vector<double> ema;
        
        if (values.size() < period) return ema;
        
        double multiplier = 2.0 / (period + 1);
        
        // First EMA
        double sum = 0;
        for (int i = 0; i < period; i++) {
            sum += values[i];
        }
        ema.push_back(sum / period);
        
        // Calculate rest
        for (size_t i = period; i < values.size(); i++) {
            double prevEma = ema.back();
            double newEma = (values[i] - prevEma) * multiplier + prevEma;
            ema.push_back(newEma);
        }
        
        return ema;
    }
};

// =============================================================================
// CHART DATA PROVIDER
// =============================================================================

class ChartDataProvider {
public:
    ChartDataProvider() {
        curl_ = curl_easy_init();
    }
    
    ~ChartDataProvider() {
        if (curl_) curl_easy_cleanup(curl_);
    }
    
    // Fetch historical candles from a real exchange API (Binance klines).
    // Fail-closed: returns an empty vector on any HTTP/parse failure.
    std::vector<Candle> fetchCandles(
        const Symbol& symbol,
        const Interval& interval,
        int limit = 500
    ) {
        return fetchBinanceKlines(symbol, interval, limit);
    }
    
    // Fetch real-time candle
    std::optional<Candle> fetchLatestCandle(const Symbol& symbol) {
        auto candles = fetchCandles(symbol, "1m", 1);
        if (!candles.empty()) {
            return candles.back();
        }
        return std::nullopt;
    }
    
    // Subscribe to real-time updates
    void subscribe(const Symbol& symbol, std::function<void(const Candle&)> callback) {
        subscriptions_[symbol].push_back(callback);
        
        // Start background feed if not already running
        if (!isRunning_) {
            startFeed();
        }
    }
    
    // Unsubscribe
    void unsubscribe(const Symbol& symbol) {
        subscriptions_.erase(symbol);
    }
    
private:
    CURL* curl_;
    bool isRunning_ = false;
    std::map<Symbol, std::vector<std::function<void(const Candle&)>>> subscriptions_;
    std::thread feedThread_;
    
    void startFeed() {
        isRunning_ = true;
        
        feedThread_ = std::thread([this]() {
            while (isRunning_) {
                // Update all subscribed symbols
                for (const auto& [symbol, callbacks] : subscriptions_) {
                    auto candle = fetchLatestCandle(symbol);
                    if (candle.has_value()) {
                        for (const auto& callback : callbacks) {
                            callback(candle.value());
                        }
                    }
                }
                
                std::this_thread::sleep_for(std::chrono::seconds(1));
            }
        });
    }
    
std::vector<Candle> fetchBinanceKlines(
        const Symbol& symbol,
        const Interval& interval,
        int count
    ) {
        std::vector<Candle> candles;
        if (!curl_) return candles;
        std::string sym;
        for (char c : symbol) if (c != '/' && c != '-' && c != '_') sym += static_cast<char>(toupper(c));
        std::string iv = binanceInterval(interval);
        int limit = count > 1000 ? 1000 : count;
        std::string url = "https://api.binance.com/api/v3/klines?symbol=" + sym +
                          "&interval=" + iv + "&limit=" + std::to_string(limit);
        std::string resp;
        curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, &ChartDataProvider::curlWriteCb);
        curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &resp);
        curl_easy_setopt(curl_, CURLOPT_TIMEOUT, 10L);
        curl_easy_setopt(curl_, CURLOPT_CONNECTTIMEOUT, 5L);
        CURLcode rc = curl_easy_perform(curl_);
        long http = 0;
        curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &http);
        if (rc != CURLE_OK || http != 200) return candles;
        try {
            json j = json::parse(resp);
            if (!j.is_array()) return candles;
            for (const auto& k : j) {
                if (!k.is_array() || k.size() < 6) continue;
                Candle c;
                c.time = k[0].get<uint64_t>() / 1000;
                c.open = std::stod(k[1].get<std::string>());
                c.high = std::stod(k[2].get<std::string>());
                c.low = std::stod(k[3].get<std::string>());
                c.close = std::stod(k[4].get<std::string>());
                c.volume = std::stod(k[5].get<std::string>());
                candles.push_back(c);
            }
        } catch (...) {
            return candles;
        }
        return candles;
    }

    static size_t curlWriteCb(char* ptr, size_t size, size_t nmemb, void* userdata) {
        auto* buf = static_cast<std::string*>(userdata);
        buf->append(ptr, size * nmemb);
        return size * nmemb;
    }

    static std::string binanceInterval(const std::string& iv) {
        if (iv == "1m" || iv == "5m" || iv == "15m" || iv == "30m" ||
            iv == "1h" || iv == "4h" || iv == "1D" || iv == "1W" || iv == "1M") return iv;
        if (iv == "1") return "1m";
        if (iv == "60") return "1h";
        if (iv == "240") return "4h";
        if (iv == "D" || iv == "1d") return "1D";
        return "1h";
    }
    
    uint64_t getIntervalSeconds(const Interval& interval) {
        if (interval == "1m") return 60;
        if (interval == "5m") return 300;
        if (interval == "15m") return 900;
        if (interval == "1h") return 3600;
        if (interval == "4h") return 14400;
        if (interval == "1D") return 86400;
        if (interval == "1W") return 604800;
        if (interval == "1M") return 2592000;
        return 60;
    }
};

// =============================================================================
// CHART MANAGER
// =============================================================================

class ChartManager {
public:
    ChartManager() : dataProvider_() {}
    
    // Create a new chart
    TradingViewChart* createChart(const ChartConfig& config) {
        auto chart = new TradingViewChart(config);
        chart->initialize();
        return chart;
    }
    
    // Get chart
    TradingViewChart* getChart(const std::string& containerId) {
        auto it = TradingViewChart::widgets_.find(containerId);
        if (it != TradingViewChart::widgets_.end()) {
            return it->second;
        }
        return nullptr;
    }
    
    // Destroy chart
    void destroyChart(const std::string& containerId) {
        auto it = TradingViewChart::widgets_.find(containerId);
        if (it != TradingViewChart::widgets_.end()) {
            delete it->second;
            TradingViewChart::widgets_.erase(it);
        }
    }
    
    // Get data provider
    ChartDataProvider& getDataProvider() { return dataProvider_; }
    
    // Calculate indicators for chart
    std::map<std::string, std::vector<double>> calculateIndicators(
        const std::vector<Candle>& candles,
        const std::vector<IndicatorConfig>& indicators
    ) {
        std::map<std::string, std::vector<double>> results;
        
        for (const auto& indicator : indicators) {
            if (!indicator.enabled) continue;
            
            switch (indicator.type) {
                case IndicatorType::SMA: {
                    int period = (int)indicator.params.at("period");
                    results[indicator.name] = TechnicalIndicators::calculateSMA(candles, period);
                    break;
                }
                case IndicatorType::EMA: {
                    int period = (int)indicator.params.at("period");
                    results[indicator.name] = TechnicalIndicators::calculateEMA(candles, period);
                    break;
                }
                case IndicatorType::RSI: {
                    int period = (int)indicator.params.at("period");
                    results[indicator.name] = TechnicalIndicators::calculateRSI(candles, period);
                    break;
                }
                case IndicatorType::MACD: {
                    auto macd = TechnicalIndicators::calculateMACD(candles);
                    results[indicator.name + "_macd"] = macd["macd"];
                    results[indicator.name + "_signal"] = macd["signal"];
                    results[indicator.name + "_histogram"] = macd["histogram"];
                    break;
                }
                case IndicatorType::BOLLINGER: {
                    int period = (int)indicator.params.at("period");
                    double stdDev = indicator.params.count("stdDev") ? indicator.params.at("stdDev") : 2.0;
                    auto bb = TechnicalIndicators::calculateBollingerBands(candles, period, stdDev);
                    results[indicator.name + "_upper"] = bb["upper"];
                    results[indicator.name + "_middle"] = bb["middle"];
                    results[indicator.name + "_lower"] = bb["lower"];
                    break;
                }
                case IndicatorType::VWAP: {
                    results[indicator.name] = TechnicalIndicators::calculateVWAP(candles);
                    break;
                }
                default:
                    break;
            }
        }
        
        return results;
    }
    
private:
    ChartDataProvider dataProvider_;
};

std::map<std::string, TradingViewChart*> TradingViewChart::widgets_;

} // namespace tiger

#endif // TIGERWALLET_TRADINGVIEW_HPP
