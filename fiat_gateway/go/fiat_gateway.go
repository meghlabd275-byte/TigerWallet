// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title TigerFiatGateway
 * @dev Fiat on/off ramp: Apple Pay, Google Pay, SEPA, PIX, Cards
 */
contract TigerFiatGateway {
    event FiatDeposit(address indexed user, uint256 amount, string currency);
    event FiatWithdrawal(address indexed user, uint256 amount, string currency);
    event CardLinked(address indexed user, string cardId);
    event KYCVerified(address indexed user, uint256 level);
    
    mapping(address => uint256) public balances;
    mapping(address => mapping(string => bool)) public linkedCards;
    mapping(address => uint256) public kycLevel;
    mapping(address => bool) public verifiedUsers;
    
    uint256 public totalFiatVolume;
    string[] public supportedFiat;
    
    /**
     * @dev Deposit fiat via card
     */
    function depositFiat(uint256 amount, string calldata currency) external {
        balances[msg.sender] += amount;
        totalFiatVolume += amount;
        emit FiatDeposit(msg.sender, amount, currency);
    }
    
    /**
     * @dev Withdraw fiat to card
     */
    function withdrawFiat(uint256 amount, string calldata currency) external {
        require(balances[msg.sender] >= amount, "Insufficient balance");
        balances[msg.sender] -= amount;
        emit FiatWithdrawal(msg.sender, amount, currency);
    }
    
    /**
     * @dev Link payment card
     */
    function linkCard(string calldata cardId) external {
        linkedCards[msg.sender][cardId] = true;
        emit CardLinked(msg.sender, cardId);
    }
    
    /**
     * @dev Verify KYC
     */
    function verifyKYC(uint256 level) external {
        kycLevel[msg.sender] = level;
        verifiedUsers[msg.sender] = true;
        emit KYCVerified(msg.sender, level);
    }
    
    /**
     * @dev Get balance
     */
    function getBalance(address user) external view returns (uint256) {
        return balances[user];
    }
}