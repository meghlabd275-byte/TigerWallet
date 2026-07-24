/**
 * TigerWallet System Monitoring Service
 * High-Security Rust Implementation
 * 
 * Features:
 * - System health monitoring
 * - Alert management
 * - Metrics collection
 * - Performance tracking
 */

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Clone)]
pub struct HealthCheck {
    pub service: String,
    pub status: String,
    pub latency_ms: u64,
    pub last_check: u64,
    pub message: String,
}

#[derive(Debug, Clone)]
pub struct Metric {
    pub name: String,
    pub value: f64,
    pub timestamp: u64,
    pub labels: HashMap<String, String>,
}

#[derive(Debug, Clone)]
pub struct Alert {
    pub id: String,
    pub severity: String,
    pub service: String,
    pub message: String,
    pub triggered_at: u64,
    pub acknowledged: bool,
}

pub struct MonitoringService {
    health_checks: RwLock<HashMap<String, HealthCheck>>,
    metrics: RwLock<Vec<Metric>>,
    alerts: RwLock<Vec<Alert>>,
}

impl MonitoringService {
    pub fn new() -> Self {
        Self {
            health_checks: RwLock::new(HashMap::new()),
            metrics: RwLock::new(Vec::new()),
            alerts: RwLock::new(Vec::new()),
        }
    }

    pub fn check_health(&self, service: &str, status: &str, latency_ms: u64) {
        let check = HealthCheck {
            service: service.to_string(),
            status: status.to_string(),
            latency_ms,
            last_check: current_time(),
            message: "OK".to_string(),
        };
        
        self.health_checks.write().unwrap()
            .insert(service.to_string(), check);
    }

    pub fn record_metric(&self, name: &str, value: f64, labels: HashMap<String, String>) {
        let metric = Metric {
            name: name.to_string(),
            value,
            timestamp: current_time(),
            labels,
        };
        
        self.metrics.write().unwrap().push(metric);
    }

    pub fn create_alert(&self, severity: &str, service: &str, message: &str) -> String {
        let id = format!("alert_{}", current_time());
        
        let alert = Alert {
            id: id.clone(),
            severity: severity.to_string(),
            service: service.to_string(),
            message: message.to_string(),
            triggered_at: current_time(),
            acknowledged: false,
        };
        
        self.alerts.write().unwrap().push(alert);
        id
    }

    pub fn get_health(&self) -> Vec<HealthCheck> {
        let checks = self.health_checks.read().unwrap();
        checks.values().cloned().collect()
    }

    pub fn get_metrics(&self, name: &str) -> Vec<Metric> {
        let metrics = self.metrics.read().unwrap();
        metrics.iter()
            .filter(|m| m.name == name)
            .cloned()
            .collect()
    }

    pub fn get_alerts(&self) -> Vec<Alert> {
        let alerts = self.alerts.read().unwrap();
        alerts.clone()
    }

    pub fn acknowledge_alert(&self, id: &str) -> bool {
        let mut alerts = self.alerts.write().unwrap();
        if let Some(alert) = alerts.iter_mut().find(|a| a.id == id) {
            alert.acknowledged = true;
            return true;
        }
        false
    }
}

fn current_time() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

fn main() {
    println!("TigerWallet System Monitoring Service");
    println!("======================================");
    
    let monitor = Arc::new(MonitoringService::new());
    
    // Demo: Record some health checks
    monitor.check_health("api_gateway", "healthy", 15);
    monitor.check_health("blockchain", "healthy", 45);
    monitor.check_health("database", "healthy", 8);
    
    // Demo: Record metrics
    let mut labels = HashMap::new();
    labels.insert("chain".to_string(), "ethereum".to_string());
    monitor.record_metric("tx_count", 1250.0, labels);
    
    monitor.record_metric("cpu_usage", 45.5, HashMap::new());
    monitor.record_metric("memory_usage", 68.2, HashMap::new());
    
    // Demo: Create alert
    let alert_id = monitor.create_alert("warning", "database", "High latency detected");
    println!("Alert created: {}", alert_id);
    
    // Show health
    let health = monitor.get_health();
    println!("\nHealth Status:");
    for h in &health {
        println!("  {}: {} ({}ms)", h.service, h.status, h.latency_ms);
    }
    
    // Show alerts
    let alerts = monitor.get_alerts();
    println!("\nActive Alerts: {}", alerts.len());
    
    println!("\nService running on :8091");
}
