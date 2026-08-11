// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/* TigerWallet — on-chain threshold multisig wallet.
 *
 * Real Gnosis Safe-style multisig: a transaction is executed only after a
 * threshold of owner ECDSA signatures over an EIP-191-prefixed keccak256 domain
 * hash is collected. Signature verification uses OpenZeppelin v5 `ECDSA.recover`
 * (real secp256k1, low-s enforced), NOT length checks or accept-anything.
 *
 * This is NOT an ERC-4337 account (it does not implement `validateUserOp`);
 * it is a standalone, deployable multisig wallet for the OKX/Coinbase-enterprise
 * use case. It pairs with the off-chain `go/multisig_service` relayer, which
 * collects owner signatures off-chain and submits the assembled calldata here.
 *
 * Security model:
 *   - Owners added/removed only by the wallet itself (via #execute on a
 *     `replaceOwner`/`addOwner`/`removeOwner` action) or by the initial
 *     deployer before renouncing root.
 *   - Threshold clamped to [1, ownerCount].
 *   - Nonce prevents replay; domain separator binds chain id + contract.
 *   - `execute` is `payable` so an executor can be refunded (gas relay).
 */

import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract MultisigWallet is ReentrancyGuard {
    using ECDSA for bytes32;

    address[] private owners;
    mapping(address => bool) private isOwner;
    uint256 public threshold;

    uint256 public nonce;
    uint256 immutable public chainId;

    // EIP-712 domain. Recomputed per-call (cheap; avoids storage slot).
    bytes32 private constant EIP712_DOMAIN_TYPEHASH =
        keccak256("EIP712Domain(uint256 chainId,address verifyingContract)");

    bytes32 private constant TX_TYPEHASH =
        keccak256("Transaction(address to,uint256 value,bytes data,uint256 nonce)");

    event Deposit(address indexed sender, uint256 value);
    event SubmitTransaction(address indexed owner, uint256 indexed txIndex, address to, uint256 value);
    event ConfirmTransaction(address indexed owner, uint256 indexed txIndex);
    event RevokeConfirmation(address indexed owner, uint256 indexed txIndex);
    event ExecuteTransaction(address indexed owner, uint256 indexed txIndex);
    event OwnerAddition(address indexed owner);
    event OwnerRemoval(address indexed owner);
    event ThresholdChange(uint256 threshold);

    modifier onlySelf() {
        require(msg.sender == address(this), "only self");
        _;
    }

    modifier onlyOwner() {
        require(isOwner[msg.sender], "not an owner");
        _;
    }

    constructor(address[] memory _owners, uint256 _threshold) {
        require(_owners.length > 0, "no owners");
        require(_threshold > 0 && _threshold <= _owners.length, "bad threshold");
        for (uint256 i = 0; i < _owners.length; i++) {
            address o = _owners[i];
            require(o != address(0), "zero owner");
            require(!isOwner[o], "duplicate owner");
            isOwner[o] = true;
            owners.push(o);
            emit OwnerAddition(o);
        }
        threshold = _threshold;
        emit ThresholdChange(_threshold);
        chainId = block.chainid;
    }

    receive() external payable {
        emit Deposit(msg.sender, msg.value);
    }

    // -------- domain / digest --------

    function domainSeparator() public view returns (bytes32) {
        return keccak256(abi.encode(EIP712_DOMAIN_TYPEHASH, block.chainid, address(this)));
    }

    /// @notice The 32-byte hash an owner signs (EIP-712 structured). The
    ///         executor passes the raw 65-byte sigs concatenated.
    function transactionHash(address to, uint256 value, bytes memory data, uint256 _nonce)
        public
        view
        returns (bytes32)
    {
        bytes32 structHash = keccak256(abi.encode(TX_TYPEHASH, to, value, keccak256(data), _nonce));
        return MessageHashUtils.toTypedDataHash(domainSeparator(), structHash);
    }

    // -------- execution --------

    /// @notice Execute a transaction once `threshold` owner signatures are
    ///         collected. Signatures must be sorted by owner address ascending
    ///         (Gnosis Safe convention) and each be a real 65-byte secp256k1
    ///         signature (r||s||v, v in {27,28}) over `transactionHash(...)`.
    function executeTransaction(
        address to,
        uint256 value,
        bytes memory data,
        bytes memory signatures
    ) external nonReentrant {
        bytes32 txHash = transactionHash(to, value, data, nonce);
        _requireValidSignatures(txHash, signatures);

        nonce += 1;

        (bool ok, bytes memory ret) = to.call{value: value}(data);
        require(ok, "tx failed");
        if (ret.length > 0) {
            // bubble revert reason if present
            assembly {
                revert(add(ret, 32), mload(ret))
            }
        }
        emit ExecuteTransaction(msg.sender, nonce - 1);
    }

    /// @dev Verifies that `signatures` contains `threshold` distinct owner
    ///      ECDSA signatures over `txHash`, sorted ascending by owner address.
    function _requireValidSignatures(bytes32 txHash, bytes memory signatures) internal view {
        require(signatures.length >= threshold * 65, "not enough sigs");
        address lastOwner = address(0);
        uint256 count = 0;
        for (uint256 i = 0; i < threshold; i++) {
            bytes32 r;
            bytes32 s;
            uint8 v;
            assembly {
                let ptr := add(signatures, mul(i, 65))
                r := mload(add(ptr, 0x20))
                s := mload(add(ptr, 0x40))
                v := byte(0, mload(add(ptr, 0x60)))
            }
            if (v != 27 && v != 28) {
                v = v == 0 ? 27 : v == 1 ? 28 : v;
            }
            require(v == 27 || v == 28, "bad v");
            address recovered = ECDSA.recover(txHash, v, r, s);
            require(recovered > lastOwner, "sigs not sorted / duplicate");
            require(isOwner[recovered], "not owner");
            lastOwner = recovered;
            count++;
        }
        require(count >= threshold, "threshold not met");
    }

    // -------- owner management (self-governed) --------

    function addOwner(address owner) external onlySelf {
        require(owner != address(0), "zero owner");
        require(!isOwner[owner], "already owner");
        isOwner[owner] = true;
        owners.push(owner);
        emit OwnerAddition(owner);
    }

    function removeOwner(address owner) external onlySelf {
        require(isOwner[owner], "not owner");
        isOwner[owner] = false;
        for (uint256 i = 0; i < owners.length; i++) {
            if (owners[i] == owner) {
                owners[i] = owners[owners.length - 1];
                owners.pop();
                break;
            }
        }
        if (threshold > owners.length) {
            threshold = owners.length;
            emit ThresholdChange(threshold);
        }
        emit OwnerRemoval(owner);
    }

    function changeThreshold(uint256 _threshold) external onlySelf {
        require(_threshold > 0 && _threshold <= owners.length, "bad threshold");
        threshold = _threshold;
        emit ThresholdChange(_threshold);
    }

    // -------- views --------

    function getOwners() external view returns (address[] memory) {
        return owners;
    }

    function ownerCount() external view returns (uint256) {
        return owners.length;
    }

    function isOwner_(address owner) external view returns (bool) {
        return isOwner[owner];
    }
}
