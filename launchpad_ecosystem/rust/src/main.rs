/**
 * TigerWallet Launchpad Service
 * High-Speed Token Launch & IEO Platform
 * 
 * Features:
 * - Token Launches
 * - IEO (Initial Exchange Offering)
 * - IDO (Initial DEX Offering)
 * - Fair Launch
 * - Locked Liquidity
 * - Vesting Schedules
 * - Tier System
 * - Allocation Management
 */

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use uuid::Uuid;

// ============================================================================
// Types
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum LaunchStatus {
    Pending,
    Upcoming,
    Active,
    Ended,
    Completed,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum LaunchType {
    IEO,      // Initial Exchange Offering
    IDO,      // Initial DEX Offering
    FairLaunch,
    SeedSale,
    PrivateSale,
    PublicSale,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum SalePhase {
    Whitelist,
    Tier1,
    Tier2,
    Tier3,
    Public,
}

// ============================================================================
// Core Structures
// ============================================================================

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Token {
    pub token_id: String,
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub total_supply: String,
    pub contract_address: String,
    pub chain_id: u64,
    pub logo_url: String,
    pub description: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LaunchProject {
    pub project_id: String,
    pub name: String,
    pub symbol: String,
    pub description: String,
    pub logo_url: String,
    pub banner_url: String,
    pub website: String,
    pub whitepaper: String,
    pub social_links: HashMap<String, String>,
    
    // Token info
    pub token: Token,
    pub sale_token_supply: String,
    pub token_price: String,
    pub min_allocation: String,
    pub max_allocation: String,
    
    // Launch config
    pub launch_type: LaunchType,
    pub status: LaunchStatus,
    pub start_time: DateTime<Utc>,
    pub end_time: DateTime<Utc>,
    
    // Funding
    pub hard_cap: String,
    pub soft_cap: String,
    pub raised_amount: String,
    pub total_contributors: u64,
    
    // Liquidity
    pub liquidity_percentage: f64,
    pub liquidity_lock_days: u32,
    pub listing_price: String,
    
    // Vesting
    pub team_vesting_percentage: f64,
    pub team_vesting_months: u32,
    pub advisor_vesting_percentage: f64,
    pub advisor_vesting_months: u32,
    
    // Status
    pub is_verified: bool,
    pub kyc_status: String,
    pub audit_status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserTier {
    pub user_id: String,
    pub tier: u8,  // 1-5
    pub allocation: String,
    pub used_allocation: String,
    pub remaining_allocation: String,
    pub is_whitelisted: bool,
    pub whitelist_phase: Option<SalePhase>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Contribution {
    pub contribution_id: String,
    pub project_id: String,
    pub user_id: String,
    pub amount: String,
    pub token_amount: String,
    pub tx_hash: String,
    pub status: String,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VestingSchedule {
    pub schedule_id: String,
    pub project_id: String,
    pub beneficiary: String,
    pub total_amount: String,
    pub released_amount: String,
    pub start_time: DateTime<Utc>,
    pub end_time: DateTime<Utc>,
    pub cliff_period_days: u32,
    pub linear_unlock_days: u32,
    pub claimable_amount: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LaunchStats {
    pub project_id: String,
    pub total_raised: String,
    pub total_contributors: u64,
    pub unique_wallets: u64,
    pub average_contribution: String,
    pub hard_cap_percentage: f64,
    pub token_sold: String,
    pub participants_by_tier: HashMap<u8, u64>,
}

// ============================================================================
// Launchpad Service
// ============================================================================

pub struct LaunchpadService {
    projects: Arc<RwLock<HashMap<String, LaunchProject>>>,
    contributions: Arc<RwLock<HashMap<String, Vec<Contribution>>>>,
    user_tiers: Arc<RwLock<HashMap<String, UserTier>>>,
    vesting_schedules: Arc<RwLock<HashMap<String, Vec<VestingSchedule>>>>,
}

impl LaunchpadService {
    pub fn new() -> Self {
        LaunchpadService {
            projects: Arc::new(RwLock::new(HashMap::new())),
            contributions: Arc::new(RwLock::new(HashMap::new())),
            user_tiers: Arc::new(RwLock::new(HashMap::new())),
            vesting_schedules: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    // ========================================================================
    // Project Management
    // ========================================================================

    pub fn create_project(&self, mut project: LaunchProject) -> Result<LaunchProject, String> {
        project.project_id = Uuid::new_v4().to_string();
        project.status = LaunchStatus::Pending;
        project.created_at = Utc::now();
        project.updated_at = Utc::now();
        
        let mut projects = self.projects.write().map_err(|e| e.to_string())?;
        projects.insert(project.project_id.clone(), project.clone());
        
        Ok(project)
    }

    pub fn get_project(&self, project_id: &str) -> Result<Option<LaunchProject>, String> {
        let projects = self.projects.read().map_err(|e| e.to_string())?;
        Ok(projects.get(project_id).cloned())
    }

    pub fn update_project_status(&self, project_id: &str, status: LaunchStatus) -> Result<(), String> {
        let mut projects = self.projects.write().map_err(|e| e.to_string())?;
        
        if let Some(project) = projects.get_mut(project_id) {
            project.status = status;
            project.updated_at = Utc::now();
            Ok(())
        } else {
            Err("Project not found".to_string())
        }
    }

    pub fn list_projects(&self, status_filter: Option<LaunchStatus>) -> Result<Vec<LaunchProject>, String> {
        let projects = self.projects.read().map_err(|e| e.to_string())?;
        
        let mut result: Vec<LaunchProject> = projects.values().cloned().collect();
        
        if let Some(status) = status_filter {
            result.retain(|p| p.status == status);
        }
        
        result.sort_by(|a, b| b.start_time.cmp(&a.start_time));
        
        Ok(result)
    }

    pub fn get_upcoming_projects(&self) -> Result<Vec<LaunchProject>, String> {
        let now = Utc::now();
        let projects = self.projects.read().map_err(|e| e.to_string())?;
        
        let result: Vec<LaunchProject> = projects
            .values()
            .filter(|p| p.start_time > now && p.status == LaunchStatus::Upcoming)
            .cloned()
            .collect();
        
        Ok(result)
    }

    pub fn get_active_projects(&self) -> Result<Vec<LaunchProject>, String> {
        let projects = self.projects.read().map_err(|e| e.to_string())?;
        
        let result: Vec<LaunchProject> = projects
            .values()
            .filter(|p| p.status == LaunchStatus::Active)
            .cloned()
            .collect();
        
        Ok(result)
    }

    // ========================================================================
    // Contribution Management
    // ========================================================================

    pub fn contribute(
        &self,
        project_id: &str,
        user_id: &str,
        amount: &str,
    ) -> Result<Contribution, String> {
        // Get project
        let project = {
            let projects = self.projects.read().map_err(|e| e.to_string())?;
            projects.get(project_id).cloned()
        }.ok_or("Project not found")?;

        // Check if project is active
        if project.status != LaunchStatus::Active {
            return Err("Project is not active for contributions".to_string());
        }

        // Check timing
        let now = Utc::now();
        if now < project.start_time {
            return Err("Project has not started yet".to_string());
        }
        if now > project.end_time {
            return Err("Project has ended".to_string());
        }

        // Check user tier/allocation
        let user_tier = self.get_user_tier(user_id)?;
        
        // Parse amount
        let amount_float: f64 = amount.parse().map_err(|_| "Invalid amount")?;
        
        // Check allocation
        if let Some(tier) = user_tier {
            let max_alloc: f64 = tier.max_allocation.parse().map_err(|_| "Invalid allocation")?;
            let used: f64 = tier.used_allocation.parse().map_err(|_| "Invalid allocation")?;
            
            if amount_float + used > max_alloc {
                return Err("Exceeds allocation limit".to_string());
            }
        }

        // Calculate token amount
        let token_price: f64 = project.token_price.parse().map_err(|_| "Invalid price")?;
        let token_amount = amount_float / token_price;

        // Create contribution
        let contribution = Contribution {
            contribution_id: Uuid::new_v4().to_string(),
            project_id: project_id.to_string(),
            user_id: user_id.to_string(),
            amount: amount.to_string(),
            token_amount: format!("{:.8}", token_amount),
            tx_hash: generate_tx_hash(),
            status: "pending".to_string(),
            timestamp: Utc::now(),
        };

        // Store contribution
        {
            let mut contributions = self.contributions.write().map_err(|e| e.to_string())?;
            contributions
                .entry(project_id.to_string())
                .or_insert_with(Vec::new)
                .push(contribution.clone());
        }

        // Update project stats
        {
            let mut projects = self.projects.write().map_err(|e| e.to_string())?;
            if let Some(p) = projects.get_mut(project_id) {
                let raised: f64 = p.raised_amount.parse().unwrap_or(0.0);
                p.raised_amount = format!("{:.8}", raised + amount_float);
                p.total_contributors += 1;
            }
        }

        // Update user tier
        if let Some(mut tier) = user_tier {
            let used: f64 = tier.used_allocation.parse().unwrap_or(0.0);
            tier.used_allocation = format!("{:.8}", used + amount_float);
            
            let mut tiers = self.user_tiers.write().map_err(|e| e.to_string())?;
            tiers.insert(user_id.to_string(), tier);
        }

        Ok(contribution)
    }

    pub fn get_contributions(&self, project_id: &str) -> Result<Vec<Contribution>, String> {
        let contributions = self.contributions.read().map_err(|e| e.to_string())?;
        Ok(contributions.get(project_id).cloned().unwrap_or_default())
    }

    pub fn get_user_contributions(&self, user_id: &str) -> Result<Vec<Contribution>, String> {
        let contributions = self.contributions.read().map_err(|e| e.to_string())?;
        
        let mut result: Vec<Contribution> = Vec::new();
        for (_, contribs) in contributions.iter() {
            for c in contribs.iter() {
                if c.user_id == user_id {
                    result.push(c.clone());
                }
            }
        }
        
        Ok(result)
    }

    // ========================================================================
    // Tier Management
    // ========================================================================

    pub fn set_user_tier(&self, user_id: &str, tier: u8, allocation: &str) -> Result<UserTier, String> {
        let user_tier = UserTier {
            user_id: user_id.to_string(),
            tier,
            allocation: allocation.to_string(),
            used_allocation: "0".to_string(),
            remaining_allocation: allocation.to_string(),
            is_whitelisted: true,
            whitelist_phase: None,
        };
        
        let mut tiers = self.user_tiers.write().map_err(|e| e.to_string())?;
        tiers.insert(user_id.to_string(), user_tier.clone());
        
        Ok(user_tier)
    }

    pub fn get_user_tier(&self, user_id: &str) -> Result<Option<UserTier>, String> {
        let tiers = self.user_tiers.read().map_err(|e| e.to_string())?;
        Ok(tiers.get(user_id).cloned())
    }

    pub fn add_to_whitelist(&self, user_id: &str, phase: SalePhase) -> Result<(), String> {
        let mut tiers = self.user_tiers.write().map_err(|e| e.to_string())?;
        
        if let Some(tier) = tiers.get_mut(user_id) {
            tier.is_whitelisted = true;
            tier.whitelist_phase = Some(phase);
        }
        
        Ok(())
    }

    // ========================================================================
    // Vesting
    // ========================================================================

    pub fn create_vesting_schedule(
        &self,
        project_id: &str,
        beneficiary: &str,
        total_amount: &str,
        start_time: DateTime<Utc>,
        months: u32,
    ) -> Result<VestingSchedule, String> {
        let end_time = start_time + chrono::Duration::days((months * 30).into());
        
        let schedule = VestingSchedule {
            schedule_id: Uuid::new_v4().to_string(),
            project_id: project_id.to_string(),
            beneficiary: beneficiary.to_string(),
            total_amount: total_amount.to_string(),
            released_amount: "0".to_string(),
            start_time,
            end_time,
            cliff_period_days: 30,
            linear_unlock_days: (months * 30) - 30,
            claimable_amount: "0".to_string(),
        };
        
        let mut schedules = self.vesting_schedules.write().map_err(|e| e.to_string())?;
        schedules
            .entry(project_id.to_string())
            .or_insert_with(Vec::new)
            .push(schedule.clone());
        
        Ok(schedule)
    }

    pub fn get_claimable_amount(&self, schedule_id: &str) -> Result<String, String> {
        let schedules = self.vesting_schedules.read().map_err(|e| e.to_string())?;
        
        for (_, scheds) in schedules.iter() {
            for schedule in scheds.iter() {
                if schedule.schedule_id == schedule_id {
                    let now = Utc::now();
                    
                    // Check cliff
                    if now < schedule.start_time + chrono::Duration::days(schedule.cliff_period_days.into()) {
                        return Ok("0".to_string());
                    }
                    
                    // Calculate vested amount
                    let total: f64 = schedule.total_amount.parse().unwrap_or(0.0);
                    let days_since_start = (now - schedule.start_time).num_days() as f64;
                    let vesting_days = schedule.linear_unlock_days as f64;
                    
                    let vested = if days_since_start >= vesting_days {
                        total
                    } else {
                        total * (days_since_start / vesting_days)
                    };
                    
                    let released: f64 = schedule.released_amount.parse().unwrap_or(0.0);
                    let claimable = vested - released;
                    
                    return Ok(format!("{:.8}", claimable));
                }
            }
        }
        
        Err("Schedule not found".to_string())
    }

    // ========================================================================
    // Analytics
    // ========================================================================

    pub fn get_project_stats(&self, project_id: &str) -> Result<LaunchStats, String> {
        let project = {
            let projects = self.projects.read().map_err(|e| e.to_string())?;
            projects.get(project_id).cloned()
        }.ok_or("Project not found")?;

        let contributions = {
            let contribs = self.contributions.read().map_err(|e| e.to_string())?;
            contribs.get(project_id).cloned().unwrap_or_default()
        };

        let mut participants_by_tier: HashMap<u8, u64> = HashMap::new();
        let mut unique_wallets: std::collections::HashSet<String> = std::collections::HashSet::new();

        for c in &contributions {
            unique_wallets.insert(c.user_id.clone());
        }

        let total_raised: f64 = project.raised_amount.parse().unwrap_or(0.0);
        let hard_cap: f64 = project.hard_cap.parse().unwrap_or(1.0);
        let average = if contributions.is_empty() {
            0.0
        } else {
            total_raised / contributions.len() as f64
        };

        Ok(LaunchStats {
            project_id: project_id.to_string(),
            total_raised: project.raised_amount.clone(),
            total_contributors: project.total_contributors,
            unique_wallets: unique_wallets.len() as u64,
            average_contribution: format!("{:.8}", average),
            hard_cap_percentage: (total_raised / hard_cap) * 100.0,
            token_sold: format!("{:.8}", total_raised / project.token_price.parse::<f64>().unwrap_or(1.0)),
            participants_by_tier,
        })
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

fn generate_tx_hash() -> String {
    let mut hasher = Sha256::new();
    hasher.update(Uuid::new_v4().to_string().as_bytes());
    hasher.update(Utc::now().timestamp_nanos_opt().unwrap_or(0).to_string().as_bytes());
    format!("0x{:x}", hasher.finalize())
}

// ============================================================================
// Main
// ============================================================================

#[tokio::main]
async fn main() {
    env_logger::init();
    
    let service = LaunchpadService::new();
    
    // Create sample project
    let project = LaunchProject {
        project_id: String::new(),
        name: "TigerLaunch Token".to_string(),
        symbol: "TIGER".to_string(),
        description: "A revolutionary DeFi token".to_string(),
        logo_url: "https://example.com/logo.png".to_string(),
        banner_url: "https://example.com/banner.png".to_string(),
        website: "https://tigerwallet.io".to_string(),
        whitepaper: "https://tigerwallet.io/whitepaper.pdf".to_string(),
        social_links: HashMap::new(),
        token: Token {
            token_id: "token_001".to_string(),
            name: "TigerLaunch".to_string(),
            symbol: "TIGER".to_string(),
            decimals: 18,
            total_supply: "1000000000".to_string(),
            contract_address: "0x742d35Cc6634C0532925a3b844Bc9e7595f".to_string(),
            chain_id: 1,
            logo_url: "".to_string(),
            description: "".to_string(),
        },
        sale_token_supply: "100000000".to_string(),
        token_price: "0.001".to_string(),
        min_allocation: "100".to_string(),
        max_allocation: "10000".to_string(),
        launch_type: LaunchType::IDO,
        status: LaunchStatus::Active,
        start_time: Utc::now(),
        end_time: Utc::now() + chrono::Duration::days(7),
        hard_cap: "1000000".to_string(),
        soft_cap: "100000".to_string(),
        raised_amount: "0".to_string(),
        total_contributors: 0,
        liquidity_percentage: 80.0,
        liquidity_lock_days: 365,
        listing_price: "0.0015".to_string(),
        team_vesting_percentage: 15.0,
        team_vesting_months: 12,
        advisor_vesting_percentage: 5.0,
        advisor_vesting_months: 6,
        is_verified: true,
        kyc_status: "verified".to_string(),
        audit_status: "completed".to_string(),
        created_at: Utc::now(),
        updated_at: Utc::now(),
    };
    
    match service.create_project(project) {
        Ok(created) => {
            println!("Created project: {}", created.project_id);
            
            // Set user tier
            service.set_user_tier("user_001", 2, "5000").unwrap();
            
            // Contribute
            match service.contribute(&created.project_id, "user_001", "1000") {
                Ok(contrib) => {
                    println!("Contribution: {} - {} tokens", contrib.contribution_id, contrib.token_amount);
                }
                Err(e) => println!("Contribution error: {}", e),
            }
            
            // Get stats
            match service.get_project_stats(&created.project_id) {
                Ok(stats) => {
                    println!("Stats - Raised: {}, Contributors: {}", 
                        stats.total_raised, stats.total_contributors);
                }
                Err(e) => println!("Stats error: {}", e),
            }
        }
        Err(e) => println!("Error creating project: {}", e),
    }
    
    // List active projects
    let active = service.get_active_projects().unwrap();
    println!("Active projects: {}", active.len());
}
