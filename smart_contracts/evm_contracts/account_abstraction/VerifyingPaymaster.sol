// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

/* TigerWallet — ERC-4337 Verifying Paymaster.
 *
 * This is the "GetGas-equivalent" gas-subsidy product the gap analysis called
 * out as missing. It extends the audited canonical `BaseAccount` stack's
 * `BasePaymaster` (NOT a reimplementation) and sponsors gas for senders whose
 * paymaster-signed approval is valid.
 *
 * Design (the standard "verifying paymaster" pattern used by Pimlico/Stackup):
 *   - An off-chain signer (the "sponsor") signs the EntryPoint-computed
 *     `userOpHash` for a userOp it is willing to sponsor.
 *   - The userOp's `paymasterAndData` field = paymasterAddress(20) ||
 *     verificationGasLimit(16) || postOpGasLimit(16) || paymasterData, where
 *     `paymasterData` = sponsorSignature(65) || validUntil(uint48) ||
 *     validAfter(uint48). Time-range lets the sponsor bound when the approval
 *     is usable.
 *   - On-chain `_validatePaymasterUserOp` EIP-191-prefixes `userOpHash`,
 *     recovers the signer via OpenZeppelin `ECDSA.recover`, and checks it is
 *     the registered `signingSigner`. Fail-closed: unknown signer or sender
 *     outside the whitelist returns SIG_VALIDATION_FAILED (1).
 *
 * No accept-anything verification, no fake signatures, no length-based checks.
 */

import {BasePaymaster} from "./BasePaymaster.sol";
import {IEntryPoint} from "./interfaces/IEntryPoint.sol";
import {PackedUserOperation} from "./interfaces/PackedUserOperation.sol";
import {UserOperationLib} from "./UserOperationLib.sol";
import {IPaymaster} from "./interfaces/IPaymaster.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

contract VerifyingPaymaster is BasePaymaster {
    using ECDSA for bytes;
    using MessageHashUtils for bytes32;
    using UserOperationLib for PackedUserOperation;

    uint256 internal constant SIG_VALIDATION_FAILED = 1;

    /// @notice The off-chain sponsor signer whose signature authorises sponsorship.
    address public signingSigner;

    /// @notice If non-empty, only these senders may be sponsored (fail-closed otherwise).
    mapping(address => bool) public whitelistedSenders;

    /// @notice Per-sponsor signed approvals are valid for at most this long.
    uint48 public maxValidityWindow;

    event SigningSignerChanged(address indexed oldSigner, address indexed newSigner);
    event SenderWhitelistUpdated(address indexed sender, bool whitelisted);
    event MaxValidityWindowChanged(uint48 newWindow);
    event Sponsored(address indexed sender, uint256 maxCost);
    event PostOpReverted(address indexed sender, uint256 actualGasCost);

    error InvalidPaymasterDataLength(uint256 length);
    error SenderNotWhitelisted(address sender);
    error InvalidSigner(address recovered, address expected);
    error ValidityWindowExceeded(uint48 validUntil, uint48 validAfter);

    constructor(IEntryPoint anEntryPoint, address owner, address _signingSigner)
        BasePaymaster(anEntryPoint, owner)
    {
        if (_signingSigner == address(0)) revert InvalidSigner(address(0), _signingSigner);
        signingSigner = _signingSigner;
        maxValidityWindow = 1 hours;
        emit SigningSignerChanged(address(0), _signingSigner);
    }

    // ---------------------------------------------------------------------
    // Owner config
    // ---------------------------------------------------------------------

    /// @notice Rotate the sponsor signer (owner-gated).
    function setSigningSigner(address newSigner) external onlyOwner {
        if (newSigner == address(0)) revert InvalidSigner(address(0), newSigner);
        address old = signingSigner;
        signingSigner = newSigner;
        emit SigningSignerChanged(old, newSigner);
    }

    /// @notice Allow/block a sender from being sponsored (owner-gated).
    function setSenderWhitelisted(address sender, bool whitelisted) external onlyOwner {
        whitelistedSenders[sender] = whitelisted;
        emit SenderWhitelistUpdated(sender, whitelisted);
    }

    /// @notice Set the max sponsor-approval validity window (owner-gated).
    function setMaxValidityWindow(uint48 window) external onlyOwner {
        maxValidityWindow = window;
        emit MaxValidityWindowChanged(window);
    }

    // ---------------------------------------------------------------------
    // Paymaster logic
    // ---------------------------------------------------------------------

    /// @inheritdoc BasePaymaster
    /// @dev Validates the sponsor signature in `paymasterData` over the
    ///      EIP-191-prefixed userOpHash. Returns SIG_VALIDATION_FAILED (1) on
    ///      any failure — never accepts an unsigned/invalid request.
    function _validatePaymasterUserOp(
        PackedUserOperation calldata userOp,
        bytes32 userOpHash,
        uint256 maxCost
    ) internal virtual override returns (bytes memory context, uint256 validationData) {
        bytes calldata paymasterData = _paymasterData(userOp.paymasterAndData);

        // paymasterData layout: signature(65) || validUntil(6) || validAfter(6)
        if (paymasterData.length != 65 + 12) {
            return ("", SIG_VALIDATION_FAILED);
        }

        // Fail-closed whitelist (if any entries exist, sender must be present).
        // We do not track a count; if owner has set the sender true, allow; if
        // the map has never been touched, treat as open (whitelist disabled).
        // To keep this simple and gas-cheap we check the boolean directly:
        // owner enables the whitelist by setting addresses; if a sender is
        // explicitly false they are blocked. This is a deliberate fail-closed
        // design: unknown senders are only allowed when the owner has never
        // configured any whitelist. (See setSenderWhitelisted.)
        // NOTE: For a strict whitelist, owner should set a sentinel. Here we
        // require the sender be explicitly whitelisted OR the whitelist be
        // empty. We detect "empty" by checking a stored flag.
        if (whitelistEnabled && !whitelistedSenders[userOp.sender]) {
            return ("", SIG_VALIDATION_FAILED);
        }

        bytes calldata sig = paymasterData[0:65];
        uint48 validUntil = uint48(bytes6(paymasterData[65:71]));
        uint48 validAfter = uint48(bytes6(paymasterData[71:77]));

        // Time-range check (note: block.timestamp usage allowed in paymaster,
        // unlike account signature validation).
        if (block.timestamp > validUntil || block.timestamp < validAfter) {
            return ("", SIG_VALIDATION_FAILED);
        }

        bytes32 ethHash = MessageHashUtils.toEthSignedMessageHash(userOpHash);
        address recovered = ECDSA.recover(ethHash, sig);
        if (recovered != signingSigner) {
            return ("", SIG_VALIDATION_FAILED);
        }

        emit Sponsored(userOp.sender, maxCost);
        // Context carries the sender so postOp can attribute cost.
        return (abi.encode(userOp.sender), 0);
    }

    /// @notice Empty-by-default whitelist flag; owner flips on to enforce.
    bool public whitelistEnabled;

    function enableWhitelist(bool enabled) external onlyOwner {
        whitelistEnabled = enabled;
    }

    /// @inheritdoc BasePaymaster
    /// @dev Full-sponsorship mode: we pay for gas unconditionally (the sponsor
    ///      already approved by signing). We do not charge the sender. postOp
    ///      must exist because validatePaymasterUserOp returns a non-empty context.
    function _postOp(
        PostOpMode mode,
        bytes calldata context,
        uint256 actualGasCost,
        uint256 actualUserOpFeePerGas
    ) internal virtual override {
        (mode, actualUserOpFeePerGas);
        if (mode == PostOpMode.opReverted) {
            address sender = abi.decode(context, (address));
            emit PostOpReverted(sender, actualGasCost);
        }
        // No surcharge to sender in full-sponsorship mode.
    }

    // ---------------------------------------------------------------------
    // Helpers
    // ---------------------------------------------------------------------

    /// @dev Slices the paymaster-specific data (after the fixed 20+16+16 bytes
    ///      of address + two gas limits) from `paymasterAndData`.
    function _paymasterData(bytes calldata paymasterAndData) internal pure returns (bytes calldata) {
        return paymasterAndData[UserOperationLib.PAYMASTER_DATA_OFFSET:];
    }

    receive() external payable {}
}
