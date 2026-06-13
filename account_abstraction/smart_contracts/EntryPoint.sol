// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title EntryPoint
 * @dev ERC-4337 EntryPoint Contract
 * @dev Minimal ERC-4337 entry point implementation with enhanced security
 * Based on ERC-4337 specification with additional gas optimizations
 */
contract EntryPoint {
    // Revert reasons
    string internal constant REVERT_REASON_INVALID_OPERATION = "Invalid operation";
    string internal constant REVERT_REASON_INVALID_SIGNATURE = "Invalid signature";
    string internal constant REVERT_REASON_INVALID_OWNER = "Invalid owner";
    string internal constant REVERT_REASON_INVALID_CALLER = "Invalid caller";
    string internal constant REVERT_REASON_INSUFFICIENT_FUNDS = "Insufficient funds";
    string internal constant REVERT_REASON_ALREADY_INITIALIZED = "Already initialized";
    string internal constant REVERT_REASON_ZERO_ADDRESS = "Zero address";
    string internal constant REVERT_REASON_INVALID_GAS = "Invalid gas limit";
    string internal constant REVERT_REASON_INVALID_STATE = "Invalid state";
    string internal constant REVERT_REASON_VALIDATION_FAILED = "Validation failed";
    string internal constant REVERT_REASON_PAYMASTER = "Paymaster verification failed";

    // Event emitted for each user operation
    event UserOperationEvent(
        bytes32 indexed userOpHash,
        address indexed sender,
        address indexed paymaster,
        uint256 nonce,
        bool success,
        uint256 actualGasCost,
        bytes32[] factoryData
    );

    // Event for deposit changes
    event Deposited(address indexed account, uint256 totalDeposit);

    // Event for withdrawal
    event Withdrawn(
        address indexed account,
        address indexed withdrawer,
        uint256 amount
    );

    // Event for stake management
    event StakeLocked(
        address indexed account,
        uint256 stake,
        uint256 withdrawTime
    );

    event StakeUnlocked(address indexed account, uint256 withdrawTime);

    event StakeWithdrawn(
        address indexed account,
        address indexed withdrawer,
        uint256 amount
    );

    // Struct for user operation
    struct UserOperation {
        address sender;
        uint256 nonce;
        bytes initCode;
        bytes callData;
        uint256 callGasLimit;
        uint256 verificationGasLimit;
        uint256 preVerificationGas;
        bytes paymasterData;
        bytes signature;
    }

    // Struct for sender information
    struct SenderInfo {
        uint256 nonce;
        uint256 deposit;
    }

    // Struct for stake information
    struct StakeInfo {
        uint256 stake;
        uint256 withdrawTime;
    }

    // Sender address => nonce map
    mapping(address => uint256) internal senderNonce;

    // Sender address => deposit
    mapping(address => uint256) internal senderDeposit;

    // Account => StakeInfo
    mapping(address => StakeInfo) internal stakes;

    // Factory => deployed addresses (stored as mapping for tracking)
    mapping(address => mapping(uint256 => address)) internal factoryDeployed;

    // Factory deployed count
    mapping(address => uint256) internal factoryDeployedCount;

    // Paymaster stake
    mapping(address => uint256) internal paymasterStake;

    // Paymaster deposit
    mapping(address => uint256) internal paymasterDeposit;

    // Gas excess
    uint256 public gasExcess;

    // Minimum stake amount
    uint256 public immutable MIN_STAKE_VALUE = 1e18; // 1 ETH

    // Minimum unstake delay
    uint256 public immutable UNSTAKE_DELAY = 2 days;

    // Maximum gas for verification
    uint256 public immutable MAX_VERIFICATION_GAS = 5e6;

    // Validation function selector
    bytes4 internal constant VALIDATION_SELECTOR = 0x1b2fbb4f;

    // Execution function selector
    bytes4 internal constant EXECUTION_SELECTOR = 0x8f2eb2b1;

    // EIP-1559 chain ID (dynamic)
    uint256 internal chainId;

    // Hash of entry point
    bytes32 internal entryPointHash;

    // Marker for initialized
    bool internal initialized;

    /**
     * @dev Initialize the entry point
     */
    constructor() {
        chainId = block.chainid;
        entryPointHash = keccak256(
            abi.encode(
                "EntryPoint",
                "1.0.0",
                block.chainid,
                address(this)
            )
        );
        initialized = true;
    }

    /**
     * @dev Get nonce for sender
     * @param sender The sender address
     * @return Current nonce
     */
    function getNonce(address sender) public view returns (uint256) {
        return senderNonce[sender];
    }

    /**
     * @dev Get deposit for account
     * @param account The account address
     * @return Current deposit
     */
    function getDeposit(address account) public view returns (uint256) {
        return senderDeposit[account];
    }

    /**
     * @dev Get stake info
     * @param account The account address
     * @return stake amount and withdraw time
     */
    function getStakeInfo(address account) public view returns (uint256, uint256) {
        StakeInfo memory stakeInfo = stakes[account];
        return (stakeInfo.stake, stakeInfo.withdrawTime);
    }

    /**
     * @dev Add deposit for account
     * @param account The account to deposit to
     */
    function depositTo(address account) public payable {
        require(account != address(0), REVERT_REASON_ZERO_ADDRESS);
        senderDeposit[account] += msg.value;
        emit Deposited(account, senderDeposit[account]);
    }

    /**
     * @dev Add stake for account (requires MIN_STAKE_VALUE)
     * @param account The account to add stake for
     */
    function addStake(uint256 unstakeDelay) public payable {
        require(msg.value >= MIN_STAKE_VALUE, REVERT_REASON_INSUFFICIENT_FUNDS);
        require(unstakeDelay >= UNSTAKE_DELAY, REVERT_REASON_INVALID_STATE);
        
        StakeInfo storage stakeInfo = stakes[msg.sender];
        
        // If already locked, add to existing stake
        if (stakeInfo.stake > 0) {
            require(stakeInfo.withdrawTime == 0, REVERT_REASON_INVALID_STATE);
            stakeInfo.stake += msg.value;
        } else {
            stakeInfo.stake = msg.value;
            stakeInfo.withdrawTime = 0;
        }
        
        emit StakeLocked(msg.sender, stakeInfo.stake, stakeInfo.withdrawTime);
    }

    /**
     * @dev Unlock stake
     */
    function unlockStake() public {
        StakeInfo storage stakeInfo = stakes[msg.sender];
        require(stakeInfo.stake > 0, REVERT_REASON_INVALID_STATE);
        require(stakeInfo.withdrawTime == 0, REVERT_REASON_ALREADY_INITIALIZED);
        
        stakeInfo.withdrawTime = block.timestamp + UNSTAKE_DELAY;
        
        emit StakeUnlocked(msg.sender, stakeInfo.withdrawTime);
    }

    /**
     * @dev Withdraw stake
     * @param withdrawAddress Address to send withdrawn stake
     */
    function withdrawStake(address payable withdrawAddress) public {
        require(withdrawAddress != address(0), REVERT_REASON_ZERO_ADDRESS);
        
        StakeInfo storage stakeInfo = stakes[msg.sender];
        require(stakeInfo.stake > 0, REVERT_REASON_INVALID_STATE);
        require(stakeInfo.withdrawTime > 0, REVERT_REASON_INVALID_STATE);
        require(block.timestamp >= stakeInfo.withdrawTime, REVERT_REASON_INVALID_STATE);
        
        uint256 amount = stakeInfo.stake;
        stakeInfo.stake = 0;
        stakeInfo.withdrawTime = 0;
        
        (bool success, ) = withdrawAddress.call{value: amount}("");
        require(success, REVERT_REASON_INVALID_CALLER);
        
        emit StakeWithdrawn(msg.sender, withdrawAddress, amount);
    }

    /**
     * @dev Withdraw deposit
     * @param withdrawAddress Address to send withdrawn deposit
     * @param amount Amount to withdraw
     */
    function withdrawFrom(address payable withdrawAddress, uint256 amount) public {
        require(withdrawAddress != address(0), REVERT_REASON_ZERO_ADDRESS);
        
        address payable account = payable(msg.sender);
        require(senderDeposit[account] >= amount, REVERT_REASON_INSUFFICIENT_FUNDS);
        
        senderDeposit[account] -= amount;
        
        (bool success, ) = withdrawAddress.call{value: amount}("");
        require(success, REVERT_REASON_INVALID_CALLER);
        
        emit Withdrawn(account, withdrawAddress, amount);
    }

    /**
     * @dev Internal function to execute user operation
     * @param op The user operation
     * @param beneficiary The beneficiary address
     * @return actualGasCost The actual gas cost
     */
    function _executeUserOp(
        UserOperation calldata op,
        address payable beneficiary
    ) internal returns (uint256) {
        uint256 gasBefore = gasleft();
        
        // Execute the call
        if (op.callData.length > 0) {
            (bool success, ) = op.sender.call{gas: op.callGasLimit}(op.callData);
            require(success, REVERT_REASON_INVALID_CALLER);
        }
        
        uint256 gasUsed = gasBefore - gasleft();
        uint256 actualGasCost = gasUsed * block.basefee;
        
        // Refund excess gas
        if (msg.value > actualGasCost) {
            uint256 refund = msg.value - actualGasCost;
            (bool success, ) = beneficiary.call{value: refund}("");
            require(success, REVERT_REASON_INVALID_CALLER);
        }
        
        return actualGasCost;
    }

    /**
     * @dev Internal function to validate user operation
     * @param op The user operation
     * @param opHash The operation hash
     * @return sigFailed Whether signature validation failed
     * @return validationData Encoded validation data
     */
    function _validateUserOp(
        UserOperation calldata op,
        bytes32 opHash
    ) internal returns (uint256 sigFailed, uint256 validationData) {
        // Initialize account if needed
        if (op.initCode.length > 0) {
            // Decode factory and factory data from initCode
            (address factory, bytes memory factoryData) = _decodeInitCode(
                op.initCode
            );
            
            // Execute account creation via factory
            (bool success, ) = factory.call{value: 0}(factoryData);
            require(success, REVERT_REASON_VALIDATION_FAILED);
        }
        
        // Call validateUserOp on the account
        (bool success, bytes memory result) = op.sender.call{value: 0}(
            abi.encodeWithSelector(
                VALIDATION_SELECTOR,
                opHash,
                op.nonce,
                op.signature
            )
        );
        
        if (!success || result.length == 0) {
            return (1, 0);
        }
        
        (sigFailed, validationData) = abi.decode(result, (uint256, uint256));
        
        return (sigFailed, validationData);
    }

    /**
     * @dev Decode init code into factory and data
     * @param initCode The init code
     * @return factory The factory address
     * @return factoryData The factory data
     */
    function _decodeInitCode(
        bytes memory initCode
    ) internal pure returns (address factory, bytes memory factoryData) {
        require(initCode.length >= 20, REVERT_REASON_INVALID_CALLER);
        
        factory = address(bytes20(initCode[:20]));
        
        if (initCode.length > 20) {
            factoryData = initCode[20:];
        }
    }

    /**
     * @dev Handle user operations
     * @param ops The array of user operations
     * @param beneficiary The beneficiary address for refunds
     */
    function handleOps(
        UserOperation[] calldata ops,
        address payable beneficiary
    ) public payable {
        _handleOps(ops, beneficiary, msg.sender);
    }

    /**
     * @dev Internal handle operations
     */
    function _handleOps(
        UserOperation[] calldata ops,
        address payable beneficiary,
        address msgSender
    ) internal {
        // Use unchecked math for gas accounting
        uint256 gasExcessCounter = gasExcess;
        
        for (uint256 i = 0; i < ops.length; ) {
            UserOperation calldata op = ops[i];
            
            bytes32 opHash = getUserOpHash(op);
            
            // Validate user operation
            (uint256 sigFailed, uint256 validationData) = _validateUserOp(
                op,
                opHash
            );
            
            // Calculate required gas
            uint256 requiredGas = op.verificationGasLimit +
                op.callGasLimit +
                op.preVerificationGas;
            
            // Execute operation
            uint256 actualGasCost;
            bool success;
            
            if (sigFailed == 0) {
                // Execute the user operation
                actualGasCost = _executeUserOp(op, beneficiary);
                success = true;
            } else {
                success = false;
                actualGasCost = 0;
            }
            
            // Update nonce
            if (sigFailed == 0) {
                senderNonce[op.sender] = op.nonce + 1;
            }
            
            // Update gas excess
            gasExcessCounter += requiredGas - (requiredGas / 64);
            
            // Emit event
            emit UserOperationEvent(
                opHash,
                op.sender,
                address(0),
                op.nonce,
                success,
                actualGasCost,
                new bytes32[](0)
            );
            
            unchecked {
                i++;
            }
        }
        
        gasExcess = gasExcessCounter;
    }

    /**
     * @dev Get user operation hash
     * @param op The user operation
     * @return Hash of the user operation
     */
    function getUserOpHash(
        UserOperation calldata op
    ) public view returns (bytes32) {
        return
            keccak256(
                abi.encode(
                    op.sender,
                    op.nonce,
                    keccak256(op.initCode),
                    keccak256(op.callData),
                    op.callGasLimit,
                    op.verificationGasLimit,
                    op.preVerificationGas,
                    block.chainid,
                    address(this),
                    keccak256(op.signature)
                )
            );
    }

    /**
     * @dev Get entry point version
     * @return Version string
     */
    function getEntryPointVersion() public pure returns (string memory) {
        return "1.0.0";
    }

    // Receive ETH
    receive() external payable {
        senderDeposit[address(0)] += msg.value;
    }

    // Fallback
    fallback() external payable {
        senderDeposit[address(0)] += msg.value;
    }
}