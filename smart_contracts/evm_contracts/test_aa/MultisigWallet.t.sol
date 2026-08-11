// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Test} from "forge-std/Test.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

import {MultisigWallet} from "../account_abstraction/tigerwallet/MultisigWallet.sol";

/// @title MultisigWallet test
/// @notice Real ECDSA threshold verification. No mocks: owners sign the real
///         EIP-712 transactionHash with `vm.sign`.
contract MultisigWalletTest is Test {
    MultisigWallet wallet;

    // Three real keys.
    uint256 pk1 = 0xA1;
    uint256 pk2 = 0xA2;
    uint256 pk3 = 0xA3;
    address o1;
    address o2;
    address o3;

    function setUp() public {
        o1 = vm.addr(pk1);
        o2 = vm.addr(pk2);
        o3 = vm.addr(pk3);
        // The contract requires owner signatures in ascending-address order, so
        // sort the three owner addresses before constructing the wallet.
        address[3] memory arr = [o1, o2, o3];
        if (arr[0] > arr[1]) (arr[0], arr[1]) = (arr[1], arr[0]);
        if (arr[1] > arr[2]) (arr[1], arr[2]) = (arr[2], arr[1]);
        if (arr[0] > arr[1]) (arr[0], arr[1]) = (arr[1], arr[0]);
        address[] memory owners = new address[](3);
        owners[0] = arr[0];
        owners[1] = arr[1];
        owners[2] = arr[2];
        wallet = new MultisigWallet(owners, 2);
    }

    // Map each owner address back to its private key so tests can sign in the
    // correct ascending order regardless of which addr(pk) is smallest.
    function _pkFor(address a) internal pure returns (uint256) {
        if (a == vm.addr(0xA1)) return 0xA1;
        if (a == vm.addr(0xA2)) return 0xA2;
        return 0xA3;
    }

    // Build a sorted, threshold-length signature blob for `txHash` using the
    // first `n` owners (which are already stored ascending by the constructor).
    function _sortedSigs(bytes32 txHash, uint256 n) internal view returns (bytes memory) {
        address[] memory owners = wallet.getOwners();
        bytes memory sigs = _sign(txHash, _pkFor(owners[0]));
        for (uint256 i = 1; i < n; i++) {
            sigs = _pack(sigs, _sign(txHash, _pkFor(owners[i])));
        }
        return sigs;
    }

    // Helper: sign `txHash` with `pk` and return a packed r||s||v (v=27/28).
    function _sign(bytes32 txHash, uint256 pk) internal pure returns (bytes memory) {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(pk, txHash);
        return abi.encodePacked(r, s, v);
    }

    // Helper: pack signatures in ascending-owner order.
    function _pack(bytes memory a, bytes memory b) internal pure returns (bytes memory) {
        return abi.encodePacked(a, b);
    }

    function test_Deposit() public {
        vm.deal(address(this), 5 ether);
        (bool ok,) = address(wallet).call{value: 5 ether}("");
        assertTrue(ok);
        assertEq(address(wallet).balance, 5 ether);
    }

    function test_ExecuteWithThresholdSignatures() public {
        // Fund the wallet.
        vm.deal(address(wallet), 2 ether);

        address recipient = address(0xCAFE);
        bytes32 txHash = wallet.transactionHash(recipient, 1 ether, "", 0);

        // Sign with the first `threshold` (2) owners, ascending.
        bytes memory sigs = _sortedSigs(txHash, 2);

        uint256 balBefore = recipient.balance;
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
        assertEq(recipient.balance - balBefore, 1 ether);
        assertEq(wallet.nonce(), 1);
    }

    function test_RevertWhenInsufficientSigs() public {
        vm.deal(address(wallet), 2 ether);
        address recipient = address(0xCAFE);
        bytes32 txHash = wallet.transactionHash(recipient, 1 ether, "", 0);
        // Only 1 sig (threshold is 2).
        bytes memory sigs = _sign(txHash, pk1);
        vm.expectRevert();
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
    }

    function test_RevertWhenNonOwnerSigns() public {
        vm.deal(address(wallet), 2 ether);
        address recipient = address(0xCAFE);
        bytes32 txHash = wallet.transactionHash(recipient, 1 ether, "", 0);
        uint256 attackerPk = 0xDEAD;
        // attacker (address will be low/high) + first owner — but attacker is
        // not an owner, so it must revert. We put the two sigs in ascending
        // order of recovered address to isolate the "not owner" failure.
        address attacker = vm.addr(attackerPk);
        bytes memory attackerSig = _sign(txHash, attackerPk);
        bytes memory ownerSig = _sign(txHash, _pkFor(wallet.getOwners()[0]));
        bytes memory sigs = attacker < wallet.getOwners()[0]
            ? _pack(attackerSig, ownerSig)
            : _pack(ownerSig, attackerSig);
        vm.expectRevert();
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
    }

    function test_RevertWhenWrongTxHash() public {
        vm.deal(address(wallet), 2 ether);
        address recipient = address(0xCAFE);
        // Sign a DIFFERENT tx (different value) with the real owners, sorted.
        bytes32 wrongHash = wallet.transactionHash(recipient, 2 ether, "", 0);
        bytes memory sigs = _sortedSigs(wrongHash, 2);
        vm.expectRevert();
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
    }

    function test_RevertWhenSigsUnsorted() public {
        vm.deal(address(wallet), 2 ether);
        address recipient = address(0xCAFE);
        bytes32 txHash = wallet.transactionHash(recipient, 1 ether, "", 0);
        // Sign with owner[1] then owner[0] — descending order must reject.
        address[] memory owners = wallet.getOwners();
        bytes memory sigs = _pack(
            _sign(txHash, _pkFor(owners[1])),
            _sign(txHash, _pkFor(owners[0]))
        );
        vm.expectRevert();
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
    }

    function test_RevertWhenDuplicateOwner() public {
        vm.deal(address(wallet), 2 ether);
        address recipient = address(0xCAFE);
        bytes32 txHash = wallet.transactionHash(recipient, 1 ether, "", 0);
        // Same owner twice — duplicate must fail the sorted/dup check.
        address[] memory owners = wallet.getOwners();
        bytes memory sigs = _pack(
            _sign(txHash, _pkFor(owners[0])),
            _sign(txHash, _pkFor(owners[0]))
        );
        vm.expectRevert();
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
    }

    function test_RevertWhenReplayNonce() public {
        vm.deal(address(wallet), 4 ether);
        address recipient = address(0xCAFE);
        bytes32 txHash = wallet.transactionHash(recipient, 1 ether, "", 0);
        bytes memory sigs = _sortedSigs(txHash, 2);
        wallet.executeTransaction(recipient, 1 ether, "", sigs);

        // Reusing the same nonce (1 now, but sigs are for nonce 0) must fail.
        vm.expectRevert();
        wallet.executeTransaction(recipient, 1 ether, "", sigs);
    }

    function test_ExecuteWithCalldata() public {
        // Deploy a target counter contract.
        Counter counter = new Counter();
        bytes memory data = abi.encodeWithSignature("increment()");
        bytes32 txHash = wallet.transactionHash(address(counter), 0, data, 0);
        bytes memory sigs = _sortedSigs(txHash, 2);
        wallet.executeTransaction(address(counter), 0, data, sigs);
        assertEq(counter.count(), 1);
    }

    function test_OwnerManagement() public {
        // Add a 4th owner via the wallet itself: craft an addOwner tx, sign,
        // execute. Use threshold 2 of the current 3 owners.
        address newOwner = address(0xBABE);
        bytes memory data = abi.encodeWithSignature("addOwner(address)", newOwner);
        bytes32 txHash = wallet.transactionHash(address(wallet), 0, data, 0);
        bytes memory sigs = _sortedSigs(txHash, 2);
        wallet.executeTransaction(address(wallet), 0, data, sigs);
        assertTrue(wallet.isOwner_(newOwner));
        assertEq(wallet.ownerCount(), 4);
    }

    function test_ChangeThreshold() public {
        bytes memory data = abi.encodeWithSignature("changeThreshold(uint256)", 3);
        bytes32 txHash = wallet.transactionHash(address(wallet), 0, data, 0);
        bytes memory sigs = _sortedSigs(txHash, 2);
        wallet.executeTransaction(address(wallet), 0, data, sigs);
        assertEq(wallet.threshold(), 3);
    }

    function test_RevertConstructorBadThreshold() public {
        address[] memory owners = new address[](2);
        owners[0] = o1;
        owners[1] = o2;
        vm.expectRevert();
        new MultisigWallet(owners, 3); // threshold > owners
    }

    function test_RevertConstructorDuplicateOwner() public {
        address[] memory owners = new address[](2);
        owners[0] = o1;
        owners[1] = o1;
        vm.expectRevert();
        new MultisigWallet(owners, 1);
    }
}

contract Counter {
    uint256 public count;

    function increment() external {
        count += 1;
    }
}
