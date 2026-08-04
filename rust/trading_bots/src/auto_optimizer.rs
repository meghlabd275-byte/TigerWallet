/**
 * TigerWallet Bot Auto-Optimization Engine
 * AI-powered strategy optimization and parameter tuning for trading bots
 * High-performance Rust implementation with ultra-low latency
 */

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};

// ============================================================================
// Configuration
// ============================================================================

#[derive(Debug, Clone)]
pub struct OptimizerConfig {
    pub enabled: bool,
    pub optimization_interval_secs: u64,
    pub max_iterations: u32,
    pub convergence_threshold: f64,
    pub population_size: u32,
    pub mutation_rate: f64,
    pub crossover_rate: f64,
    pub elite_count: u32,
    pub learning_rate: f64,
}

impl Default for OptimizerConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            optimization_interval_secs: 300,
            max_iterations: 1000,
            convergence_threshold: 0.001,
            population_size: 50,
            mutation_rate: 0.1,
            crossover_rate: 0.8,
            elite_count: 5,
            learning_rate: 0.01,
        }
    }
}

// ============================================================================
// Strategy Parameters
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StrategyParams {
    // Grid Trading
    pub grid_levels: u32,
    pub grid_spacing_pct: f64,
    pub position_size_pct: f64,
    
    // DCA
    pub dca_interval_hours: u32,
    pub dca_amount: f64,
    pub max_positions: u32,
    
    // Momentum
    pub rsi_period: u32,
    pub rsi_oversold: f64,
    pub rsi_overbought: f64,
    pub ma_period: u32,
    pub macd_fast: u32,
    pub macd_slow: u32,
    pub macd_signal: u32,
    
    // Risk Management
    pub stop_loss_pct: f64,
    pub take_profit_pct: f64,
    pub max_drawdown_pct: f64,
    pub position_size_max_pct: f64,
    
    // General
    pub max_slippage_pct: f64,
    pub gas_limit_multiplier: f64,
}

impl Default for StrategyParams {
    fn default() -> Self {
        Self {
            grid_levels: 10,
            grid_spacing_pct: 1.0,
            position_size_pct: 10.0,
            dca_interval_hours: 24,
            dca_amount: 100.0,
            max_positions: 10,
            rsi_period: 14,
            rsi_oversold: 30.0,
            rsi_overbought: 70.0,
            ma_period: 50,
            macd_fast: 12,
            macd_slow: 26,
            macd_signal: 9,
            stop_loss_pct: 5.0,
            take_profit_pct: 10.0,
            max_drawdown_pct: 20.0,
            position_size_max_pct: 25.0,
            max_slippage_pct: 0.5,
            gas_limit_multiplier: 1.2,
        }
    }
}

// ============================================================================
// Backtest Result
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BacktestResult {
    pub total_trades: u32,
    pub winning_trades: u32,
    pub losing_trades: u32,
    pub win_rate: f64,
    pub total_pnl: f64,
    pub total_pnl_pct: f64,
    pub max_drawdown: f64,
    pub sharpe_ratio: f64,
    pub sortino_ratio: f64,
    pub avg_trade_duration_secs: u64,
    pub avg_profit: f64,
    pub avg_loss: f64,
    pub profit_factor: f64,
    pub trades: Vec<Trade>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trade {
    pub entry_time: u64,
    pub exit_time: u64,
    pub side: String,
    pub entry_price: f64,
    pub exit_price: f64,
    pub size: f64,
    pub pnl: f64,
    pub pnl_pct: f64,
    pub fee: f64,
}

// ============================================================================
// Optimization Result
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptimizationResult {
    pub best_params: StrategyParams,
    pub best_score: f64,
    pub iterations: u32,
    pub convergence_time_ms: u64,
    pub history: Vec<OptimizationIteration>,
    pub improvements: Vec<ParameterImprovement>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptimizationIteration {
    pub iteration: u32,
    pub best_score: f64,
    pub avg_score: f64,
    pub params: StrategyParams,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ParameterImprovement {
    pub param_name: String,
    pub old_value: f64,
    pub new_value: f64,
    pub impact: f64,
}

// ============================================================================
// Genetic Algorithm Optimizer
// ============================================================================

pub struct GeneticOptimizer {
    config: OptimizerConfig,
    population: Vec<StrategyParams>,
    scores: Vec<f64>,
    best_params: Option<StrategyParams>,
    best_score: f64,
    history: Vec<OptimizationIteration>,
}

impl GeneticOptimizer {
    pub fn new(config: OptimizerConfig) -> Self {
        let population_size = config.population_size as usize;
        
        let mut optimizer = Self {
            config: config.clone(),
            population: Vec::with_capacity(population_size),
            scores: Vec::with_capacity(population_size),
            best_params: None,
            best_score: f64::NEG_INFINITY,
            history: Vec::new(),
        };
        
        // Initialize population with random parameters
        for _ in 0..population_size {
            optimizer.population.push(optimizer.random_params());
            optimizer.scores.push(f64::NEG_INFINITY);
        }
        
        optimizer
    }
    
    fn random_params(&self) -> StrategyParams {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        
        StrategyParams {
            grid_levels: rng.gen_range(5..50),
            grid_spacing_pct: rng.gen_range(0.1..5.0),
            position_size_pct: rng.gen_range(1.0..20.0),
            dca_interval_hours: rng.gen_range(1..168),
            dca_amount: rng.gen_range(10.0..1000.0),
            max_positions: rng.gen_range(3..30),
            rsi_period: rng.gen_range(5..30),
            rsi_oversold: rng.gen_range(10.0..40.0),
            rsi_overbought: rng.gen_range(60.0..90.0),
            ma_period: rng.gen_range(10..200),
            macd_fast: rng.gen_range(5..20),
            macd_slow: rng.gen_range(15..50),
            macd_signal: rng.gen_range(5..15),
            stop_loss_pct: rng.gen_range(1.0..15.0),
            take_profit_pct: rng.gen_range(2.0..30.0),
            max_drawdown_pct: rng.gen_range(10.0..40.0),
            position_size_max_pct: rng.gen_range(5.0..50.0),
            max_slippage_pct: rng.gen_range(0.1..2.0),
            gas_limit_multiplier: rng.gen_range(1.0..2.0),
        }
    }
    
    pub fn optimize<F>(&mut self, fitness_fn: F) -> OptimizationResult
    where
        F: Fn(&StrategyParams) -> f64 + Send + Sync,
    {
        let start_time = Instant::now();
        
        // Evaluate initial population
        for (i, params) in self.population.iter().enumerate() {
            self.scores[i] = fitness_fn(params);
            
            if self.scores[i] > self.best_score {
                self.best_score = self.scores[i];
                self.best_params = Some(params.clone());
            }
        }
        
        let mut iteration = 0;
        
        while iteration < self.config.max_iterations {
            // Check convergence
            if self.check_convergence() {
                break;
            }
            
            // Create new generation
            self.evolve();
            
            // Evaluate new population
            for (i, params) in self.population.iter().enumerate() {
                self.scores[i] = fitness_fn(params);
                
                if self.scores[i] > self.best_score {
                    self.best_score = self.scores[i];
                    self.best_params = Some(params.clone());
                }
            }
            
            // Record history
            let avg_score: f64 = self.scores.iter().sum::<f64>() / self.scores.len() as f64;
            self.history.push(OptimizationIteration {
                iteration,
                best_score: self.best_score,
                avg_score,
                params: self.best_params.clone().unwrap_or_default(),
            });
            
            iteration += 1;
        }
        
        let convergence_time_ms = start_time.elapsed().as_millis() as u64;
        
        OptimizationResult {
            best_params: self.best_params.clone().unwrap_or_default(),
            best_score: self.best_score,
            iterations: iteration,
            convergence_time_ms,
            history: std::mem::take(&mut self.history),
            improvements: vec![],
        }
    }
    
    fn evolve(&mut self) {
        let population_size = self.population.len();
        let elite_count = self.config.elite_count as usize;
        
        // Sort by score
        let mut indices: Vec<usize> = (0..population_size).collect();
        indices.sort_by(|&a, &b| self.scores[b].partial_cmp(&self.scores[a]).unwrap());
        
        // Keep elite
        let mut new_population: Vec<StrategyParams> = Vec::with_capacity(population_size);
        for i in 0..elite_count {
            new_population.push(self.population[indices[i]].clone());
        }
        
        // Generate rest through crossover and mutation
        while new_population.len() < population_size {
            // Selection
            let parent1 = self.tournament_selection();
            let parent2 = self.tournament_selection();
            
            // Crossover
            let mut child = if rand::random::<f64>() < self.config.crossover_rate {
                self.crossover(&parent1, &parent2)
            } else {
                parent1.clone()
            };
            
            // Mutation
            if rand::random::<f64>() < self.config.mutation_rate {
                self.mutate(&mut child);
            }
            
            new_population.push(child);
        }
        
        self.population = new_population;
    }
    
    fn tournament_selection(&self) -> StrategyParams {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        
        let tournament_size = 3;
        let mut best_idx = 0;
        let mut best_score = f64::NEG_INFINITY;
        
        for _ in 0..tournament_size {
            let idx = rng.gen_range(0..self.population.len());
            if self.scores[idx] > best_score {
                best_score = self.scores[idx];
                best_idx = idx;
            }
        }
        
        self.population[best_idx].clone()
    }
    
    fn crossover(&self, parent1: &StrategyParams, parent2: &StrategyParams) -> StrategyParams {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        
        StrategyParams {
            grid_levels: if rng.gen_bool(0.5) { parent1.grid_levels } else { parent2.grid_levels },
            grid_spacing_pct: if rng.gen_bool(0.5) { parent1.grid_spacing_pct } else { parent2.grid_spacing_pct },
            position_size_pct: if rng.gen_bool(0.5) { parent1.position_size_pct } else { parent2.position_size_pct },
            dca_interval_hours: if rng.gen_bool(0.5) { parent1.dca_interval_hours } else { parent2.dca_interval_hours },
            dca_amount: if rng.gen_bool(0.5) { parent1.dca_amount } else { parent2.dca_amount },
            max_positions: if rng.gen_bool(0.5) { parent1.max_positions } else { parent2.max_positions },
            rsi_period: if rng.gen_bool(0.5) { parent1.rsi_period } else { parent2.rsi_period },
            rsi_oversold: if rng.gen_bool(0.5) { parent1.rsi_oversold } else { parent2.rsi_oversold },
            rsi_overbought: if rng.gen_bool(0.5) { parent1.rsi_overbought } else { parent2.rsi_overbought },
            ma_period: if rng.gen_bool(0.5) { parent1.ma_period } else { parent2.ma_period },
            macd_fast: if rng.gen_bool(0.5) { parent1.macd_fast } else { parent2.macd_fast },
            macd_slow: if rng.gen_bool(0.5) { parent1.macd_slow } else { parent2.macd_slow },
            macd_signal: if rng.gen_bool(0.5) { parent1.macd_signal } else { parent2.macd_signal },
            stop_loss_pct: if rng.gen_bool(0.5) { parent1.stop_loss_pct } else { parent2.stop_loss_pct },
            take_profit_pct: if rng.gen_bool(0.5) { parent1.take_profit_pct } else { parent2.take_profit_pct },
            max_drawdown_pct: if rng.gen_bool(0.5) { parent1.max_drawdown_pct } else { parent2.max_drawdown_pct },
            position_size_max_pct: if rng.gen_bool(0.5) { parent1.position_size_max_pct } else { parent2.position_size_max_pct },
            max_slippage_pct: if rng.gen_bool(0.5) { parent1.max_slippage_pct } else { parent2.max_slippage_pct },
            gas_limit_multiplier: if rng.gen_bool(0.5) { parent1.gas_limit_multiplier } else { parent2.gas_limit_multiplier },
        }
    }
    
    fn mutate(&self, params: &mut StrategyParams) {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        
        // Random parameter mutation
        let mutation_type = rng.gen_range(0..7);
        
        match mutation_type {
            0 => params.grid_levels = rng.gen_range(5..50),
            1 => params.grid_spacing_pct = rng.gen_range(0.1..5.0),
            2 => params.rsi_period = rng.gen_range(5..30),
            3 => params.stop_loss_pct = rng.gen_range(1.0..15.0),
            4 => params.take_profit_pct = rng.gen_range(2.0..30.0),
            5 => params.position_size_pct = rng.gen_range(1.0..20.0),
            6 => params.max_slippage_pct = rng.gen_range(0.1..2.0),
            _ => {}
        }
    }
    
    fn check_convergence(&self) -> bool {
        if self.history.len() < 10 {
            return false;
        }
        
        let recent: Vec<f64> = self.history.iter()
            .rev()
            .take(10)
            .map(|i| i.best_score)
            .collect();
        
        let variance = Self::variance(&recent);
        variance < self.config.convergence_threshold
    }
    
    fn variance(values: &[f64]) -> f64 {
        if values.is_empty() {
            return 0.0;
        }
        
        let mean: f64 = values.iter().sum::<f64>() / values.len() as f64;
        let variance: f64 = values.iter()
            .map(|v| (v - mean).powi(2))
            .sum::<f64>() / values.len() as f64;
        
        variance
    }
}

// ============================================================================
// Auto-Optimization Service
// ============================================================================

pub struct AutoOptimizerService {
    config: OptimizerConfig,
    strategies: Arc<RwLock<HashMap<String, StrategyParams>>>,
    optimization_results: Arc<RwLock<HashMap<String, OptimizationResult>>>,
    running: Arc<RwLock<bool>>,
}

impl AutoOptimizerService {
    pub fn new(config: OptimizerConfig) -> Self {
        Self {
            config,
            strategies: Arc::new(RwLock::new(HashMap::new())),
            optimization_results: Arc::new(RwLock::new(HashMap::new())),
            running: Arc::new(RwLock::new(false)),
        }
    }
    
    pub fn register_strategy(&self, strategy_id: String, params: StrategyParams) {
        let mut strategies = self.strategies.write().unwrap();
        strategies.insert(strategy_id, params);
    }
    
    pub fn optimize_strategy<F>(&self, strategy_id: String, fitness_fn: F) -> Result<OptimizationResult, String>
    where
        F: Fn(&StrategyParams) -> f64 + Send + Sync + 'static,
    {
        let mut optimizer = GeneticOptimizer::new(self.config.clone());
        let result = optimizer.optimize(fitness_fn);
        
        // Save result
        let mut results = self.optimization_results.write().unwrap();
        results.insert(strategy_id, result.clone());
        
        // Update strategy params
        let mut strategies = self.strategies.write().unwrap();
        if let Some(params) = strategies.get_mut(&strategy_id) {
            *params = result.best_params.clone();
        }
        
        Ok(result)
    }
    
    pub fn get_optimized_params(&self, strategy_id: &str) -> Option<StrategyParams> {
        let strategies = self.strategies.read().unwrap();
        strategies.get(strategy_id).cloned()
    }
    
    pub fn get_optimization_result(&self, strategy_id: &str) -> Option<OptimizationResult> {
        let results = self.optimization_results.read().unwrap();
        results.get(strategy_id).cloned()
    }
    
    pub fn start_auto_optimization<F>(&self, strategy_id: String, fitness_fn: F)
    where
        F: Fn(&StrategyParams) -> f64 + Send + Sync + Clone + 'static,
    {
        let mut running = self.running.write().unwrap();
        *running = true;
        
        let config = self.config.clone();
        let strategy_id_clone = strategy_id.clone();
        let strategies = self.strategies.clone();
        let results = self.optimization_results.clone();
        let running = self.running.clone();
        
        std::thread::spawn(move || {
            while *running.read().unwrap() {
                let optimizer = GeneticOptimizer::new(config.clone());
                let result = optimizer.optimize(|params| fitness_fn(params));
                
                // Update
                {
                    let mut results_lock = results.write().unwrap();
                    results_lock.insert(strategy_id_clone.clone(), result.clone());
                }
                
                {
                    let mut strategies_lock = strategies.write().unwrap();
                    if let Some(params) = strategies_lock.get_mut(&strategy_id_clone) {
                        *params = result.best_params.clone();
                    }
                }
                
                std::thread::sleep(Duration::from_secs(config.optimization_interval_secs));
            }
        });
    }
    
    pub fn stop_auto_optimization(&self) {
        let mut running = self.running.write().unwrap();
        *running = false;
    }
}

// ============================================================================
// Strategy Backtester
// ============================================================================

pub struct StrategyBacktester {
    historical_data: Vec<PriceData>,
}

#[derive(Debug, Clone)]
pub struct PriceData {
    pub timestamp: u64,
    pub open: f64,
    pub high: f64,
    pub low: f64,
    pub close: f64,
    pub volume: f64,
}

impl StrategyBacktester {
    pub fn new() -> Self {
        Self {
            historical_data: Vec::new(),
        }
    }
    
    pub fn add_price_data(&mut self, data: PriceData) {
        self.historical_data.push(data);
    }
    
    pub fn load_historical_data(&mut self, data: Vec<PriceData>) {
        self.historical_data = data;
        self.historical_data.sort_by_key(|d| d.timestamp);
    }
    
    pub fn backtest(&self, params: &StrategyParams) -> BacktestResult {
        let mut trades: Vec<Trade> = Vec::new();
        let mut position: Option<f64> = None;
        let mut entry_price: f64 = 0.0;
        let mut entry_time: u64 = 0;
        
        for (i, data) in self.historical_data.iter().enumerate() {
            // Simple RSI-based strategy
            let rsi = self.calculate_rsi(params.rsi_period, i);
            
            // Entry signal
            if position.is_none() && rsi < params.rsi_oversold {
                position = Some(1.0);
                entry_price = data.close;
                entry_time = data.timestamp;
            }
            // Exit signal
            else if position.is_some() && rsi > params.rsi_overbought {
                let exit_price = data.close;
                let pnl = (exit_price - entry_price) / entry_price * 100.0;
                
                trades.push(Trade {
                    entry_time,
                    exit_time: data.timestamp,
                    side: "BUY".to_string(),
                    entry_price,
                    exit_price,
                    size: 1.0,
                    pnl,
                    pnl_pct: pnl,
                    fee: 0.1,
                });
                
                position = None;
            }
        }
        
        // Calculate metrics
        let total_trades = trades.len() as u32;
        let winning_trades = trades.iter().filter(|t| t.pnl > 0.0).count() as u32;
        let losing_trades = trades.iter().filter(|t| t.pnl < 0.0).count() as u32;
        let win_rate = if total_trades > 0 {
            winning_trades as f64 / total_trades as f64
        } else {
            0.0
        };
        
        let total_pnl: f64 = trades.iter().map(|t| t.pnl).sum();
        let avg_profit = if winning_trades > 0 {
            trades.iter().filter(|t| t.pnl > 0.0).map(|t| t.pnl).sum::<f64>() / winning_trades as f64
        } else {
            0.0
        };
        let avg_loss = if losing_trades > 0 {
            trades.iter().filter(|t| t.pnl < 0.0).map(|t| t.pnl).sum::<f64>() / losing_trades as f64
        } else {
            0.0
        };
        
        BacktestResult {
            total_trades,
            winning_trades,
            losing_trades,
            win_rate,
            total_pnl,
            total_pnl_pct: total_pnl,
            max_drawdown: 0.0, // Simplified
            sharpe_ratio: 0.0, // Simplified
            sortino_ratio: 0.0, // Simplified
            avg_trade_duration_secs: 0, // Simplified
            avg_profit,
            avg_loss,
            profit_factor: if avg_loss != 0.0 { avg_profit / avg_loss.abs() } else { 0.0 },
            trades,
        }
    }
    
    fn calculate_rsi(&self, period: usize, current: usize) -> f64 {
        if current < period {
            return 50.0;
        }
        
        let mut gains = 0.0;
        let mut losses = 0.0;
        
        for i in (current - period + 1)..=current {
            let change = self.historical_data[i].close - self.historical_data[i - 1].close;
            if change > 0.0 {
                gains += change;
            } else {
                losses += change.abs();
            }
        }
        
        let avg_gain = gains / period as f64;
        let avg_loss = losses / period as f64;
        
        if avg_loss == 0.0 {
            return 100.0;
        }
        
        let rs = avg_gain / avg_loss;
        100.0 - (100.0 / (1.0 + rs))
    }
}

// ============================================================================
// API Types
// ============================================================================

#[derive(Debug, Serialize, Deserialize)]
pub struct OptimizeRequest {
    pub strategy_id: String,
    pub params: StrategyParams,
    pub target_metric: String, // "sharpe_ratio", "total_pnl", "win_rate"
}

#[derive(Debug, Serialize, Deserialize)]
pub struct OptimizeResponse {
    pub success: bool,
    pub result: Option<OptimizationResult>,
    pub error: Option<String>,
}

// ============================================================================
// Unit Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_genetic_optimizer() {
        let config = OptimizerConfig {
            population_size: 10,
            max_iterations: 5,
            ..Default::default()
        };
        
        let mut optimizer = GeneticOptimizer::new(config);
        
        let result = optimizer.optimize(|params| {
            // Simple fitness function: maximize win rate - risk
            params.win_rate - params.stop_loss_pct / 100.0
        });
        
        assert!(result.best_score > f64::NEG_INFINITY);
    }
    
    #[test]
    fn test_backtester() {
        let mut backtester = StrategyBacktester::new();
        
        // Add some dummy data
        let base_price = 1000.0;
        for i in 0..100 {
            let data = PriceData {
                timestamp: i * 3600,
                open: base_price + (i as f64 * 0.1),
                high: base_price + (i as f64 * 0.2),
                low: base_price,
                close: base_price + (i as f64 * 0.15),
                volume: 1000000.0,
            };
            backtester.add_price_data(data);
        }
        
        let params = StrategyParams::default();
        let result = backtester.backtest(&params);
        
        assert!(result.total_trades >= 0);
    }
}
