/**
 * TigerWallet Reputation Service
 * High-Security Rust Implementation
 */

use std::collections::HashMap;
use std::sync::{Arc, RwLock};

#[derive(Debug, Clone)]
pub struct Reputation {
    pub user_id: String,
    pub score: f64,
    pub trades: u64,
    pub verified: bool,
}

pub struct ReputationService {
    reputations: RwLock<HashMap<String, Reputation>>,
}

impl ReputationService {
    pub fn new() -> Self {
        Self { reputations: RwLock::new(HashMap::new()) }
    }

    pub fn update(&self, user_id: &str, score_delta: f64) {
        let mut reps = self.reputations.write().unwrap();
        if let Some(rep) = reps.get_mut(user_id) {
            rep.score += score_delta;
            rep.trades += 1;
        } else {
            reps.insert(user_id.to_string(), Reputation {
                user_id: user_id.to_string(),
                score: 50.0 + score_delta,
                trades: 1,
                verified: false,
            });
        }
    }

    pub fn get(&self, user_id: &str) -> Option<Reputation> {
        self.reputations.read().unwrap().get(user_id).cloned()
    }
}

fn main() {
    println!("Reputation Service on :8101");
    
    let service = Arc::new(ReputationService::new());
    
    service.update("user1", 10.0);
    service.update("user2", 5.0);
    
    if let Some(rep) = service.get("user1") {
        println!("User1 score: {} ({} trades)", rep.score, rep.trades);
    }
}
