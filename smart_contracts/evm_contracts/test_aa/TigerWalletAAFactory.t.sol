// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "forge-std/Test.sol";
import "../account_abstraction/EntryPoint.sol";
import "../account_abstraction/tigerwallet/TigerWalletAccountFactory.sol";
import "../account_abstraction/tigerwallet/TigerWalletAccount.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

contract TigerWalletAAFactoryTest is Test {
    using ECDSA for bytes32;

    EntryPoint entryPoint;
    TigerWalletAccountFactory factory;
    uint256 ownerPk;
    address owner;

    function setUp() public {
        entryPoint = new EntryPoint();
        factory = new TigerWalletAccountFactory(IEntryPoint(address(entryPoint)));

        ownerPk = 0xA11CE;
        owner = vm.addr(ownerPk);
    }

    function testCounterfactualAddressMatchesDeployment() public {
        address predicted = factory.getAccountAddress(owner, 1);
        assertTrue(predicted.code.length == 0, "should be undeployed");

        TigerWalletAccount account = factory.createAccount(owner, 1);
        assertEq(address(account), predicted, "deployed address must match counterfactual");
        assertGt(address(account).code.length, 0, "account must have code");
    }

    function testCreateIdempotent() public {
        TigerWalletAccount a1 = factory.createAccount(owner, 7);
        TigerWalletAccount a2 = factory.createAccount(owner, 7);
        assertEq(address(a1), address(a2), "same key must return same account");
    }

    function testAccountOwnerAndEntryPoint() public {
        TigerWalletAccount account = factory.createAccount(owner, 2);
        assertEq(account.owner(), owner, "owner mismatch");
        assertEq(address(account.entryPoint()), address(entryPoint), "entrypoint mismatch");
    }

    function testSignatureValidationAcceptsOwner() public {
        TigerWalletAccount account = factory.createAccount(owner, 3);
        bytes32 userOpHash = keccak256("test-userop");
        bytes32 ethHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", userOpHash));
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(ownerPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);

        // Call validateUserOp as the entrypoint
        PackedUserOperation memory op = _emptyOp(address(account), sig);
        vm.prank(address(entryPoint));
        uint256 vd = account.validateUserOp(op, ethHash, 0);
        assertEq(vd, 0, "owner sig must validate");
    }

    function testSignatureValidationRejectsNonOwner() public {
        TigerWalletAccount account = factory.createAccount(owner, 4);
        uint256 attackerPk = 0xB0B;
        bytes32 userOpHash = keccak256("test-userop");
        bytes32 ethHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", userOpHash));
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(attackerPk, ethHash);
        bytes memory sig = abi.encodePacked(r, s, v);

        PackedUserOperation memory op = _emptyOp(address(account), sig);
        vm.prank(address(entryPoint));
        uint256 vd = account.validateUserOp(op, ethHash, 0);
        assertEq(vd, 1, "non-owner sig must fail");
    }

    function _emptyOp(address sender, bytes memory sig)
        internal
        pure
        returns (PackedUserOperation memory)
    {
        return PackedUserOperation({
            sender: sender,
            nonce: 0,
            initCode: "",
            callData: "",
            accountGasLimits: bytes32(0),
            preVerificationGas: 0,
            gasFees: bytes32(0),
            paymasterAndData: "",
            signature: sig
        });
    }
}
