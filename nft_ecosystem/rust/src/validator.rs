//! NFT Validator Module - Rust Implementation
//! Validates NFT transactions and metadata

use std::sync::Arc;
use thiserror::Error;
use serde::{Deserialize, Serialize};

#[derive(Error, Debug)]
pub enum ValidationError {
    #[error("Invalid token ID")]
    InvalidTokenID,
    #[error("Invalid contract address")]
    InvalidContractAddress,
    #[error("Invalid owner")]
    InvalidOwner,
    #[error("Invalid metadata")]
    InvalidMetadata(String),
    #[error("Transfer not authorized")]
    NotAuthorized,
    #[error("Contract error: {0}")]
    ContractError(String),
}

/// NFT metadata structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTMetadata {
    pub token_id: String,
    pub name: String,
    pub description: String,
    pub image: String,
    pub animation_url: Option<String>,
    pub external_url: Option<String>,
    pub attributes: Vec<NFTTrait>,
}

/// NFT trait/attribute
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NFTTrait {
    pub trait_type: String,
    pub value: String,
    pub display_type: Option<String>,
}

/// Validation result
#[derive(Debug, Clone)]
pub struct ValidationResult {
    pub valid: bool,
    pub errors: Vec<String>,
    pub warnings: Vec<String>,
}

/// NFT Validator
pub struct NFTValidator {
    // Configuration
    max_name_length: usize,
    max_description_length: usize,
    allowed_image_extensions: Vec<String>,
    allowed_animation_extensions: Vec<String>,
}

impl Default for NFTValidator {
    fn default() -> Self {
        Self::new()
    }
}

impl NFTValidator {
    /// Create new validator
    pub fn new() -> Self {
        Self {
            max_name_length: 100,
            max_description_length: 1000,
            allowed_image_extensions: vec![
                "png".to_string(),
                "jpg".to_string(),
                "jpeg".to_string(),
                "gif".to_string(),
                "svg".to_string(),
                "webp".to_string(),
            ],
            allowed_animation_extensions: vec![
                "mp4".to_string(),
                "webm".to_string(),
                "glb".to_string(),
                "gltf".to_string(),
                "html".to_string(),
            ],
        }
    }

    /// Validate metadata
    pub fn validate_metadata(&self, metadata: &NFTMetadata) -> ValidationResult {
        let mut errors = Vec::new();
        let mut warnings = Vec::new();

        // Validate name
        if metadata.name.is_empty() {
            errors.push("Name is required".to_string());
        } else if metadata.name.len() > self.max_name_length {
            errors.push(format!("Name exceeds maximum length of {}", self.max_name_length));
        }

        // Validate description
        if metadata.description.len() > self.max_description_length {
            errors.push(format!("Description exceeds maximum length of {}", self.max_description_length));
        }

        // Validate image URL
        if metadata.image.is_empty() {
            warnings.push("Image URL is recommended".to_string());
        } else if !self.is_valid_url(&metadata.image) {
            errors.push("Invalid image URL".to_string());
        } else if !self.has_valid_extension(&metadata.image, &self.allowed_image_extensions) {
            warnings.push("Image format may not be widely supported".to_string());
        }

        // Validate animation URL if present
        if let Some(animation_url) = &metadata.animation_url {
            if !animation_url.is_empty() {
                if !self.is_valid_url(animation_url) {
                    errors.push("Invalid animation URL".to_string());
                } else if !self.has_valid_extension(animation_url, &self.allowed_animation_extensions) {
                    warnings.push("Animation format may not be widely supported".to_string());
                }
            }
        }

        // Validate external URL if present
        if let Some(external_url) = &metadata.external_url {
            if !external_url.is_empty() && !self.is_valid_url(external_url) {
                warnings.push("External URL may be invalid".to_string());
            }
        }

        // Validate attributes
        for (i, attr) in metadata.attributes.iter().enumerate() {
            if attr.trait_type.is_empty() {
                warnings.push(format!("Attribute {} has empty trait_type", i));
            }
            if attr.value.is_empty() {
                warnings.push(format!("Attribute {} has empty value", i));
            }
        }

        ValidationResult {
            valid: errors.is_empty(),
            errors,
            warnings,
        }
    }

    /// Validate token ID format
    pub fn validate_token_id(&self, token_id: &str) -> Result<(), ValidationError> {
        // Token ID should be a valid number or hex string
        if token_id.is_empty() {
            return Err(ValidationError::InvalidTokenID);
        }

        // Try parsing as number
        if token_id.parse::<u128>().is_ok() {
            return Ok(());
        }

        // Try parsing as hex
        if token_id.starts_with("0x") {
            if token_id[2..].parse::<u128>().is_ok() {
                return Ok(());
            }
        }

        Err(ValidationError::InvalidTokenID)
    }

    /// Validate contract address
    pub fn validate_contract_address(&self, address: &str) -> Result<(), ValidationError> {
        if address.is_empty() {
            return Err(ValidationError::InvalidContractAddress);
        }

        // Check if valid Ethereum address format
        if address.len() == 42 && address.starts_with("0x") {
            // Valid format
            return Ok(());
        }

        Err(ValidationError::InvalidContractAddress)
    }

    /// Validate owner address
    pub fn validate_owner(&self, owner: &str) -> Result<(), ValidationError> {
        if owner.is_empty() {
            return Err(ValidationError::InvalidOwner);
        }

        // Check if valid Ethereum address format
        if owner.len() == 42 && owner.starts_with("0x") {
            return Ok(());
        }

        Err(ValidationError::InvalidOwner)
    }

    /// Validate transfer authorization
    pub fn validate_transfer(
        &self,
        from: &str,
        to: &str,
        token_owner: &str,
        approved_operator: Option<&str>,
    ) -> Result<(), ValidationError> {
        // Validate addresses
        self.validate_owner(from)?;
        self.validate_owner(to)?;

        // Check authorization
        let is_owner = token_owner.eq_ignore_ascii_case(from);
        let is_approved = approved_operator
            .map(|op| op.eq_ignore_ascii_case(token_owner))
            .unwrap_or(false);

        if !is_owner && !is_approved {
            return Err(ValidationError::NotAuthorized);
        }

        // Prevent transfers to zero address
        if to == "0x0000000000000000000000000000000000000000" {
            return Err(ValidationError::InvalidOwner);
        }

        Ok(())
    }

    /// Batch validate NFTs
    pub fn batch_validate(&self, metadata_list: &[NFTMetadata]) -> Vec<ValidationResult> {
        metadata_list
            .iter()
            .map(|m| self.validate_metadata(m))
            .collect()
    }

    // Helper: Check if URL is valid
    fn is_valid_url(&self, url: &str) -> bool {
        url.starts_with("http://") || url.starts_with("https://") || url.starts_with("ipfs://")
    }

    // Helper: Check file extension
    fn has_valid_extension(&self, url: &str, allowed: &[String]) -> bool {
        if let Some(ext) = url.rsplit('.').next() {
            allowed.contains(&ext.to_lowercase())
        } else {
            false
        }
    }
}

/// Security validator for NFT operations
pub struct NFTSecurityValidator {
    validator: NFTValidator,
    // Rate limiting
    max_transfers_per_block: u64,
    max_mints_per_block: u64,
}

impl Default for NFTSecurityValidator {
    fn default() -> Self {
        Self::new()
    }
}

impl NFTSecurityValidator {
    pub fn new() -> Self {
        Self {
            validator: NFTValidator::new(),
            max_transfers_per_block: 100,
            max_mints_per_block: 50,
        }
    }

    /// Validate batch transfer size
    pub fn validate_batch_size(&self, count: usize) -> Result<(), ValidationError> {
        if count as u64 > self.max_transfers_per_block {
            return Err(ValidationError::ContractError(format!(
                "Batch size {} exceeds limit {}",
                count, self.max_transfers_per_block
            )));
        }
        Ok(())
    }

    /// Validate mint batch size
    pub fn validate_mint_batch_size(&self, count: usize) -> Result<(), ValidationError> {
        if count as u64 > self.max_mints_per_block {
            return Err(ValidationError::ContractError(format!(
                "Mint batch size {} exceeds limit {}",
                count, self.max_mints_per_block
            )));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_metadata() {
        let validator = NFTValidator::new();
        
        let metadata = NFTMetadata {
            token_id: "1".to_string(),
            name: "Test NFT".to_string(),
            description: "A test NFT".to_string(),
            image: "https://example.com/image.png".to_string(),
            animation_url: None,
            external_url: None,
            attributes: vec![],
        };
        
        let result = validator.validate_metadata(&metadata);
        assert!(result.valid);
        assert!(result.errors.is_empty());
    }

    #[test]
    fn test_validate_contract_address() {
        let validator = NFTValidator::new();
        
        assert!(validator.validate_contract_address("0x1234567890123456789012345678901234567890").is_ok());
        assert!(validator.validate_contract_address("").is_err());
    }
}