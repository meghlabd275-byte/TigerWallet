//! TigerWallet Portfolio Pro
use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Portfolio {
    pub user_id: String,
    pub total_value: Decimal,
    pub positions: Vec<Position>,
    pub pnl: PnL,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    pub symbol: String,
    pub quantity: Decimal,
    pub avg_cost: Decimal,
    pub current_price: Decimal,
    pub value: Decimal,
    pub unrealized_pnl: Decimal,
    pub cost_basis: Decimal,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PnL {
    pub realized: Decimal,
    pub unrealized: Decimal,
    pub total: Decimal,
    pub roi_percent: Decimal,
}

impl Portfolio {
    pub fn new(user_id: &str) -> Self {
        Self {
            user_id: user_id.to_string(),
            total_value: Decimal::from(0),
            positions: Vec::new(),
            pnl: PnL::default(),
        }
    }

    pub fn calculate_pnl(&mut self) {
        let mut total_unrealized = Decimal::from(0);
        let mut total_cost = Decimal::from(0);

        for pos in &mut self.positions {
            pos.unrealized_pnl = (pos.current_price - pos.avg_cost) * pos.quantity;
            total_unrealized += pos.unrealized_pnl;
            total_cost += pos.cost_basis;
        }

        let total = self.pnl.realized + total_unrealized;
        let roi = if total_cost > Decimal::from(0) {
            (total / total_cost) * Decimal::from(100)
        } else {
            Decimal::from(0)
        };

        self.pnl = PnL {
            realized: self.pnl.realized,
            unrealized: total_unrealized,
            total,
            roi_percent: roi,
        };
    }
}

impl Default for PnL {
    fn default() -> Self {
        Self {
            realized: Decimal::from(0),
            unrealized: Decimal::from(0),
            total: Decimal::from(0),
            roi_percent: Decimal::from(0),
        }
    }
}