//! TigerWallet multi-signature service.
//!
//! This binary exercises the library in `lib.rs`. It does not sign or broadcast
//! anything: the crate holds no RPC client, so `execute_transaction` records the
//! transaction hash that a real broadcaster supplies and refuses to invent one.

use std::collections::HashMap;
use std::process::ExitCode;

use tigerwallet_multisig::{
    MultiSigError, MultiSigWallet, ThresholdSigner, TransactionStatus,
};

fn main() -> ExitCode {
    env_logger::init();

    let owners = vec![
        "0x1111111111111111111111111111111111111111".to_string(),
        "0x2222222222222222222222222222222222222222".to_string(),
        "0x3333333333333333333333333333333333333333".to_string(),
    ];

    let wallet = match MultiSigWallet::new(owners.clone(), 2, 1) {
        Ok(wallet) => wallet,
        Err(error) => {
            eprintln!("failed to create the wallet: {error}");
            return ExitCode::FAILURE;
        }
    };

    let tx_id = match wallet.submit_transaction(
        owners[0].clone(),
        "0x4444444444444444444444444444444444444444".to_string(),
        "1000000000000000000".to_string(),
        vec![],
    ) {
        Ok(id) => id,
        Err(error) => {
            eprintln!("failed to submit the transaction: {error}");
            return ExitCode::FAILURE;
        }
    };
    println!("submitted transaction {tx_id}");

    // The first confirmation must not reach the threshold.
    match wallet.confirm_transaction(&tx_id, &owners[0]) {
        Ok(true) => {
            eprintln!("a single confirmation must not satisfy a 2-of-3 threshold");
            return ExitCode::FAILURE;
        }
        Ok(false) => println!("confirmation 1/2 recorded"),
        Err(error) => {
            eprintln!("failed to confirm: {error}");
            return ExitCode::FAILURE;
        }
    }

    match wallet.confirm_transaction(&tx_id, &owners[1]) {
        Ok(true) => println!("confirmation 2/2 recorded, threshold reached"),
        Ok(false) => {
            eprintln!("the second confirmation should have reached the threshold");
            return ExitCode::FAILURE;
        }
        Err(error) => {
            eprintln!("failed to confirm: {error}");
            return ExitCode::FAILURE;
        }
    }

    // Without a broadcast hash the execution is recorded but no hash is stored.
    match wallet.execute_transaction(&tx_id, "") {
        Ok(None) => println!("executed without a broadcast hash (none was supplied)"),
        Ok(Some(hash)) => {
            eprintln!("no hash should be recorded when none was supplied, got {hash}");
            return ExitCode::FAILURE;
        }
        Err(error) => {
            eprintln!("failed to execute: {error}");
            return ExitCode::FAILURE;
        }
    }

    match wallet.get_transaction(&tx_id) {
        Some(tx) => {
            if tx.status != TransactionStatus::Executed || tx.tx_hash.is_some() {
                eprintln!("unexpected stored state: {:?}", tx.status);
                return ExitCode::FAILURE;
            }
            println!("stored status: {:?}", tx.status);
        }
        None => {
            eprintln!("the transaction disappeared");
            return ExitCode::FAILURE;
        }
    }

    // A malformed broadcast hash is rejected.
    if !matches!(
        wallet.execute_transaction(&tx_id, "0xdeadbeef"),
        Err(MultiSigError::InvalidTransactionHash)
    ) {
        eprintln!("a malformed broadcast hash must be rejected");
        return ExitCode::FAILURE;
    }

    // Real Shamir secret sharing over GF(256).
    let mut signer = match ThresholdSigner::new(2, 3) {
        Ok(signer) => signer,
        Err(error) => {
            eprintln!("failed to build the threshold signer: {error}");
            return ExitCode::FAILURE;
        }
    };
    let secret = [0x5au8; 32];
    let shares = match signer.generate_shares(&secret) {
        Ok(shares) => shares,
        Err(error) => {
            eprintln!("failed to split the key: {error}");
            return ExitCode::FAILURE;
        }
    };
    println!(
        "split the key into {} shares, threshold {}",
        shares.len(),
        signer.threshold()
    );

    let mut subset = HashMap::new();
    subset.insert("signer_0".to_string(), shares["signer_0"].clone());
    subset.insert("signer_2".to_string(), shares["signer_2"].clone());
    match signer.combine_shares(&subset) {
        Ok(rebuilt) if rebuilt == secret => println!("rebuilt the key from 2 of 3 shares"),
        Ok(_) => {
            eprintln!("the rebuilt key does not match");
            return ExitCode::FAILURE;
        }
        Err(error) => {
            eprintln!("failed to rebuild the key: {error}");
            return ExitCode::FAILURE;
        }
    }

    ExitCode::SUCCESS
}
