// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title TigerCustody
 * @dev Institutional MPC Custody Smart Contract
 * @dev Multi-party computation, treasury management, RBAC, compliance
 */
contract TigerCustody {
    // Events
    event WalletCreated(
        address indexed wallet,
        address indexed owner,
        uint256 walletType
    );
    event Deposit(
        address indexed wallet,
        address indexed depositor,
        uint256 amount
    );
    event Withdrawal(
        address indexed wallet,
        address indexed recipient,
        uint256 amount,
        bytes32 requestId
    );
    event Transfer(
        address indexed from,
        address indexed to,
        uint256 amount
    );
    event RoleGranted(
        address indexed wallet,
        address indexed account,
        uint8 role
    );
    event RoleRevoked(
        address indexed wallet,
        address indexed account,
        uint8 role
    );
    event PolicyUpdated(
        address indexed wallet,
        bytes32 policyHash
    );
    event RequestCreated(
        address indexed wallet,
        bytes32 requestId,
        uint8 requestType
    );
    event RequestApproved(
        address indexed wallet,
        bytes32 requestId,
        address indexed approver
    );
    event RequestExecuted(
        address indexed wallet,
        bytes32 requestId
    );
    event RequestCancelled(
        address indexed wallet,
        bytes32 requestId
    );
    event ComplianceReport(
        address indexed wallet,
        bytes32 reportHash,
        uint256 timestamp
    );
    event EmergencyFreeze(
        address indexed wallet,
        address indexed caller
    );
    event EmergencyUnfreeze(
        address indexed wallet,
        address indexed caller
    );

    // Constants
    uint256 public constant ROLE_OWNER = 1;
    uint256 public constant ROLE_ADMIN = 2;
    uint256 public constant ROLE_SIGNER = 3;
    uint256 public constant ROLE_OBSERVER = 4;
    uint256 public constant ROLE_COMPLIANCE = 5;
    uint256 public constant ROLE_LIMIT = 6;

    uint256 public constant REQUEST_WITHDRAWAL = 1;
    uint256 public constant REQUEST_TRANSFER = 2;
    uint256 public constant REQUEST_SETTINGS = 3;
    uint256 public constant REQUEST_EMERGENCY = 4;

    uint256 public constant POLICY_ALLOW_ALL = 1;
    uint256 public constant POLICY_BLOCK_HIGH_RISK = 2;
    uint256 public constant POLICY_COMPLIANCE_REQUIRED = 3;

    // Structs
    struct Wallet {
        address owner;
        uint256 walletType;
        uint256 createdAt;
        bool frozen;
        uint256 freezeTime;
        bytes32 policyHash;
    }

    struct Request {
        bytes32 id;
        address wallet;
        uint8 requestType;
        address recipient;
        uint256 amount;
        bytes data;
        uint256 approvals;
        uint256 requiredApprovals;
        uint256 createdAt;
        uint256 executedAt;
        bool executed;
        bool cancelled;
    }

    struct Role {
        address account;
        uint256 role;
        uint256 grantedAt;
    }

    struct ComplianceData {
        address wallet;
        uint256 transactionCount;
        uint256 totalVolume;
        uint256 lastTransactionTime;
        uint256 dailyLimit;
        uint256 dailySpent;
        uint256 dailyResetTime;
        address[] restrictedAssets;
    }

    // State
    mapping(address => Wallet) public wallets;
    mapping(address => address[]) public walletOwners;
    mapping(bytes32 => Request) public requests;
    mapping(address => Role[]) public walletRoles;
    mapping(address => mapping(address => bool)) public roleHolders;
    mapping(address => ComplianceData) public complianceData;
    mapping(address => bool) public globalFreeze;

    // Treasury
    address public treasury;
    uint256 public treasuryBalance;

    // Governance
    address public governance;
    uint256 public proposalThreshold = 1e18;
    uint256 public votingPeriod = 3 days;

    // Modifiers
    modifier onlyWalletOwner(address wallet) {
        require(wallets[wallet].owner == msg.sender, "Not owner");
        _;
    }

    modifier onlyWalletRole(address wallet, uint256 role) {
        require(roleHolders[wallet][msg.sender], "Not authorized");
        _;
    }

    modifier whenNotFrozen(address wallet) {
        require(!wallets[wallet].frozen, "Wallet frozen");
        require(!globalFreeze[msg.sender], "Global freeze active");
        _;
    }

    /**
     * @dev Constructor
     * @param _treasury Treasury address
     * @param _governance Governance address
     */
    constructor(address _treasury, address _governance) {
        require(_treasury != address(0), "Zero treasury");
        require(_governance != address(0), "Zero governance");
        
        treasury = _treasury;
        governance = _governance;
    }

    /**
     * @dev Create new custody wallet
     * @param owners Initial owners
     * @param signers Required signers
     * @param requiredApprovals Required approvals
     * @param walletType Wallet type
     * @param policyHash Policy hash
     * @return wallet Address
     */
    function createWallet(
        address[] memory owners,
        address[] memory signers,
        uint256 requiredApprovals,
        uint256 walletType,
        bytes32 policyHash
    ) external returns (address wallet) {
        require(owners.length > 0, "No owners");
        require(signers.length >= requiredApprovals, "Invalid approvals");
        require(walletType <= 5, "Invalid type");
        
        // Generate deterministic wallet address
        wallet = computeWalletAddress(owners, wallets[msg.sender].createdAt);
        
        // Create wallet
        wallets[wallet] = Wallet({
            owner: owners[0],
            walletType: walletType,
            createdAt: block.timestamp,
            frozen: false,
            freezeTime: 0,
            policyHash: policyHash
        });
        
        // Add owners
        for (uint256 i = 0; i < owners.length; ) {
            walletOwners[wallet].push(owners[i]);
            unchecked {
                i++;
            }
        }
        
        // Add roles
        for (uint256 i = 0; i < signers.length; ) {
            _grantRole(wallet, signers[i], ROLE_SIGNER);
            unchecked {
                i++;
            }
        }
        
        // Initialize compliance data
        complianceData[wallet] = ComplianceData({
            wallet: wallet,
            transactionCount: 0,
            totalVolume: 0,
            lastTransactionTime: block.timestamp,
            dailyLimit: 0,
            dailySpent: 0,
            dailyResetTime: block.timestamp,
            restrictedAssets: new address[](0)
        });
        
        emit WalletCreated(wallet, owners[0], walletType);
    }

    /**
     * @dev Compute wallet address
     */
    function computeWalletAddress(
        address[] memory owners,
        uint256 salt
    ) public pure returns (address) {
        bytes32 hash = keccak256(abi.encodePacked(owners, salt));
        return address(uint160(uint256(hash)));
    }

    /**
     * @dev Deposit
     */
    function deposit(address wallet) external payable whenNotFrozen(wallet) {
        require(msg.value > 0, "No value");
        
        complianceData[wallet].totalVolume += msg.value;
        complianceData[wallet].lastTransactionTime = block.timestamp;
        complianceData[wallet].transactionCount++;
        
        emit Deposit(wallet, msg.sender, msg.value);
    }

    /**
     * @dev Withdraw
     * @param wallet Wallet address
     * @param amount Amount
     * @param recipient Recipient
     * @param data Additional data
     */
    function withdraw(
        address wallet,
        uint256 amount,
        address recipient,
        bytes memory data
    ) external whenNotFrozen(wallet) {
        require(wallets[wallet].owner == msg.sender, "Not owner");
        require(amount > 0, "No amount");
        require(recipient != address(0), "Zero recipient");
        
        // Check daily limit
        _checkDailyLimit(wallet, amount);
        
        // Check compliance
        _checkCompliance(wallet, recipient, amount);
        
        // Create request
        bytes32 requestId = keccak256(abi.encodePacked(
            wallet,
            amount,
            recipient,
            block.timestamp
        ));
        
        requests[requestId] = Request({
            id: requestId,
            wallet: wallet,
            requestType: uint8(REQUEST_WITHDRAWAL),
            recipient: recipient,
            amount: amount,
            data: data,
            approvals: 0,
            requiredApprovals: 1,
            createdAt: block.timestamp,
            executedAt: 0,
            executed: false,
            cancelled: false
        });
        
        emit RequestCreated(wallet, requestId, REQUEST_WITHDRAWAL);
        
        // Execute if auto-approved
        _executeRequest(requestId);
    }

    /**
     * @dev Approve request
     */
    function approveRequest(bytes32 requestId) external {
        Request storage request = requests[requestId];
        require(request.wallet != address(0), "Not found");
        require(!request.executed, "Executed");
        require(!request.cancelled, "Cancelled");
        
        // Verify role
        require(
            roleHolders[request.wallet][msg.sender] || 
            wallets[request.wallet].owner == msg.sender,
            "Not authorized"
        );
        
        request.approvals++;
        
        emit RequestApproved(request.wallet, requestId, msg.sender);
        
        // Execute if threshold reached
        if (request.approvals >= request.requiredApprovals) {
            _executeRequest(requestId);
        }
    }

    /**
     * @dev Execute request
     */
    function _executeRequest(bytes32 requestId) internal {
        Request storage request = requests[requestId];
        
        if (request.requestType == REQUEST_WITHDRAWAL) {
            (bool success, ) = request.recipient.call{value: request.amount}("");
            require(success, "Transfer failed");
            
            emit Withdrawal(
                request.wallet,
                request.recipient,
                request.amount,
                requestId
            );
        }
        
        request.executed = true;
        request.executedAt = block.timestamp;
        
        emit RequestExecuted(request.wallet, requestId);
    }

    /**
     * @dev Grant role
     */
    function grantRole(
        address wallet,
        address account,
        uint256 role
    ) external onlyWalletOwner(wallet) {
        _grantRole(wallet, account, role);
    }

    /**
     * @dev Internal grant role
     */
    function _grantRole(
        address wallet,
        address account,
        uint256 role
    ) internal {
        require(!roleHolders[wallet][account], "Already has role");
        
        walletRoles[wallet].push(Role({
            account: account,
            role: role,
            grantedAt: block.timestamp
        }));
        
        roleHolders[wallet][account] = true;
        
        emit RoleGranted(wallet, account, uint8(role));
    }

    /**
     * @dev Revoke role
     */
    function revokeRole(
        address wallet,
        address account
    ) external onlyWalletOwner(wallet) {
        require(roleHolders[wallet][account], "No role");
        
        roleHolders[wallet][account] = false;
        
        emit RoleRevoked(wallet, account, 0);
    }

    /**
     * @dev Update policy
     */
    function updatePolicy(
        address wallet,
        bytes32 policyHash
    ) external onlyWalletOwner(wallet) {
        wallets[wallet].policyHash = policyHash;
        
        emit PolicyUpdated(wallet, policyHash);
    }

    /**
     * @dev Set daily limit
     */
    function setDailyLimit(
        address wallet,
        uint256 limit
    ) external onlyWalletOwner(wallet) {
        complianceData[wallet].dailyLimit = limit;
    }

    /**
     * @dev Check daily limit
     */
    function _checkDailyLimit(address wallet, uint256 amount) internal view {
        ComplianceData storage data = complianceData[wallet];
        
        if (data.dailyLimit > 0) {
            // Reset daily spent if new day
            if (block.timestamp >= data.dailyResetTime + 1 days) {
                data.dailySpent = 0;
                data.dailyResetTime = block.timestamp;
            }
            
            require(
                data.dailySpent + amount <= data.dailyLimit,
                "Daily limit exceeded"
            );
        }
    }

    /**
     * @dev Check compliance
     */
    function _checkCompliance(
        address wallet,
        address recipient,
        uint256 amount
    ) internal view {
        Wallet storage walletData = wallets[wallet];
        
        // Check policy
        if (walletData.policyHash == bytes32(POLICY_COMPLIANCE_REQUIRED)) {
            require(
                recipient != address(0) && recipient != wallet,
                "Compliance violation"
            );
        }
    }

    /**
     * @dev Emergency freeze
     */
    function emergencyFreeze(address wallet) external {
        require(msg.sender == governance, "Not governance");
        
        wallets[wallet].frozen = true;
        wallets[wallet].freezeTime = block.timestamp;
        
        emit EmergencyFreeze(wallet, msg.sender);
    }

    /**
     * @dev Emergency unfreeze
     */
    function emergencyUnfreeze(address wallet) external {
        require(msg.sender == governance, "Not governance");
        
        wallets[wallet].frozen = false;
        wallets[wallet].freezeTime = 0;
        
        emit EmergencyUnfreeze(wallet, msg.sender);
    }

    /**
     * @dev Generate compliance report
     */
    function generateComplianceReport(
        address wallet
    ) external returns (bytes32) {
        require(
            roleHolders[wallet][msg.sender] || 
            wallets[wallet].owner == msg.sender,
            "Not authorized"
        );
        
        ComplianceData storage data = complianceData[wallet];
        
        bytes32 reportHash = keccak256(abi.encodePacked(
            wallet,
            data.transactionCount,
            data.totalVolume,
            data.lastTransactionTime,
            block.timestamp
        ));
        
        emit ComplianceReport(wallet, reportHash, block.timestamp);
        
        return reportHash;
    }

    /**
     * @dev Get wallet info
     */
    function getWalletInfo(
        address wallet
    ) external view returns (
        address owner,
        uint256 walletType,
        uint256 createdAt,
        bool frozen
    ) {
        Wallet storage w = wallets[wallet];
        return (w.owner, w.walletType, w.createdAt, w.frozen);
    }

    /**
     * @dev Get roles
     */
    function getRoles(
        address wallet
    ) external view returns (address[] memory) {
        Role[] storage roles = walletRoles[wallet];
        address[] memory result = new address[](roles.length);
        
        for (uint256 i = 0; i < roles.length; ) {
            result[i] = roles[i].account;
            unchecked {
                i++;
            }
        }
        
        return result;
    }

    /**
     * @dev Get compliance data
     */
    function getComplianceData(
        address wallet
    ) external view returns (
        uint256 transactionCount,
        uint256 totalVolume,
        uint256 dailyLimit,
        uint256 dailySpent
    ) {
        ComplianceData storage data = complianceData[wallet];
        return (
            data.transactionCount,
            data.totalVolume,
            data.dailyLimit,
            data.dailySpent
        );
    }

    receive() external payable {
        treasuryBalance += msg.value;
    }
}