// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Test} from "forge-std/Test.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

import {AccountFactory} from "../account_abstraction/AccountFactory.sol";
import {SimpleAccount} from "../account_abstraction/SimpleAccount.sol";
import {IEntryPoint} from "../account_abstraction/interfaces/IEntryPoint.sol";
import {EntryPoint} from "../account_abstraction/EntryPoint.sol";
import {PackedUserOperation} from "../account_abstraction/interfaces/PackedUserOperation.sol";

/// @title AccountFactory + SimpleAccount test
/// @notice Exercises real counterfactual deployment + real ECDSA validation.
///         No mocks: a real key signs the real EIP-191-prefixed userOpHash.
contract AccountFactoryTest is Test {
    AccountFactory factory;
    IEntryPoint entryPoint;

    uint256 ownerPk;
    address owner;

    function setUp() public {
        // Use a fresh random key for the owner (not a known Hardhat key).
        ownerPk = 0xa1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1;
        owner = vm.addr(ownerPk);

        // Deploy the canonical EntryPoint (no-arg constructor).
        entryPoint = IEntryPoint(payable(address(new EntryPoint())));
        factory = new AccountFactory(entryPoint);
    }

    function testCreateAccountAndAddressPrediction() public {
        uint256 salt = uint256(keccak256("tigerwallet-test-1"));
        address predicted = factory.getAddress(owner, salt);
        assertEq(predicted.code.length, 0, "predicted should be undeployed");

        address deployed = factory.createAccount(owner, salt);
        assertEq(deployed, predicted, "deployed != predicted");
        assertGt(deployed.code.length, 0, "account not deployed");

        SimpleAccount acct = SimpleAccount(payable(deployed));
        assertEq(acct.owner(), owner, "owner mismatch");
    }

    function testIdempotentCreate() public {
        uint256 salt = uint256(keccak256("tigerwallet-test-2"));
        address first = factory.createAccount(owner, salt);
        address second = factory.createAccount(owner, salt);
        assertEq(first, second, "re-create should return same address");
    }

    function testValidateUserOpValidSignature() public {
        uint256 salt = uint256(keccak256("tigerwallet-test-3"));
        address deployed = factory.createAccount(owner, salt);
        SimpleAccount acct = SimpleAccount(payable(deployed));

        bytes32 userOpHash = keccak256("some-user-op");
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(ownerPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);

        PackedUserOperation memory op = _emptyOp();
        op.sender = deployed;
        op.signature = sig;

        // Call validateUserOp as the entryPoint.
        vm.prank(address(entryPoint));
        uint256 vd = acct.validateUserOp(op, userOpHash, 0);
        assertEq(vd, 0, "valid sig should return 0");
    }

    function testValidateUserOpInvalidSignature() public {
        uint256 salt = uint256(keccak256("tigerwallet-test-4"));
        address deployed = factory.createAccount(owner, salt);
        SimpleAccount acct = SimpleAccount(payable(deployed));

        bytes32 userOpHash = keccak256("some-user-op");
        // Sign with a different key.
        uint256 attackerPk = 0xb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2;
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(attackerPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);

        PackedUserOperation memory op = _emptyOp();
        op.sender = deployed;
        op.signature = sig;

        vm.prank(address(entryPoint));
        uint256 vd = acct.validateUserOp(op, userOpHash, 0);
        assertEq(vd, 1, "invalid sig should return SIG_VALIDATION_FAILED (1)");
    }

    function testValidateUserOpWrongLengthSignature() public {
        uint256 salt = uint256(keccak256("tigerwallet-test-5"));
        address deployed = factory.createAccount(owner, salt);
        SimpleAccount acct = SimpleAccount(payable(deployed));

        PackedUserOperation memory op = _emptyOp();
        op.sender = deployed;
        op.signature = bytes("not-65-bytes");

        vm.prank(address(entryPoint));
        uint256 vd = acct.validateUserOp(op, keccak256("h"), 0);
        assertEq(vd, 1, "non-65-byte sig should return SIG_VALIDATION_FAILED (1)");
    }

    function _emptyOp() internal pure returns (PackedUserOperation memory op) {
        op.accountGasLimits = bytes32(uint256(0));
        op.gasFees = bytes32(uint256(0));
    }
}
