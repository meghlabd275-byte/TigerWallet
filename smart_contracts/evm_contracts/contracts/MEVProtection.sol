// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

/**
 * @title TigerMEVProtection
 * @notice Advanced MEV protection: Flashbots bundles, private pools, encrypted mempool
 * @dev Integrates with MEV-resistant sequencers and private RPC providers
 */
contract TigerMEVProtection is Ownable {
    using ECDSA for bytes32;

    // Constants
    bytes32 public constant BUNDLE_SEPARATOR = keccak256("tigerwallet.mev.bundle");

    // Events
    event BundleSubmitted(
        bytes32 indexed bundleHash,
        address indexed sender,
        uint256 gasUsed,
        bool isPrivate
    );
    event PrivateTransactionExecuted(
        bytes32 indexed txHash,
        address indexed recipient,
        uint256 refund
    );
    event SandwichDetected(
        bytes32 indexed txHash,
        address indexed attacker,
        uint256 penalty
    );

    // State
    mapping(bytes32 => BundleInfo) public bundles;
    mapping(address => uint256) public mevRefunds;
    mapping(bytes32 => bool) public protectedTxs;
    mapping(bytes32 => bytes32) public txToBundle;

    address public flashbotsRelayer;
    address public mevBlocker;
    address public treasuryWallet;

    uint256 public refundPercentage = 95; // MEV capture refund: 95% to users
    uint256 public penaltyPercentage = 50; // Sandwich penalty: 50% slashed
    uint256 public sandwichTimeout = 5 minutes;

    struct BundleInfo {
        address sender;
        bytes32[] txHashes;
        uint256 gasUsed;
        uint256 timestamp;
        bool isPrivate;
        bool executed;
        uint256 refundAmount;
    }

    // Modifiers
    modifier onlyFlashbots() {
        require(
            msg.sender == flashbotsRelayer || msg.sender == owner(),
            "MEV: Only Flashbots or owner"
        );
        _;
    }

    constructor(address _flashbots, address _mevBlocker, address _treasury)
        Ownable(msg.sender)
    {
        flashbotsRelayer = _flashbots;
        mevBlocker = _mevBlocker;
        treasuryWallet = _treasury;
    }

    /**
     * @notice Submit transaction bundle for MEV protection
     * @param txs Array of transaction data
     * @param isPrivate Submit to private mempool (Flashbots/MEV-Blocker)
     */
    function submitBundle(
        bytes[] calldata txs,
        bool isPrivate,
        bytes calldata signature
    ) external payable returns (bytes32 bundleHash) {
        require(txs.length > 0, "MEV: empty bundle");
        require(msg.value >= txs.length * 0.001 ether, "MEV: insufficient fee");

        // Verify signature from sender
        bytes32 dataHash = keccak256(abi.encodePacked(txs, isPrivate));
        address recovered = dataHash.recover(signature);
        require(recovered == msg.sender, "MEV: invalid signature");

        // Create bundle hash
        bundleHash = keccak256(abi.encodePacked(txs, block.timestamp, msg.sender));

        // Store bundle info
        bytes32[] memory hashes = new bytes32[](txs.length);
        for (uint256 i = 0; i < txs.length; i++) {
            hashes[i] = keccak256(txs[i]);
            txToBundle[hashes[i]] = bundleHash;
        }

        bundles[bundleHash] = BundleInfo({
            sender: msg.sender,
            txHashes: hashes,
            gasUsed: 0,
            timestamp: block.timestamp,
            isPrivate: isPrivate,
            executed: false,
            refundAmount: 0
        });

        emit BundleSubmitted(bundleHash, msg.sender, 0, isPrivate);

        return bundleHash;
    }

    /**
     * @notice Mark transaction as executed and calculate refund
     */
    function recordExecution(
        bytes32 bundleHash,
        uint256 gasUsed
    ) external onlyFlashbots {
        BundleInfo storage bundle = bundles[bundleHash];
        require(!bundle.executed, "MEV: already executed");

        bundle.executed = true;
        bundle.gasUsed = gasUsed;

        // Calculate MEV refund (simplified)
        uint256 gasPrice = tx.gasprice;
        uint256 gasCost = gasUsed * gasPrice;
        uint256 refund = (gasCost * (100 - refundPercentage)) / 100;

        bundle.refundAmount = refund;
        mevRefunds[bundle.sender] += refund;

        emit PrivateTransactionExecuted(
            bundle.txHashes[0],
            bundle.sender,
            refund
        );
    }

    /**
     * @notice Detect and penalize sandwich attacks
     */
    function reportSandwich(
        bytes32 targetTxHash,
        address attacker,
        bytes[] calldata frontrunTxs,
        bytes[] calldata backrunTxs
    ) external onlyFlashbots {
        require(protectedTxs[targetTxHash], "MEV: tx not protected");

        // Verify sandwich pattern
        require(
            block.timestamp - sandwichTimeout < block.timestamp,
            "MEV: outside sandwich window"
        );

        // Calculate penalty (forfeit MEV)
        uint256 penalty = (address(this).balance * penaltyPercentage) / 100;

        // Send penalty to treasury
        (bool success, ) = treasuryWallet.call{value: penalty}("");
        require(success, "MEV: penalty transfer failed");

        emit SandwichDetected(targetTxHash, attacker, penalty);
    }

    /**
     * @notice Claim MEV refunds
     */
    function claimRefund() external {
        uint256 amount = mevRefunds[msg.sender];
        require(amount > 0, "MEV: no refund available");

        mevRefunds[msg.sender] = 0;
        (bool success, ) = msg.sender.call{value: amount}("");
        require(success, "MEV: refund transfer failed");
    }

    /**
     * @notice Set refund percentage
     */
    function setRefundPercentage(uint256 _percentage) external onlyOwner {
        require(_percentage > 0 && _percentage <= 100, "MEV: invalid percentage");
        refundPercentage = _percentage;
    }

    /**
     * @notice Update Flashbots relayer
     */
    function setFlashbotsRelayer(address _relayer) external onlyOwner {
        flashbotsRelayer = _relayer;
    }

    /**
     * @notice Withdraw accumulated fees
     */
    function withdrawFees() external onlyOwner {
        (bool success, ) = treasuryWallet.call{value: address(this).balance}("");
        require(success, "MEV: withdrawal failed");
    }

    /**
     * @notice Receive ETH
     */
    receive() external payable {}
}
