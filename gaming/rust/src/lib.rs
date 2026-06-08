//! TigerWallet Gaming Platform
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Game asset
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GameAsset {
    pub id: String,
    pub game_id: String,
    pub owner: String,
    pub asset_type: AssetType,
    pub metadata: String,
    pub level: u32,
    pub experience: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AssetType {
    Character,
    Weapon,
    Armor,
    Consumable,
    Land,
    NFT,
}

/// Inventory
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Inventory {
    pub user_id: String,
    pub game_id: String,
    pub items: Vec<GameAsset>,
    pub capacity: u32,
}

impl Inventory {
    pub fn new(user_id: &str, game_id: &str, capacity: u32) -> Self {
        Self {
            user_id: user_id.to_string(),
            game_id: game_id.to_string(),
            items: Vec::new(),
            capacity,
        }
    }

    pub fn add_item(&mut self, asset: GameAsset) -> Result<(), InventoryError> {
        if self.items.len() >= self.capacity as usize {
            return Err(InventoryError::Full);
        }
        self.items.push(asset);
        Ok(())
    }

    pub fn remove_item(&mut self, asset_id: &str) -> Option<GameAsset> {
        if let Some(pos) = self.items.iter().position(|a| a.id == asset_id) {
            Some(self.items.remove(pos))
        } else {
            None
        }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum InventoryError {
    #[error("Inventory full")]
    Full,
    #[error("Item not found")]
    NotFound,
}

/// Leaderboard entry
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaderboardEntry {
    pub rank: u32,
    pub user_id: String,
    pub score: u64,
    pub metadata: String,
}

/// Rewards system
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Rewards {
    pub user_id: String,
    pub game_id: String,
    pub xp: u64,
    pub level: u32,
    pub coins: u64,
    pub achievements: Vec<String>,
}

impl Rewards {
    pub fn new(user_id: &str, game_id: &str) -> Self {
        Self {
            user_id: user_id.to_string(),
            game_id: game_id.to_string(),
            xp: 0,
            level: 1,
            coins: 0,
            achievements: Vec::new(),
        }
    }

    pub fn add_xp(&mut self, xp: u64) {
        self.xp += xp;
        
        // Level up every 1000 XP
        while self.xp >= 1000 {
            self.xp -= 1000;
            self.level += 1;
        }
    }

    pub fn add_coins(&mut self, amount: u64) {
        self.coins += amount;
    }
}

/// Real-time transaction for gaming
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GameTransaction {
    pub id: String,
    pub user_id: String,
    pub game_id: String,
    pub tx_type: TransactionType,
    pub amount: i64,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum TransactionType {
    Purchase,
    Sale,
    Reward,
    Bet,
    Win,
    Loss,
}