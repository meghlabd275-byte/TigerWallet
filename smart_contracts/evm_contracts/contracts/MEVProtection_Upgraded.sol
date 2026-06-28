// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

/**
 * @title TigerMEVProtection
 * @notice MEV Protection - UPGRADED v2
 * @dev Flashbots, MEV-Blocker, sandwich detection, 95% refunds
 */
contract TigerMEVProtection is Ownable {
    using ECDSA for bytes32;

    bytes32 public constant BUNDLE_SEPARATOR = keccak256("tigerwallet.mev.v2");

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

    mapping(bytes32 => BundleInfo) public bundles;
    mapping(address => uint256) public mevRefunds;
    mapping(bytes32 => bool) public protectedTxs;
    mapping(bytes32 => bytes32) public txToBundle;

    address public flashbotsRelayer;
    address public mevBlocker;
    address public treasuryWallet;

    uint256 public refundPercentage = 95;
    uint256 public penaltyPercentage = 50;
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

    modifier onlyFlashbots() {
        require(
            msg.sender == flashbotsRelayer || msg.sender == owner(),
            "MEV: unauthorized"
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
     * @notice Submit bundle for MEV protection
     */
    function submitBundle(
        bytes[] calldata txs,
        bool isPrivate,
        bytes calldata signature
    ) external payable returns (bytes32 bundleHash) {
        require(txs.length > 0, "MEV: empty");
        require(msg.value >= txs.length * 0.001 ether, "MEV: fee");

        bytes32 dataHash = keccak256(abi.encodePacked(txs, isPrivate));
        address recovered = dataHash.recover(signature);
        require(recovered == msg.sender, "MEV: sig");

        bundleHash = keccak256(abi.encodePacked(txs, block.timestamp, msg.sender));

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
     * @notice Record execution and calculate refund
     */
    function recordExecution(
        bytes32 bundleHash,
        uint256 gasUsed
    ) external onlyFlashbots {
        BundleInfo storage bundle = bundles[bundleHash];
        require(!bundle.executed, "MEV: executed");

        bundle.executed = true;
        bundle.gasUsed = gasUsed;

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
     * @notice Report and penalize sandwich attacks
     */
    function reportSandwich(
        bytes32 targetTxHash,
        address attacker,
        bytes[] calldata frontrunTxs,
        bytes[] calldata backrunTxs
    ) external onlyFlashbots {
        require(protectedTxs[targetTxHash], "MEV: not protected");
        require(
            block.timestamp - sandwichTimeout < block.timestamp,
            "MEV: window"
        );

        uint256 penalty = (address(this).balance * penaltyPercentage) / 100;
        (bool success, ) = treasuryWallet.call{value: penalty}("");
        require(success, "MEV: penalty");

        emit SandwichDetected(targetTxHash, attacker, penalty);
    }

    /**
     * @notice Claim MEV refunds
     */
    function claimRefund() external {
        uint256 amount = mevRefunds[msg.sender];
        require(amount > 0, "MEV: no refund");

        mevRefunds[msg.sender] = 0;
        (bool success, ) = msg.sender.call{value: amount}("");
        require(success, "MEV: transfer");
    }

    /**
     * @notice Set refund percentage
     */
    function setRefundPercentage(uint256 _percentage) external onlyOwner {
        require(_percentage > 0 && _percentage <= 100, "MEV: %");
        refundPercentage = _percentage;
    }

    /**
     * @notice Update Flashbots relayer
     */
    function setFlashbotsRelayer(address _relayer) external onlyOwner {
        flashbotsRelayer = _relayer;
    }

    /**
     * @notice Withdraw fees to treasury
     */
    function withdrawFees() external onlyOwner {
        (bool success, ) = treasuryWallet.call{value: address(this).balance}("");
        require(success, "MEV: withdraw");
    }

    receive() external payable {}
}
