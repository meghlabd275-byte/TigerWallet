//! NFT Marketplace Module - Rust Implementation
//! Handles NFT trading, offers, and auctions

use std::sync::Arc;
use tokio::sync::RwLock;
use thiserror::Error;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

#[derive(Error, Debug)]
pub enum MarketplaceError {
    #[error("Offer not found")]
    OfferNotFound,
    #[error("Offer expired")]
    OfferExpired,
    #[error("Insufficient funds")]
    InsufficientFunds,
    #[error("Not the owner")]
    NotOwner,
    #[error("Price mismatch")]
    PriceMismatch,
    #[error("Invalid offer")]
    InvalidOffer,
}

/// Offer status
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub enum OfferStatus {
    Open,
    Cancelled,
    Accepted,
    Completed,
    Expired,
}

/// Offer type
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub enum OfferType {
    FixedPrice,
    Auction,
    Bundle,
}

/// NFT Offer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTOffer {
    pub offer_id: String,
    pub contract_address: String,
    pub token_id: String,
    pub seller: String,
    pub buyer: Option<String>,
    pub offer_type: OfferType,
    pub price: String,
    pub price_token: String,
    pub chain_id: u64,
    pub status: OfferStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub accepted_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
}

/// Auction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTAuction {
    pub auction_id: String,
    pub contract_address: String,
    pub token_id: String,
    pub seller: String,
    pub offer_type: OfferType,
    pub starting_price: String,
    pub current_price: String,
    pub buy_now_price: Option<String>,
    pub price_token: String,
    pub chain_id: u64,
    pub highest_bidder: Option<String>,
    pub bids: Vec<NFTAuctionBid>,
    pub status: OfferStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

/// Auction bid
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTAuctionBid {
    pub bidder: String,
    pub amount: String,
    pub timestamp: DateTime<Utc>,
}

/// Bundle offer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTBundle {
    pub bundle_id: String,
    pub seller: String,
    pub items: Vec<NFTBundleItem>,
    pub total_price: String,
    pub price_token: String,
    pub chain_id: u64,
    pub status: OfferStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
}

/// Bundle item
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTBundleItem {
    pub contract_address: String,
    pub token_id: String,
}

/// Marketplace orderbook
pub struct OrderBook {
    offers: Arc<RwLock<Vec<NFTOffer>>>,
    auctions: Arc<RwLock<Vec<NFTAuction>>>,
    bundles: Arc<RwLock<Vec<NFTBundle>>>,
}

impl Default for OrderBook {
    fn default() -> Self {
        Self::new()
    }
}

impl OrderBook {
    pub fn new() -> Self {
        Self {
            offers: Arc::new(RwLock::new(Vec::new())),
            auctions: Arc::new(RwLock::new(Vec::new())),
            bundles: Arc::new(RwLock::new(Vec::new())),
        }
    }

    /// Create a new offer
    pub async fn create_offer(&self, offer: NFTOffer) -> Result<String, MarketplaceError> {
        // Validate offer
        if offer.price.parse::<u128>().is_err() {
            return Err(MarketplaceError::InvalidOffer);
        }

        if offer.expires_at <= Utc::now() {
            return Err(MarketplaceError::OfferExpired);
        }

        let mut offers = self.offers.write().await;
        offers.push(offer);

        Ok(offer.offer_id.clone())
    }

    /// Accept an offer
    pub async fn accept_offer(&self, offer_id: &str, buyer: &str) -> Result<NFTOffer, MarketplaceError> {
        let mut offers = self.offers.write().await;
        
        let offer = offers
            .iter_mut()
            .find(|o| o.offer_id == offer_id)
            .ok_or(MarketplaceError::OfferNotFound)?;

        // Check if expired
        if offer.expires_at <= Utc::now() {
            return Err(MarketplaceError::OfferExpired);
        }

        // Check status
        if offer.status != OfferStatus::Open {
            return Err(MarketplaceError::InvalidOffer);
        }

        // Update offer
        offer.buyer = Some(buyer.to_string());
        offer.status = OfferStatus::Accepted;
        offer.accepted_at = Some(Utc::now());

        Ok(offer.clone())
    }

    /// Cancel an offer
    pub async fn cancel_offer(&self, offer_id: &str, seller: &str) -> Result<(), MarketplaceError> {
        let mut offers = self.offers.write().await;
        
        let offer = offers
            .iter_mut()
            .find(|o| o.offer_id == offer_id && o.seller == seller)
            .ok_or(MarketplaceError::OfferNotFound)?;

        if offer.status != OfferStatus::Open {
            return Err(MarketplaceError::InvalidOffer);
        }

        offer.status = OfferStatus::Cancelled;

        Ok(())
    }

    /// Get active offers for a collection
    pub async fn get_collection_offers(&self, contract: &str) -> Vec<NFTOffer> {
        let offers = self.offers.read().await;
        
        offers
            .iter()
            .filter(|o| o.contract_address == contract && o.status == OfferStatus::Open && o.expires_at > Utc::now())
            .cloned()
            .collect()
    }

    /// Get offers for an NFT
    pub async fn get_nft_offers(&self, contract: &str, token_id: &str) -> Vec<NFTOffer> {
        let offers = self.offers.read().await;
        
        offers
            .iter()
            .filter(|o| o.contract_address == contract && o.token_id == token_id && o.status == OfferStatus::Open && o.expires_at > Utc::now())
            .cloned()
            .collect()
    }

    /// Get user's offers
    pub async fn get_user_offers(&self, user: &str) -> Vec<NFTOffer> {
        let offers = self.offers.read().await;
        
        offers
            .iter()
            .filter(|o| o.seller == user || o.buyer.as_deref() == Some(user))
            .cloned()
            .collect()
    }

    /// Create auction
    pub async fn create_auction(&self, auction: NFTAuction) -> Result<String, MarketplaceError> {
        if auction.expires_at <= Utc::now() {
            return Err(MarketplaceError::OfferExpired);
        }

        let mut auctions = self.auctions.write().await;
        auctions.push(auction);

        Ok(auction.auction_id.clone())
    }

    /// Place bid
    pub async fn place_bid(&self, auction_id: &str, bidder: &str, amount: &str) -> Result<NFTAuction, MarketplaceError> {
        let amount_u128 = amount.parse::<u128>()
            .map_err(|_| MarketplaceError::InvalidOffer)?;

        let mut auctions = self.auctions.write().await;
        
        let auction = auctions
            .iter_mut()
            .find(|a| a.auction_id == auction_id)
            .ok_or(MarketplaceError::OfferNotFound)?;

        // Check if expired
        if auction.expires_at <= Utc::now() {
            return Err(MarketplaceError::OfferExpired);
        }

        // Check if bid is higher than current
        let current = auction.current_price.parse::<u128>().unwrap_or(0);
        if amount_u128 <= current {
            return Err(MarketplaceError::PriceMismatch);
        }

        // Add bid
        auction.bids.push(NFTAuctionBid {
            bidder: bidder.to_string(),
            amount: amount.to_string(),
            timestamp: Utc::now(),
        });

        auction.current_price = amount.to_string();
        auction.highest_bidder = Some(bidder.to_string());

        Ok(auction.clone())
    }

    /// Get active auctions
    pub async fn get_active_auctions(&self) -> Vec<NFTAuction> {
        let auctions = self.auctions.read().await;
        
        auctions
            .iter()
            .filter(|a| a.status == OfferStatus::Open && a.expires_at > Utc::now())
            .cloned()
            .collect()
    }

    /// Create bundle
    pub async fn create_bundle(&self, bundle: NFTBundle) -> Result<String, MarketplaceError> {
        if bundle.items.is_empty() {
            return Err(MarketplaceError::InvalidOffer);
        }

        if bundle.expires_at <= Utc::now() {
            return Err(MarketplaceError::OfferExpired);
        }

        let mut bundles = self.bundles.write().await;
        bundles.push(bundle);

        Ok(bundle.bundle_id.clone())
    }

    /// Get floor price for collection
    pub async fn get_floor_price(&self, contract: &str) -> Option<String> {
        let offers = self.offers.read().await;
        
        let prices: Vec<u128> = offers
            .iter()
            .filter(|o| o.contract_address == contract && o.status == OfferStatus::Open && o.expires_at > Utc::now())
            .filter_map(|o| o.price.parse().ok())
            .collect();

        prices.into_iter().min().map(|p| p.to_string())
    }

    /// Get average price for collection
    pub async fn get_average_price(&self, contract: &str) -> Option<String> {
        let offers = self.offers.read().await;
        
        let prices: Vec<u128> = offers
            .iter()
            .filter(|o| o.contract_address == contract && o.status == OfferStatus::Completed)
            .filter_map(|o| o.price.parse().ok())
            .collect();

        if prices.is_empty() {
            return None;
        }

        let sum: u128 = prices.iter().sum();
        let avg = sum / prices.len() as u128;

        Some(avg.to_string())
    }

    /// Get volume for collection
    pub async fn get_volume(&self, contract: &str) -> u128 {
        let offers = self.offers.read().await;
        
        offers
            .iter()
            .filter(|o| o.contract_address == contract && o.status == OfferStatus::Completed)
            .filter_map(|o| o.price.parse().ok())
            .sum()
    }
}

/// Marketplace main struct
pub struct Marketplace {
    orderbook: OrderBook,
}

impl Default for Marketplace {
    fn default() -> Self {
        Self::new()
    }
}

impl Marketplace {
    pub fn new() -> Self {
        Self {
            orderbook: OrderBook::new(),
        }
    }

    /// List NFT for sale
    pub async fn list_for_sale(&self, offer: NFTOffer) -> Result<String, MarketplaceError> {
        self.orderbook.create_offer(offer).await
    }

    /// Buy NFT
    pub async fn buy_nft(&self, offer_id: &str, buyer: &str) -> Result<NFTOffer, MarketplaceError> {
        self.orderbook.accept_offer(offer_id, buyer).await
    }

    /// Get listing
    pub async fn get_listings(&self, contract: &str) -> Vec<NFTOffer> {
        self.orderbook.get_collection_offers(contract).await
    }

    /// Get offers for specific NFT
    pub async fn get_nft_offers(&self, contract: &str, token_id: &str) -> Vec<NFTOffer> {
        self.orderbook.get_nft_offers(contract, token_id).await
    }

    /// Start auction
    pub async fn start_auction(&self, auction: NFTAuction) -> Result<String, MarketplaceError> {
        self.orderbook.create_auction(auction).await
    }

    /// Place bid
    pub async fn bid(&self, auction_id: &str, bidder: &str, amount: &str) -> Result<NFTAuction, MarketplaceError> {
        self.orderbook.place_bid(auction_id, bidder, amount).await
    }

    /// Get active auctions
    pub async fn get_auctions(&self) -> Vec<NFTAuction> {
        self.orderbook.get_active_auctions().await
    }

    /// Create bundle
    pub async fn create_bundle(&self, bundle: NFTBundle) -> Result<String, MarketplaceError> {
        self.orderbook.create_bundle(bundle).await
    }

    /// Get analytics
    pub async fn get_analytics(&self, contract: &str) -> MarketplaceAnalytics {
        MarketplaceAnalytics {
            floor_price: self.orderbook.get_floor_price(contract).await,
            average_price: self.orderbook.get_average_price(contract).await,
            volume: self.orderbook.get_volume(contract).await,
            active_listings: self.orderbook.get_collection_offers(contract).await.len(),
            active_auctions: self.orderbook.get_active_auctions().await.len(),
        }
    }
}

/// Marketplace analytics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketplaceAnalytics {
    pub floor_price: Option<String>,
    pub average_price: Option<String>,
    pub volume: u128,
    pub active_listings: usize,
    pub active_auctions: usize,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_create_offer() {
        let marketplace = Marketplace::new();
        
        let offer = NFTOffer {
            offer_id: "test123".to_string(),
            contract_address: "0x123".to_string(),
            token_id: "1".to_string(),
            seller: "0xseller".to_string(),
            buyer: None,
            offer_type: OfferType::FixedPrice,
            price: "100".to_string(),
            price_token: "0x0000000000000000000000000000000000000000".to_string(),
            chain_id: 1,
            status: OfferStatus::Open,
            created_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::days(1),
            accepted_at: None,
            completed_at: None,
        };

        let result = marketplace.list_for_sale(offer).await;
        assert!(result.is_ok());
    }
}