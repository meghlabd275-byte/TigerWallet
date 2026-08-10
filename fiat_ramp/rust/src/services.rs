//! Services for Fiat Ramp - Core business logic

use crate::error::Error;
use crate::models::*;
use chrono::{DateTime, Utc};
use std::sync::{Arc, RwLock};
use std::collections::HashMap;
use uuid::Uuid;

/// KYC Service
pub struct KycService {
    applications: RwLock<HashMap<String, KycApplication>>,
}

impl KycService {
    pub fn new() -> Self {
        Self {
            applications: RwLock::new(HashMap::new()),
        }
    }

    pub fn start_kyc(&self, req: StartKycRequest) -> Result<KycApplication, Error> {
        let app = KycApplication {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            email: req.email,
            phone: req.phone,
            country: req.country,
            level: match req.level {
                0 => KycLevel::None,
                1 => KycLevel::Basic,
                2 => KycLevel::Medium,
                3 => KycLevel::High,
                _ => KycLevel::Max,
            },
            status: KycStatus::Pending,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };
        
        let mut apps = self.applications.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        apps.insert(app.id.clone(), app.clone());
        
        Ok(app)
    }

    pub fn get_status(&self, user_id: &str) -> Result<KycApplication, Error> {
        let apps = self.applications.read()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        apps.values()
            .find(|a| a.user_id == user_id)
            .cloned()
            .ok_or_else(|| Error::NotFound(format!("KYC not found for user {}", user_id)))
    }

    pub fn verify(&self, user_id: &str, req: VerifyKycRequest) -> Result<KycApplication, Error> {
        let mut apps = self.applications.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        if let Some(app) = apps.values_mut().find(|a| a.user_id == user_id) {
            app.status = KycStatus::Verified;
            app.updated_at = Utc::now();
            return Ok(app.clone());
        }
        
        Err(Error::NotFound(format!("KYC not found for user {}", user_id)))
    }

    pub fn upload_document(&self, user_id: &str, req: UploadDocumentRequest) -> Result<KycApplication, Error> {
        let mut apps = self.applications.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        if let Some(app) = apps.values_mut().find(|a| a.user_id == user_id) {
            app.status = KycStatus::InProgress;
            app.updated_at = Utc::now();
            return Ok(app.clone());
        }
        
        Err(Error::NotFound(format!("KYC not found for user {}", user_id)))
    }
}

impl Default for KycService {
    fn default() -> Self {
        Self::new()
    }
}

/// Banking Service
pub struct BankingService {
    accounts: RwLock<HashMap<String, Vec<BankAccount>>>,
}

impl BankingService {
    pub fn new() -> Self {
        Self {
            accounts: RwLock::new(HashMap::new()),
        }
    }

    pub fn get_accounts(&self, user_id: &str) -> Result<Vec<BankAccount>, Error> {
        let accounts = self.accounts.read()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        Ok(accounts.get(user_id).cloned().unwrap_or_default())
    }

    pub fn add_account(&self, req: AddBankAccountRequest) -> Result<BankAccount, Error> {
        let user_id = req.user_id.clone();
        let account = BankAccount {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            bank_name: req.bank_name,
            account_num: req.account_num,
            routing_num: req.routing_num,
            country: req.country,
            currency: req.currency,
            created_at: Utc::now(),
        };

        let mut accounts = self.accounts.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;

        accounts.entry(user_id.clone()).or_insert_with(Vec::new);
        if let Some(user_accounts) = accounts.get_mut(&user_id) {
            user_accounts.push(account.clone());
        }
        
        Ok(account)
    }

    pub fn withdraw(&self, req: WithdrawalRequest) -> Result<Withdrawal, Error> {
        let withdrawal = Withdrawal {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            amount: req.amount,
            currency: req.currency,
            bank_account: req.bank_account,
            status: WithdrawalStatus::Pending,
            created_at: Utc::now(),
        };
        
        Ok(withdrawal)
    }
}

impl Default for BankingService {
    fn default() -> Self {
        Self::new()
    }
}

/// Payment Service
pub struct PaymentService {
    quotes: RwLock<HashMap<String, Quote>>,
    payments: RwLock<HashMap<String, Payment>>,
}

impl PaymentService {
    pub fn new() -> Self {
        Self {
            quotes: RwLock::new(HashMap::new()),
            payments: RwLock::new(HashMap::new()),
        }
    }

    pub fn get_buy_quote(&self, req: QuoteRequest) -> Result<Quote, Error> {
        let rate = 1.0; // Simplified rate calculation
        let quote = Quote {
            id: Uuid::new_v4().to_string(),
            fiat_amt: req.fiat_amt,
            crypto_amt: req.crypto_amt,
            rate,
            expires_at: Utc::now() + chrono::Duration::minutes(10),
        };
        
        let mut quotes = self.quotes.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        quotes.insert(quote.id.clone(), quote.clone());
        
        Ok(quote)
    }

    pub fn get_sell_quote(&self, req: QuoteRequest) -> Result<Quote, Error> {
        self.get_buy_quote(req)
    }

    pub fn execute_buy(&self, req: ExecuteBuyRequest) -> Result<Payment, Error> {
        let payment = Payment {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            order_id: req.quote_id,
            amount: req.fiat_amt,
            currency: req.crypto,
            method: req.method,
            status: PaymentStatus::Processing,
            created_at: Utc::now(),
        };
        
        let mut payments = self.payments.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        payments.insert(payment.id.clone(), payment.clone());
        
        Ok(payment)
    }

    pub fn execute_sell(&self, req: ExecuteSellRequest) -> Result<Payment, Error> {
        let payment = Payment {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            order_id: req.quote_id,
            amount: req.crypto_amt,
            currency: req.crypto,
            method: req.method,
            status: PaymentStatus::Processing,
            created_at: Utc::now(),
        };
        
        let mut payments = self.payments.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        payments.insert(payment.id.clone(), payment.clone());
        
        Ok(payment)
    }

    pub fn get_methods(&self) -> Result<Vec<PaymentMethod>, Error> {
        Ok(vec![
            PaymentMethod {
                id: "bank_transfer".to_string(),
                name: "Bank Transfer".to_string(),
                method_type: "bank".to_string(),
                supported_fiats: vec!["USD".to_string(), "EUR".to_string()],
                supported_cryptos: vec!["BTC".to_string(), "ETH".to_string()],
                fees: 0.0,
            },
            PaymentMethod {
                id: "card".to_string(),
                name: "Credit/Debit Card".to_string(),
                method_type: "card".to_string(),
                supported_fiats: vec!["USD".to_string(), "EUR".to_string()],
                supported_cryptos: vec!["BTC".to_string(), "ETH".to_string()],
                fees: 2.5,
            },
        ])
    }

    pub fn create(&self, req: CreatePaymentRequest) -> Result<Payment, Error> {
        let payment = Payment {
            id: Uuid::new_v4().to_string(),
            user_id: req.user_id,
            order_id: req.order_id,
            amount: req.amount,
            currency: req.currency,
            method: req.method,
            status: PaymentStatus::Pending,
            created_at: Utc::now(),
        };
        
        let mut payments = self.payments.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        payments.insert(payment.id.clone(), payment.clone());
        
        Ok(payment)
    }

    pub fn get_status(&self, order_id: &str) -> Result<Payment, Error> {
        let payments = self.payments.read()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        payments.values()
            .find(|p| p.order_id == order_id)
            .cloned()
            .ok_or_else(|| Error::NotFound(format!("Payment not found: {}", order_id)))
    }
}

impl Default for PaymentService {
    fn default() -> Self {
        Self::new()
    }
}

/// Limits Service
pub struct LimitsService {
    limits: RwLock<HashMap<String, Limits>>,
}

impl LimitsService {
    pub fn new() -> Self {
        Self {
            limits: RwLock::new(HashMap::new()),
        }
    }

    pub fn get(&self, user_id: &str) -> Result<Limits, Error> {
        let limits = self.limits.read()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        
        Ok(limits.get(user_id).cloned().unwrap_or(Limits {
            level: 0,
            daily_limit: 1000.0,
            monthly_limit: 10000.0,
            yearly_limit: 50000.0,
        }))
    }

    pub fn update(&self, user_id: &str, req: UpdateLimitsRequest) -> Result<Limits, Error> {
        let limits = Limits {
            level: req.level,
            daily_limit: req.daily_limit,
            monthly_limit: req.monthly_limit,
            yearly_limit: req.yearly_limit,
        };
        
        let mut limits_map = self.limits.write()
            .map_err(|_| Error::Internal("Lock error".to_string()))?;
        limits_map.insert(user_id.to_string(), limits.clone());
        
        Ok(limits)
    }
}

impl Default for LimitsService {
    fn default() -> Self {
        Self::new()
    }
}

/// Compliance Service
pub struct ComplianceService {
    screen_results: RwLock<HashMap<String, ScreenResult>>,
}

impl ComplianceService {
    pub fn new() -> Self {
        Self {
            screen_results: RwLock::new(HashMap::new()),
        }
    }

    pub fn screen(&self, req: ScreenRequest) -> Result<ScreenResult, Error> {
        // Simplified compliance check
        let result = ScreenResult {
            approved: req.amount < 10000.0,
            risk_score: if req.amount > 10000.0 { 0.8 } else { 0.1 },
            flags: if req.amount > 10000.0 { vec!["high_value".to_string()] } else { vec![] },
        };
        
        Ok(result)
    }

    pub fn report(&self, req: ReportRequest) -> Result<(), Error> {
        // Simplified reporting
        Ok(())
    }
}

impl Default for ComplianceService {
    fn default() -> Self {
        Self::new()
    }
}