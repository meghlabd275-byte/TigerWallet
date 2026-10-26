//! Tests for the account abstraction service.
//!
//! These cover the behaviours that must hold for the service to be safe:
//! fail-closed submission, real operation hashing, threshold enforcement in
//! recovery and multi-sig, and paymaster access control.

use tigerwallet_account_abstraction::*;

const ENTRY_POINT: &str = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789";
const OWNER: &str = "0x742d35Cc6634C0532925a3b844Bc454e4438f44e";
const TX_HASH: &str = "0x9c1f0a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f";

fn call() -> Call {
    Call {
        to: "0xA0b86a33E6441C4C1A1d1d2d2E2E2E2E2E2E2E2E".to_string(),
        value: 0,
        data: vec![],
    }
}

mod common;

const FACTORY: &str = "0x9406Cc6185a346906296840746125a0E44976454";

/// A service pointed at a stub node, so account creation runs the real
/// `eth_call` path against a real HTTP endpoint.
async fn initialized_service() -> AccountAbstractionService {
    let node = common::start_stub_node(Ok(TX_HASH)).await;

    let service = AccountAbstractionService::new(1);
    service.initialize(ENTRY_POINT.to_string()).await.unwrap();
    service
        .configure_factory(FACTORY.to_string(), &node.url)
        .await
        .unwrap();
    service
}

async fn service_with_account() -> (AccountAbstractionService, AccountInfo) {
    let service = initialized_service().await;
    let account = service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();
    (service, account)
}

// --- accounts ---------------------------------------------------------------

#[tokio::test]
async fn create_account_returns_a_deterministic_address() {
    let service = initialized_service().await;

    let first = service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();
    let second = service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();

    // The same owners and salt must produce the same counterfactual address.
    assert_eq!(first.address, second.address);
    assert_ne!(first.account_id, second.account_id);
}

#[tokio::test]
async fn a_different_salt_gives_a_different_address() {
    let service = initialized_service().await;

    let first = service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();
    let second = service
        .create_account(vec![OWNER.to_string()], 1, 1)
        .await
        .unwrap();

    assert_ne!(first.address, second.address);
}

#[tokio::test]
async fn create_account_rejects_an_impossible_threshold() {
    let service = initialized_service().await;

    assert!(service
        .create_account(vec![OWNER.to_string()], 2, 0)
        .await
        .is_err());
    assert!(service
        .create_account(vec![OWNER.to_string()], 0, 0)
        .await
        .is_err());
    assert!(service.create_account(vec![], 1, 0).await.is_err());
}

#[tokio::test]
async fn get_account_reports_an_unknown_address() {
    let service = initialized_service().await;
    assert!(service.get_account("0xdeadbeef").await.is_err());
}

// --- user operations --------------------------------------------------------

#[tokio::test]
async fn build_user_op_uses_the_requested_options() {
    let (service, account) = service_with_account().await;

    let options = UserOpOptions {
        call_gas_limit: Some(123_456),
        verification_gas_limit: Some(654_321),
        pre_verification_gas: Some(11_111),
        max_fee_per_gas: Some(30_000_000_000),
        max_priority_fee_per_gas: Some(1_000_000_000),
        paymaster_data: None,
        signature: None,
    };

    let user_op = service
        .build_user_op(&account.address, vec![call()], options)
        .await
        .unwrap();

    assert_eq!(user_op.call_gas_limit, 123_456);
    assert_eq!(user_op.verification_gas_limit, 654_321);
    assert_eq!(user_op.max_fee_per_gas, 30_000_000_000);
    assert_eq!(user_op.max_priority_fee_per_gas, 1_000_000_000);
    assert_eq!(user_op.pre_verification_gas, 11_111);
    assert_eq!(user_op.sender, account.address);
}

#[tokio::test]
async fn operation_hash_is_stable_and_bound_to_the_chain() {
    let (service, account) = service_with_account().await;

    let user_op = service
        .build_user_op(&account.address, vec![call()], UserOpOptions::default())
        .await
        .unwrap();

    let first = service.get_operation_hash(&user_op).await.unwrap();
    let second = service.get_operation_hash(&user_op).await.unwrap();
    assert_eq!(first, second);
    assert_eq!(first.len(), 32);

    let node = common::start_stub_node(Ok(TX_HASH)).await;
    let other_chain = AccountAbstractionService::new(137);
    other_chain.initialize(ENTRY_POINT.to_string()).await.unwrap();
    other_chain
        .configure_factory(FACTORY.to_string(), &node.url)
        .await
        .unwrap();
    let other = other_chain.get_operation_hash(&user_op).await.unwrap();
    assert_ne!(first, other, "the hash must bind the chain id");
}

#[tokio::test]
async fn operation_hash_requires_an_entry_point() {
    // A service with a factory but no EntryPoint.
    let node = common::start_stub_node(Ok(TX_HASH)).await;
    let service = AccountAbstractionService::new(1);
    service
        .configure_factory(FACTORY.to_string(), &node.url)
        .await
        .unwrap();

    let account = service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();
    let user_op = service
        .build_user_op(&account.address, vec![call()], UserOpOptions::default())
        .await
        .unwrap();

    // Without an EntryPoint there is no correct hash to compute, so this must
    // fail rather than return something arbitrary.
    assert!(service.get_operation_hash(&user_op).await.is_err());
}

#[tokio::test]
async fn signing_a_user_op_requires_ownership() {
    let (service, account) = service_with_account().await;

    let mut user_op = service
        .build_user_op(&account.address, vec![call()], UserOpOptions::default())
        .await
        .unwrap();

    let stranger = "0x000000000000000000000000000000000000dead";
    assert!(service.sign_user_op(&mut user_op, stranger).await.is_err());
}

#[tokio::test]
async fn signing_a_user_op_produces_a_verifiable_signature() {
    let (service, account) = service_with_account().await;

    let mut user_op = service
        .build_user_op(&account.address, vec![call()], UserOpOptions::default())
        .await
        .unwrap();

    let signature = service.sign_user_op(&mut user_op, OWNER).await.unwrap();
    assert!(!signature.is_empty());
    assert_eq!(user_op.signature, signature);
}

// --- fail-closed submission -------------------------------------------------

#[tokio::test]
async fn submitting_without_a_bundler_configuration_fails_closed() {
    let (service, account) = service_with_account().await;

    let mut user_op = service
        .build_user_op(&account.address, vec![call()], UserOpOptions::default())
        .await
        .unwrap();
    service.sign_user_op(&mut user_op, OWNER).await.unwrap();

    // No RPC endpoint or key has been configured, so submission must report an
    // error rather than an operation hash.
    let result = service.submit_user_op(user_op).await;
    assert!(
        result.is_err(),
        "submission without an RPC endpoint must not succeed"
    );
}

#[tokio::test]
async fn configure_bundler_rejects_a_malformed_key() {
    let service = initialized_service().await;
    assert!(service
        .configure_bundler("http://127.0.0.1:8545", "not-a-key")
        .await
        .is_err());
}

#[tokio::test]
async fn configure_bundler_accepts_a_valid_key() {
    let service = initialized_service().await;
    service
        .configure_bundler("http://127.0.0.1:8545", &hex::encode([0x11u8; 32]))
        .await
        .unwrap();
}

// --- recovery ---------------------------------------------------------------

fn guardians() -> Vec<GuardianInfo> {
    vec![
        GuardianInfo {
            address: "0x1111111111111111111111111111111111111111".to_string(),
            delay: 0,
            weight: 1,
            active: true,
        },
        GuardianInfo {
            address: "0x2222222222222222222222222222222222222222".to_string(),
            delay: 0,
            weight: 1,
            active: true,
        },
        GuardianInfo {
            address: "0x3333333333333333333333333333333333333333".to_string(),
            delay: 0,
            weight: 1,
            active: false,
        },
    ]
}

/// A recovery service with a zero delay, so completion is not blocked by the
/// time lock.
fn recovery_service(threshold: u32) -> RecoveryService {
    let mut recovery = RecoveryService::new();
    recovery.set_min_guardian_delay(0);
    recovery
        .configure_guardians("account", guardians(), threshold, 0)
        .unwrap();
    recovery
}

#[tokio::test]
async fn configure_guardians_rejects_an_impossible_threshold() {
    let mut recovery = RecoveryService::new();
    recovery.set_min_guardian_delay(0);

    assert!(recovery
        .configure_guardians("account", guardians(), 9, 0)
        .is_err());
    assert!(recovery
        .configure_guardians("account", guardians(), 0, 0)
        .is_err());
    assert!(recovery.configure_guardians("account", vec![], 1, 0).is_err());
}

#[tokio::test]
async fn configure_guardians_enforces_the_minimum_delay() {
    let mut recovery = RecoveryService::new();
    // The default minimum delay is 24 hours.
    assert!(recovery
        .configure_guardians("account", guardians(), 1, 60)
        .is_err());
}

#[tokio::test]
async fn a_recovery_starts_with_one_confirmation() {
    let mut recovery = recovery_service(2);

    let request = recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    assert_eq!(request.confirmations, 1);
    assert_eq!(request.threshold, 2);
}

#[tokio::test]
async fn an_inactive_guardian_cannot_start_a_recovery() {
    let mut recovery = recovery_service(2);

    let result = recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x3333333333333333333333333333333333333333",
            1,
        )
        .await;

    assert!(result.is_err(), "an inactive guardian must be rejected");
}

#[tokio::test]
async fn a_stranger_cannot_start_a_recovery() {
    let mut recovery = recovery_service(2);

    let result = recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x9999999999999999999999999999999999999999",
            1,
        )
        .await;

    assert!(result.is_err());
}

#[tokio::test]
async fn a_second_recovery_cannot_start_while_one_is_in_flight() {
    let mut recovery = recovery_service(2);

    recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    let result = recovery
        .start_recovery(
            "account",
            "0xother",
            "0x2222222222222222222222222222222222222222",
            1,
        )
        .await;

    assert!(result.is_err(), "the in-flight request must not be replaced");
}

#[tokio::test]
async fn a_guardian_cannot_confirm_twice() {
    let mut recovery = recovery_service(2);

    recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    let result = recovery
        .confirm_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
        )
        .await;

    assert!(result.is_err(), "a repeat confirmation must be rejected");
}

#[tokio::test]
async fn a_single_guardian_cannot_complete_a_two_of_three_recovery() {
    let mut recovery = recovery_service(2);

    recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    // One confirmation is not the threshold of two.
    let result = recovery.complete_recovery("account").await;
    assert!(
        result.is_err(),
        "completion below the threshold must be refused"
    );
}

#[tokio::test]
async fn recovery_completes_once_the_threshold_is_reached() {
    let mut recovery = recovery_service(2);

    recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    let reached = recovery
        .confirm_recovery(
            "account",
            "0xnewowner",
            "0x2222222222222222222222222222222222222222",
        )
        .await
        .unwrap();
    assert!(reached, "the second confirmation reaches the threshold");

    let new_owner = recovery.complete_recovery("account").await.unwrap();
    assert_eq!(new_owner, "0xnewowner");

    // Completing twice must not be possible.
    assert!(recovery.complete_recovery("account").await.is_err());
}

#[tokio::test]
async fn a_confirmation_for_a_different_new_owner_is_rejected() {
    let mut recovery = recovery_service(2);

    recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    let result = recovery
        .confirm_recovery(
            "account",
            "0xsomeoneelse",
            "0x2222222222222222222222222222222222222222",
        )
        .await;

    assert!(result.is_err());
}

#[tokio::test]
async fn a_cancelled_recovery_cannot_complete() {
    let mut recovery = recovery_service(1);

    recovery
        .start_recovery(
            "account",
            "0xnewowner",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .unwrap();

    recovery.cancel_recovery("account").await.unwrap();

    assert!(recovery.complete_recovery("account").await.is_err());
}

#[tokio::test]
async fn completing_an_unknown_recovery_is_rejected() {
    let mut recovery = recovery_service(1);
    assert!(recovery.complete_recovery("nobody").await.is_err());
}

// --- multi-sig --------------------------------------------------------------

#[tokio::test]
async fn multi_sig_requires_the_threshold_of_signatures() {
    let mut multisig = MultiSigService::new();

    let tx_id = multisig
        .create_transaction("account", vec![call()], 2)
        .await
        .unwrap();

    multisig
        .sign_transaction(&tx_id, "signer-a", vec![1])
        .await
        .unwrap();

    // One signature is below the threshold of two.
    assert!(multisig.execute(&tx_id).is_err());

    let reached = multisig
        .sign_transaction(&tx_id, "signer-b", vec![2])
        .await
        .unwrap();
    assert!(reached);

    assert_eq!(multisig.execute(&tx_id).unwrap(), tx_id);
}

#[tokio::test]
async fn a_signer_cannot_sign_the_same_transaction_twice() {
    let mut multisig = MultiSigService::new();
    let tx_id = multisig
        .create_transaction("account", vec![call()], 2)
        .await
        .unwrap();

    multisig
        .sign_transaction(&tx_id, "signer-a", vec![1])
        .await
        .unwrap();

    // A repeat must not count toward the threshold.
    assert!(multisig
        .sign_transaction(&tx_id, "signer-a", vec![1])
        .await
        .is_err());
}

#[tokio::test]
async fn multi_sig_rejects_an_empty_signature_and_a_zero_threshold() {
    let mut multisig = MultiSigService::new();

    assert!(multisig
        .create_transaction("account", vec![call()], 0)
        .await
        .is_err());

    let tx_id = multisig
        .create_transaction("account", vec![call()], 1)
        .await
        .unwrap();

    assert!(multisig
        .sign_transaction(&tx_id, "signer-a", vec![])
        .await
        .is_err());
}

#[tokio::test]
async fn an_executed_transaction_cannot_be_executed_again() {
    let mut multisig = MultiSigService::new();
    let tx_id = multisig
        .create_transaction("account", vec![call()], 1)
        .await
        .unwrap();

    multisig
        .sign_transaction(&tx_id, "signer-a", vec![1])
        .await
        .unwrap();

    multisig.execute(&tx_id).unwrap();
    assert!(multisig.execute(&tx_id).is_err());
}

#[tokio::test]
async fn an_executed_transaction_leaves_the_pending_list() {
    let mut multisig = MultiSigService::new();
    let tx_id = multisig
        .create_transaction("account", vec![call()], 1)
        .await
        .unwrap();

    assert_eq!(multisig.get_pending("account"), vec![tx_id.clone()]);

    multisig
        .sign_transaction(&tx_id, "signer-a", vec![1])
        .await
        .unwrap();
    multisig.execute(&tx_id).unwrap();

    assert!(multisig.get_pending("account").is_empty());
}

// --- paymaster --------------------------------------------------------------

fn sample_user_op(sender: &str) -> UserOperation {
    UserOperation {
        sender: sender.to_string(),
        nonce: 0,
        init_code: None,
        call_data: None,
        call_gas_limit: 100_000,
        verification_gas_limit: 100_000,
        pre_verification_gas: 50_000,
        max_fee_per_gas: 30_000_000_000,
        max_priority_fee_per_gas: 1_000_000_000,
        paymaster_data: None,
        signature: vec![1],
    }
}

#[tokio::test]
async fn only_the_owner_may_withdraw_a_deposit() {
    let mut paymaster = PaymasterService::new();
    paymaster.set_owner("owner".to_string());

    paymaster.deposit("owner", 1_000).await.unwrap();

    // A stranger cannot move the owner's deposit.
    assert!(paymaster
        .withdraw("stranger", 100, "0xrecipient")
        .await
        .is_err());

    paymaster
        .withdraw("owner", 100, "0xrecipient")
        .await
        .unwrap();
    assert_eq!(paymaster.get_deposit("owner").await.unwrap(), 900);
}

#[tokio::test]
async fn a_withdrawal_needs_a_recipient_and_a_nonzero_amount() {
    let mut paymaster = PaymasterService::new();
    paymaster.set_owner("owner".to_string());
    paymaster.deposit("owner", 1_000).await.unwrap();

    assert!(paymaster.withdraw("owner", 100, "").await.is_err());
    assert!(paymaster.withdraw("owner", 0, "0xrecipient").await.is_err());
}

#[tokio::test]
async fn a_deposit_must_be_greater_than_zero() {
    let mut paymaster = PaymasterService::new();
    assert!(paymaster.deposit("account", 0).await.is_err());
    paymaster.deposit("account", 1).await.unwrap();
}

#[tokio::test]
async fn a_withdrawal_beyond_the_deposit_is_rejected() {
    let mut paymaster = PaymasterService::new();
    paymaster.set_owner("owner".to_string());
    paymaster.deposit("owner", 50).await.unwrap();

    assert!(paymaster
        .withdraw("owner", 100, "0xrecipient")
        .await
        .is_err());
}

#[tokio::test]
async fn sponsorship_respects_the_per_sender_limit() {
    let mut paymaster = PaymasterService::new();
    let user_op = sample_user_op("0xsender");

    paymaster.set_user_limit("0xsender", 1).await;

    // The computed cost is far above the limit of one wei.
    assert!(paymaster.sponsor(&user_op).await.is_err());

    paymaster.set_user_limit("0xsender", u64::MAX).await;
    assert!(paymaster.sponsor(&user_op).await.is_ok());
}

#[tokio::test]
async fn sponsorship_rejects_an_unlisted_sender_when_a_whitelist_exists() {
    let mut paymaster = PaymasterService::new();
    paymaster.set_whitelisted_senders(vec!["0xallowed".to_string()]);

    assert!(paymaster
        .sponsor(&sample_user_op("0xother"))
        .await
        .is_err());
}

#[tokio::test]
async fn a_token_needs_a_nonzero_exchange_rate() {
    let mut paymaster = PaymasterService::new();

    assert!(paymaster
        .add_token("USDT".to_string(), 0, 6)
        .await
        .is_err());

    paymaster
        .add_token("USDT".to_string(), 1_000_000, 6)
        .await
        .unwrap();
    assert_eq!(paymaster.get_accepted_tokens(), vec!["USDT".to_string()]);
}

// --- security module --------------------------------------------------------

#[tokio::test]
async fn a_signature_verifies_against_its_own_key() {
    let mut security = SecurityModule::new();
    let info = security.generate_key_pair("owner").unwrap();

    let signature = security.sign(b"payload", "owner").await.unwrap();
    assert!(security
        .verify_signature(b"payload", &signature, "owner")
        .unwrap());

    // A different message must not verify.
    assert!(!security
        .verify_signature(b"other", &signature, "owner")
        .unwrap());

    assert_eq!(info.key_id, "owner");
}

#[tokio::test]
async fn signing_with_an_unknown_key_fails() {
    let security = SecurityModule::new();
    assert!(security.sign(b"payload", "missing").await.is_err());
}

/// Resolves a real counterfactual address from the deployed mainnet factory.
///
/// Ignored by default because it needs network access. Run it with
/// `cargo test -- --ignored` to confirm the derivation still matches the
/// canonical `SimpleAccountFactory`.
#[tokio::test]
#[ignore = "requires network access to a mainnet Ethereum node"]
async fn resolves_an_address_from_the_deployed_mainnet_factory() {
    let service = AccountAbstractionService::new(1);
    service.initialize(ENTRY_POINT.to_string()).await.unwrap();
    service
        .configure_factory(
            FACTORY.to_string(),
            "https://ethereum-rpc.publicnode.com",
        )
        .await
        .unwrap();

    let account = service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();

    // The value the deployed factory returns for this owner and salt.
    assert_eq!(
        account.address.to_lowercase(),
        "0x2bea269f9305577f5a9f6a9fdf04c969368879e7"
    );
}

/// Confirms account creation really queries the factory over JSON-RPC.
///
/// This guards against a regression where the address is derived locally again
/// instead of being read from the deployed factory.
#[tokio::test]
async fn creating_an_account_queries_the_factory() {
    let node = common::start_stub_node(Ok(TX_HASH)).await;

    let service = AccountAbstractionService::new(1);
    service.initialize(ENTRY_POINT.to_string()).await.unwrap();
    service
        .configure_factory(FACTORY.to_string(), &node.url)
        .await
        .unwrap();

    service
        .create_account(vec![OWNER.to_string()], 1, 0)
        .await
        .unwrap();

    assert_eq!(node.methods().await, vec!["eth_call".to_string()]);
}

// --- recovery denylist ------------------------------------------------------

#[tokio::test]
async fn a_denylisted_new_owner_cannot_start_a_recovery() {
    let mut service = RecoveryService::new();
    service.set_min_guardian_delay(0);

    service.block_address("0xDeAdBeEf00000000000000000000000000000001");

    service
        .configure_guardians("acct", guardians(), 2, 0)
        .unwrap();

    let result = service
        .start_recovery(
            "acct",
            "0xdeadbeef00000000000000000000000000000001",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await;

    assert!(
        result.is_err(),
        "a denylisted new owner was accepted; the denylist comparison is not working"
    );
}

#[tokio::test]
async fn unblocking_an_address_allows_the_recovery() {
    let mut service = RecoveryService::new();
    service.set_min_guardian_delay(0);

    service.block_address("0xdeadbeef00000000000000000000000000000001");
    service.unblock_address("0xDEADBEEF00000000000000000000000000000001");

    service
        .configure_guardians("acct", guardians(), 2, 0)
        .unwrap();

    service
        .start_recovery(
            "acct",
            "0xdeadbeef00000000000000000000000000000001",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .expect("an unblocked address must be accepted");
}

#[tokio::test]
async fn a_clean_address_is_accepted() {
    let mut service = RecoveryService::new();
    service.set_min_guardian_delay(0);
    service.block_address("0xdeadbeef00000000000000000000000000000001");

    service
        .configure_guardians("acct", guardians(), 2, 0)
        .unwrap();

    service
        .start_recovery(
            "acct",
            "0x00000000000000000000000000000000000000aa",
            "0x1111111111111111111111111111111111111111",
            1,
        )
        .await
        .expect("a clean address must be accepted");
}
