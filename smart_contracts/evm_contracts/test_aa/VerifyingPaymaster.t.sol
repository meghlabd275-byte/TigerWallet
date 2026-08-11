// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import {Test} from "forge-std/Test.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

import {VerifyingPaymaster} from "../account_abstraction/VerifyingPaymaster.sol";
import {EntryPoint} from "../account_abstraction/EntryPoint.sol";
import {IEntryPoint} from "../account_abstraction/interfaces/IEntryPoint.sol";
import {PackedUserOperation} from "../account_abstraction/interfaces/PackedUserOperation.sol";
import {UserOperationLib} from "../account_abstraction/UserOperationLib.sol";

/// @title VerifyingPaymaster test
/// @notice Real ECDSA sponsorship validation. No mocks: a real sponsor key
///         signs the real EIP-191-prefixed userOpHash.
contract VerifyingPaymasterTest is Test {
    IEntryPoint entryPoint;
    VerifyingPaymaster paymaster;

    uint256 ownerPk = 0x1111111111111111111111111111111111111111111111111111111111111111;
    address owner;
    uint256 sponsorPk = 0x2222222222222222222222222222222222222222222222222222222222222222;
    address sponsor;
    address sender = address(0xBEEF);

    function setUp() public {
        // Advance the block timestamp so validAfter = now - 1min does not underflow.
        vm.warp(1_700_000_000);
        owner = vm.addr(ownerPk);
        sponsor = vm.addr(sponsorPk);
        entryPoint = IEntryPoint(payable(address(new EntryPoint())));
        paymaster = new VerifyingPaymaster(entryPoint, owner, sponsor);
    }

    /// @dev Build paymasterAndData = address(20) || vGas(16) || postGas(16) || sig(65) || validUntil(6) || validAfter(6)
    function _buildPaymasterAndData(bytes memory sig, uint48 validUntil, uint48 validAfter)
        internal
        view
        returns (bytes memory)
    {
        bytes memory head = abi.encodePacked(
            address(paymaster),
            uint128(0x10000), // verificationGasLimit
            uint128(0x10000)  // postOpGasLimit
        );
        bytes memory data = abi.encodePacked(sig, validUntil, validAfter);
        return abi.encodePacked(head, data);
    }

    function _emptyOp() internal pure returns (PackedUserOperation memory op) {
        op.accountGasLimits = bytes32(uint256(0));
        op.gasFees = bytes32(uint256(0));
    }

    function testValidSponsorSignatureAccepted() public {
        bytes32 userOpHash = keccak256("sponsor-this-op");
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sponsorPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);
        uint48 validUntil = uint48(block.timestamp + 1 hours);
        uint48 validAfter = uint48(block.timestamp - 1 minutes);

        PackedUserOperation memory op = _emptyOp();
        op.sender = sender;
        op.paymasterAndData = _buildPaymasterAndData(sig, validUntil, validAfter);

        vm.prank(address(entryPoint));
        (bytes memory context, uint256 validationData) =
            paymaster.validatePaymasterUserOp(op, userOpHash, 1e15);
        assertEq(validationData, 0, "valid sponsor sig should return 0");
        assertEq(context.length, 32, "context should encode sender address");
        address ctxSender = abi.decode(context, (address));
        assertEq(ctxSender, sender, "context sender mismatch");
    }

    function testWrongSignerRejected() public {
        bytes32 userOpHash = keccak256("sponsor-this-op");
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        // Sign with an unrelated key.
        uint256 attackerPk = 0x3333333333333333333333333333333333333333333333333333333333333333;
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(attackerPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);
        uint48 validUntil = uint48(block.timestamp + 1 hours);
        uint48 validAfter = uint48(block.timestamp - 1 minutes);

        PackedUserOperation memory op = _emptyOp();
        op.sender = sender;
        op.paymasterAndData = _buildPaymasterAndData(sig, validUntil, validAfter);

        vm.prank(address(entryPoint));
        (, uint256 validationData) = paymaster.validatePaymasterUserOp(op, userOpHash, 1e15);
        assertEq(validationData, 1, "wrong signer should return SIG_VALIDATION_FAILED");
    }

    function testBadPaymasterDataLengthRejected() public {
        PackedUserOperation memory op = _emptyOp();
        op.sender = sender;
        // Only the head, no sponsor data.
        op.paymasterAndData = abi.encodePacked(address(paymaster), uint128(1), uint128(1));

        vm.prank(address(entryPoint));
        (, uint256 validationData) = paymaster.validatePaymasterUserOp(op, keccak256("h"), 1e15);
        assertEq(validationData, 1, "bad data length should return SIG_VALIDATION_FAILED");
    }

    function testOutsideTimeRangeRejected() public {
        bytes32 userOpHash = keccak256("expired-op");
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sponsorPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);
        // validUntil is in the past.
        uint48 validUntil = uint48(block.timestamp - 1 hours);
        uint48 validAfter = uint48(block.timestamp - 2 hours);

        PackedUserOperation memory op = _emptyOp();
        op.sender = sender;
        op.paymasterAndData = _buildPaymasterAndData(sig, validUntil, validAfter);

        vm.prank(address(entryPoint));
        (, uint256 validationData) = paymaster.validatePaymasterUserOp(op, userOpHash, 1e15);
        assertEq(validationData, 1, "expired op should return SIG_VALIDATION_FAILED");
    }

    function testWhitelistFailClosed() public {
        // Owner enables the whitelist but does NOT add `sender`.
        vm.prank(owner);
        paymaster.enableWhitelist(true);

        bytes32 userOpHash = keccak256("sponsor-this-op");
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sponsorPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);
        uint48 validUntil = uint48(block.timestamp + 1 hours);
        uint48 validAfter = uint48(block.timestamp - 1 minutes);

        PackedUserOperation memory op = _emptyOp();
        op.sender = sender;
        op.paymasterAndData = _buildPaymasterAndData(sig, validUntil, validAfter);

        vm.prank(address(entryPoint));
        (, uint256 validationData) = paymaster.validatePaymasterUserOp(op, userOpHash, 1e15);
        assertEq(validationData, 1, "non-whitelisted sender should be rejected when whitelist on");
    }

    function testWhitelistedSenderAccepted() public {
        vm.prank(owner);
        paymaster.enableWhitelist(true);
        vm.prank(owner);
        paymaster.setSenderWhitelisted(sender, true);

        bytes32 userOpHash = keccak256("sponsor-this-op");
        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sponsorPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);
        uint48 validUntil = uint48(block.timestamp + 1 hours);
        uint48 validAfter = uint48(block.timestamp - 1 minutes);

        PackedUserOperation memory op = _emptyOp();
        op.sender = sender;
        op.paymasterAndData = _buildPaymasterAndData(sig, validUntil, validAfter);

        vm.prank(address(entryPoint));
        (, uint256 validationData) = paymaster.validatePaymasterUserOp(op, userOpHash, 1e15);
        assertEq(validationData, 0, "whitelisted sender with valid sig should return 0");
    }

    function testOwnerCanRotateSigner() public {
        address newSigner = address(0xCAFE);
        vm.prank(owner);
        paymaster.setSigningSigner(newSigner);
        assertEq(paymaster.signingSigner(), newSigner, "signer not rotated");
    }

    function testNonOwnerCannotRotateSigner() public {
        vm.expectRevert();
        vm.prank(address(0x9999));
        paymaster.setSigningSigner(address(0xCAFE));
    }
}
