//! TigerWallet Security Tool
//! 
//! Command-line utility for security operations

use clap::{Parser, Subcommand};
use tigerwallet_security::{
    Encryption, KeyDerivation, Signer, HmacService, util, SecurityError,
};

#[derive(Parser)]
#[command(name = "security-tool")]
#[command(about = "TigerWallet Security Operations Tool", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Generate a new encryption key
    GenerateKey,
    
    /// Encrypt data
    Encrypt {
        /// Plaintext to encrypt
        plaintext: String,
        
        /// Encryption key (hex or base64)
        key: String,
    },
    
    /// Decrypt data
    Decrypt {
        /// Ciphertext to decrypt (base64)
        ciphertext: String,
        
        /// Decryption key
        key: String,
    },
    
    /// Hash a password
    HashPassword {
        /// Password to hash
        password: String,
    },
    
    /// Verify a password
    VerifyPassword {
        /// Password to verify
        password: String,
        
        /// Password hash to verify against
        hash: String,
    },
    
    /// Generate Ed25519 keypair
    GenerateKeypair,
    
    /// Sign a message
    Sign {
        /// Message to sign (hex)
        message: String,
        
        /// Private key (hex)
        private_key: String,
    },
    
    /// Verify a signature
    Verify {
        /// Message that was signed (hex)
        message: String,
        
        /// Signature to verify (hex)
        signature: String,
        
        /// Public key (hex)
        public_key: String,
    },
    
    /// Compute HMAC
    ComputeHmac {
        /// Key (hex)
        key: String,
        
        /// Message
        message: String,
    },
}

fn main() -> Result<(), SecurityError> {
    let cli = Cli::parse();

    match cli.command {
        Commands::GenerateKey => {
            let key = KeyDerivation::generate_key();
            println!("Generated key (hex): {}", hex::encode(key));
        }
        
        Commands::Encrypt { plaintext, key } => {
            let key_bytes = hex::decode(&key).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let key_array: [u8; 32] = key_bytes.try_into()
                .map_err(|_| SecurityError::InvalidKeyError("Key must be 32 bytes".into()))?;
            
            let ciphertext = Encryption::encrypt(plaintext.as_bytes(), &key_array)?;
            println!("Ciphertext (base64): {}", util::encode_base64(&ciphertext));
        }
        
        Commands::Decrypt { ciphertext, key } => {
            let key_bytes = hex::decode(&key).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let key_array: [u8; 32] = key_bytes.try_into()
                .map_err(|_| SecurityError::InvalidKeyError("Key must be 32 bytes".into()))?;
            
            let ciphertext_bytes = util::decode_base64(&ciphertext)?;
            let plaintext = Encryption::decrypt(&ciphertext_bytes, &key_array)?;
            println!("Plaintext: {}", String::from_utf8_lossy(&plaintext));
        }
        
        Commands::HashPassword { password } => {
            let hash = KeyDerivation::derive_key(&password, None)?;
            println!("Password hash: {}", hash);
        }
        
        Commands::VerifyPassword { password, hash } => {
            let valid = KeyDerivation::verify_password(&password, &hash)?;
            println!("Password valid: {}", valid);
        }
        
        Commands::GenerateKeypair => {
            let (signing_key, verifying_key) = Signer::generate_keypair();
            println!("Private key (hex): {}", hex::encode(signing_key.to_bytes()));
            println!("Public key (hex): {}", hex::encode(verifying_key.to_bytes()));
        }
        
        Commands::Sign { message, private_key } => {
            let msg_bytes = hex::decode(&message).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let key_bytes = hex::decode(&private_key).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let key_array: [u8; 32] = key_bytes.try_into()
                .map_err(|_| SecurityError::InvalidKeyError("Key must be 32 bytes".into()))?;
            
            let signing_key = ed25519_dalek::SigningKey::from_bytes(&key_array);
            let signature = Signer::sign(&msg_bytes, &signing_key);
            println!("Signature (hex): {}", hex::encode(signature.to_bytes()));
        }
        
        Commands::Verify { message, signature, public_key } => {
            let msg_bytes = hex::decode(&message).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let sig_bytes = hex::decode(&signature).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let pub_bytes = hex::decode(&public_key).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            
            let signature = ed25519_dalek::Signature::from_slice(&sig_bytes)
                .map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let verifying_key = ed25519_dalek::VerifyingKey::from_slice(&pub_bytes)
                .map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            
            let valid = Signer::verify(&msg_bytes, &signature, &verifying_key);
            println!("Signature valid: {}", valid);
        }
        
        Commands::ComputeHmac { key, message } => {
            let key_bytes = hex::decode(&key).map_err(|e| SecurityError::InvalidKeyError(e.to_string()))?;
            let msg_bytes = message.as_bytes();
            
            let hmac = HmacService::compute(&key_bytes, msg_bytes);
            println!("HMAC (hex): {}", hex::encode(hmac));
        }
    }

    Ok(())
}
