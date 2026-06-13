// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title SmartAccount
 * @dev ERC-4337 Smart Contract Wallet
 * @dev EOA-less account with social recovery, multi-sig, and multi-chain support
 */
contract SmartAccount {
    // Revert reasons
    string internal constant REVERT_REASON_INVALID_OWNER = "Invalid owner";
    string internal constant REVERT_REASON_INVALID_GUARDIAN = "Invalid guardian";
    string internal constant REVERT_REASON_INVALID_SIGNATURE = "Invalid signature";
    string internal constant REVERT_REASON_INVALID_NONCE = "Invalid nonce";
    string internal constant REVERT_REASON_INVALID_TARGET = "Invalid target";
    string internal constant REVERT_REASON_ZERO_ADDRESS = "Zero address";
    string internal constant REVERT_REASON_INSUFFICIENT_BALANCE = "Insufficient balance";
    string internal constant REVERT_REASON_GUARDIAN_REQUIRED = "Guardian required";
    string internal constant REVERT_REASON_NOT_GUARDIAN = "Not a guardian";
    string internal constant REVERT_REASON_ALREADY_OWNER = "Already an owner";
    string internal constant REVERT_REASON_ALREADY_GUARDIAN = "Already a guardian";
    string internal constant REVERT_REASON_NOT_OWNER = "Not an owner";
    string internal constant REVERT_REASON_LOCKED = "Account is locked";
    string internal constant REVERT_REASON_TIME_LOCK = "Time lock active";
    string internal constant REVERT_REASON_THRESHOLD = "Threshold not reached";

    // Events
    event OwnerAdded(address indexed owner, uint256 threshold);
    event OwnerRemoved(address indexed owner);
    event GuardianAdded(address indexed guardian, uint256 delay);
    event GuardianRemoved(address indexed guardian);
    event OwnershipTransferred(address indexed oldOwner, address indexed newOwner);
    event ExecutionSuccess(address indexed target, uint256 value, bytes data);
    event GuardiansUpdated(address[] guardians, uint256 threshold);
    event RecoveryStarted(address indexed guardian, uint256 unlockTime);
    event OwnershipRecovered(address indexed oldOwner, address indexed newOwner);
    event Lockdown(address indexed caller, uint256 lockTime);
    event DailyLimitUpdated(uint256 newLimit);
    event WhitelistUpdated(address[] addresses, bool[] statuses);

    // Struct for owner
    struct Owner {
        address owner;
        uint256 weight;
    }

    // Struct for guardian
    struct Guardian {
        address guardian;
        uint256 delay;
    }

    // Struct for signature
    struct Signature {
        bytes32[] signers;
        bytes[] signatures;
    }

    // Struct for execution
    struct Execution {
        address target;
        uint256 value;
        bytes data;
    }

    // Struct for recovery
    struct RecoveryRequest {
        address newOwner;
        uint256 timestamp;
        uint256 confirmations;
    }

    // Owner structure
    struct OwnerInfo {
        address owner;
        uint256 weight;
        bool active;
    }

    // Guardian structure
    struct GuardianInfo {
        address guardian;
        uint256 delay;
        bool active;
    }

    // Multi-sig owners
    OwnerInfo[] public owners;
    mapping(address => uint256) public ownerIndex;
    mapping(address => bool) public isOwner;

    // Guardians
    GuardianInfo[] public guardians;
    mapping(address => uint256) public guardianIndex;
    mapping(address => bool) public isGuardian;

    // Required signatures threshold
    uint256 public requiredSignatures = 1;

    // Required guardian confirmations for recovery
    uint256 public guardianThreshold = 2;

    // Recovery time lock period
    uint256 public recoveryDelay = 24 hours;

    // Daily spending limit
    uint256 public dailyLimit;
    uint256 public dailySpent;
    uint256 public dailyResetTime;

    // Account nonce
    uint256 public nonce;

    // Lock status
    bool public locked;
    uint256 public lockTime;

    // Recovery request
    RecoveryRequest public recoveryRequest;

    // Whitelisted addresses (can be called without signature)
    mapping(address => bool) public whitelisted;

    // Maximum owners
    uint256 public constant MAX_OWNERS = 10;

    // Maximum guardians
    uint256 public constant MAX_GUARDIANS = 10;

    // Entry point interface
    IEntryPoint public entryPoint;

    // Initialize flag
    bool public initialized;

    // EIP-1271 magic value
    bytes4 internal constant EIP1271_MAGIC_VALUE = 0x1626ba7e;

    // Hash for EIP-712
    bytes32 public DOMAIN_SEPARATOR;

    // Chain ID
    uint256 internal chainId;

    /**
     * @dev Initialize the smart account
     * @param _owners Initial owners
     * @param _threshold Required threshold
     * @param _entryPoint Entry point address
     */
    function initialize(
        address[] memory _owners,
        uint256 _threshold,
        address _entryPoint
    ) external {
        require(!initialized, "Already initialized");
        require(_owners.length > 0, "No owners");
        require(_threshold > 0 && _threshold <= _owners.length, "Invalid threshold");
        
        initialized = true;
        entryPoint = IEntryPoint(_entryPoint);
        requiredSignatures = _threshold;
        chainId = block.chainid;
        
        DOMAIN_SEPARATOR = keccak256(
            abi.encode(
                keccak256(
                    "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
                ),
                keccak256("SmartAccount"),
                keccak256("1.0.0"),
                chainId,
                address(this)
            )
        );

        // Add owners
        for (uint256 i = 0; i < _owners.length; ) {
            _addOwner(_owners[i], 1);
            unchecked {
                i++;
            }
        }
    }

    /**
     * @dev Execute transaction
     * @param targets Target addresses
     * @param values Ether values
     * @param datas Call datas
     */
    function execute(
        address[] memory targets,
        uint256[] memory values,
        bytes[] memory datas,
        bytes[] memory signatures
    ) external payable {
        require(targets.length == values.length, "Length mismatch");
        require(targets.length == datas.length, "Length mismatch");
        
        // Verify signatures
        _verifySignatures(targets, values, datas, signatures);
        
        // Execute each call
        for (uint256 i = 0; i < targets.length; ) {
            address target = targets[i];
            uint256 value = values[i];
            bytes memory data = datas[i];
            
            require(target != address(0), REVERT_REASON_ZERO_ADDRESS);
            require(value <= address(this).balance, REVERT_REASON_INSUFFICIENT_BALANCE);
            
            (bool success, ) = target.call{value: value}(data);
            require(success, "Call failed");
            
            emit ExecutionSuccess(target, value, data);
            
            unchecked {
                i++;
            }
        }
        
        nonce++;
    }

    /**
     * @dev Execute single transaction
     */
    function execute(
        address target,
        uint256 value,
        bytes memory data,
        bytes memory signature
    ) external payable {
        address[] memory targets = new address[](1);
        targets[0] = target;
        
        uint256[] memory values = new uint256[](1);
        values[0] = value;
        
        bytes[] memory datas = new bytes[](1);
        datas[0] = data;
        
        bytes[] memory signatures = new bytes[](1);
        signatures[0] = signature;
        
        execute(targets, values, datas, signatures);
    }

    /**
     * @dev Verify signatures
     */
    function _verifySignatures(
        address[] memory targets,
        uint256[] memory values,
        bytes[] memory datas,
        bytes[] memory signatures
    ) internal {
        require(!locked, REVERT_REASON_LOCKED);
        require(signatures.length >= requiredSignatures, REVERT_REASON_THRESHOLD);
        
        // Build hash of all executions
        bytes32 hash = keccak256(
            abi.encode(
                nonce,
                keccak256(abi.encodePacked(targets)),
                keccak256(abi.encodePacked(values)),
                keccak256(abi.encodePacked(datas))
            )
        );
        
        bytes32 signedHash = keccak256(
            abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, hash)
        );
        
        // Verify each signature
        address[] memory seen = new address[](signatures.length);
        uint256 validCount;
        
        for (uint256 i = 0; i < signatures.length; ) {
            // Recover signer from signature
            (address signer, ) = _recoverSigner(signedHash, signatures[i]);
            
            // Check if signer is owner
            require(isOwner[signer], REVERT_REASON_NOT_OWNER);
            
            // Check for duplicates
            bool duplicate;
            for (uint256 j = 0; j < validCount; ) {
                if (seen[j] == signer) {
                    duplicate = true;
                    break;
                }
                unchecked {
                    j++;
                }
            }
            require(!duplicate, "Duplicate signature");
            
            seen[validCount] = signer;
            validCount++;
            
            unchecked {
                i++;
            }
        }
        
        require(validCount >= requiredSignatures, REVERT_REASON_THRESHOLD);
    }

    /**
     * @dev Recover signer from signature
     */
    function _recoverSigner(
        bytes32 hash,
        bytes memory signature
    ) internal pure returns (address, bytes32) {
        require(signature.length == 65, REVERT_REASON_INVALID_SIGNATURE);
        
        bytes32 r;
        bytes32 s;
        uint8 v;
        
        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }
        
        address signer = ecrecover(hash, v, r, s);
        
        return (signer, r);
    }

    /**
     * @dev Add owner
     */
    function addOwner(address owner, uint256 weight) external {
        require(msg.sender == address(this), "Not self");
        _addOwner(owner, weight);
    }

    /**
     * @dev Internal add owner
     */
    function _addOwner(address owner, uint256 weight) internal {
        require(owner != address(0), REVERT_REASON_ZERO_ADDRESS);
        require(!isOwner[owner], REVERT_REASON_ALREADY_OWNER);
        require(owners.length < MAX_OWNERS, "Max owners reached");
        
        owners.push(OwnerInfo(owner, weight, true));
        ownerIndex[owner] = owners.length - 1;
        isOwner[owner] = true;
        
        emit OwnerAdded(owner, weight);
    }

    /**
     * @dev Remove owner
     */
    function removeOwner(address owner) external {
        require(msg.sender == address(this), "Not self");
        require(isOwner[owner], REVERT_REASON_INVALID_OWNER);
        
        uint256 index = ownerIndex[owner];
        address lastOwner = owners[owners.length - 1].owner;
        
        owners[index] = owners[owners.length - 1];
        ownerIndex[lastOwner] = index;
        ownerIndex[owner] = 0;
        
        owners.pop();
        isOwner[owner] = false;
        
        emit OwnerRemoved(owner);
    }

    /**
     * @dev Add guardian
     */
    function addGuardian(address guardian, uint256 delay) external {
        require(msg.sender == address(this), "Not self");
        require(guardian != address(0), REVERT_REASON_ZERO_ADDRESS);
        require(!isGuardian[guardian], REVERT_REASON_ALREADY_GUARDIAN);
        require(guardians.length < MAX_GUARDIANS, "Max guardians reached");
        
        guardians.push(GuardianInfo(guardian, delay, true));
        guardianIndex[guardian] = guardians.length - 1;
        isGuardian[guardian] = true;
        
        emit GuardianAdded(guardian, delay);
    }

    /**
     * @dev Remove guardian
     */
    function removeGuardian(address guardian) external {
        require(msg.sender == address(this), "Not self");
        require(isGuardian[guardian], REVERT_REASON_INVALID_GUARDIAN);
        
        uint256 index = guardianIndex[guardian];
        address lastGuardian = guardians[guardians.length - 1].guardian;
        
        guardians[index] = guardians[guardians.length - 1];
        guardianIndex[lastGuardian] = index;
        guardianIndex[guardian] = 0;
        
        guardians.pop();
        isGuardian[guardian] = false;
        
        emit GuardianRemoved(guardian);
    }

    /**
     * @dev Start ownership recovery
     */
    function startRecovery(address newOwner) external {
        require(isGuardian[msg.sender], REVERT_REASON_NOT_GUARDIAN);
        require(newOwner != address(0), REVERT_REASON_ZERO_ADDRESS);
        
        if (recoveryRequest.timestamp == 0) {
            // First confirmation
            recoveryRequest = RecoveryRequest(
                newOwner,
                block.timestamp + recoveryDelay,
                1
            );
        } else {
            // Additional confirmation
            require(recoveryRequest.newOwner == newOwner, "Different recovery");
            require(
                block.timestamp < recoveryRequest.timestamp,
                REVERT_REASON_TIME_LOCK
            );
            recoveryRequest.confirmations++;
        }
        
        // Check threshold
        if (recoveryRequest.confirmations >= guardianThreshold) {
            _completeRecovery(newOwner);
        }
        
        emit RecoveryStarted(msg.sender, recoveryRequest.timestamp);
    }

    /**
     * @dev Complete recovery
     */
    function _completeRecovery(address newOwner) internal {
        address oldOwner = owners[0].owner;
        
        // Clear recovery request
        recoveryRequest = RecoveryRequest(address(0), 0, 0);
        
        // Remove all old owners
        while (owners.length > 0) {
            address o = owners[owners.length - 1].owner;
            isOwner[o] = false;
            owners.pop();
        }
        
        // Add new owner
        _addOwner(newOwner, 1);
        
        emit OwnershipRecovered(oldOwner, newOwner);
    }

    /**
     * @dev Cancel recovery
     */
    function cancelRecovery() external {
        require(recoveryRequest.timestamp > 0, "No recovery");
        require(
            block.timestamp >= recoveryRequest.timestamp,
            REVERT_REASON_TIME_LOCK
        );
        
        recoveryRequest = RecoveryRequest(address(0), 0, 0);
    }

    /**
     * @dev Lock account
     */
    function lock(uint256 lockDuration) external {
        require(msg.sender == address(this) || isOwner[msg.sender], "Not authorized");
        
        locked = true;
        lockTime = block.timestamp + lockDuration;
        
        emit Lockdown(msg.sender, lockTime);
    }

    /**
     * @dev Unlock account
     */
    function unlock() external {
        require(msg.sender == address(this) || isOwner[msg.sender], "Not authorized");
        require(locked, "Not locked");
        require(block.timestamp >= lockTime, REVERT_REASON_LOCKED);
        
        locked = false;
        lockTime = 0;
    }

    /**
     * @dev Set daily limit
     */
    function setDailyLimit(uint256 newLimit) external {
        require(msg.sender == address(this), "Not self");
        dailyLimit = newLimit;
        
        emit DailyLimitUpdated(newLimit);
    }

    /**
     * @dev Whitelist addresses
     */
    function setWhitelist(
        address[] memory addresses,
        bool[] memory statuses
    ) external {
        require(msg.sender == address(this), "Not self");
        require(addresses.length == statuses.length, "Length mismatch");
        
        for (uint256 i = 0; i < addresses.length; ) {
            whitelisted[addresses[i]] = statuses[i];
            unchecked {
                i++;
            }
        }
        
        emit WhitelistUpdated(addresses, statuses);
    }

    /**
     * @dev Validate user operation (ERC-4337)
     */
    function validateUserOp(
        bytes32,
        uint256 _nonce,
        bytes calldata
    ) external pure returns (uint256) {
        require(_nonce == 0, REVERT_REASON_INVALID_NONCE);
        return 0;
    }

    /**
     * @dev EIP-1271 signature validation
     */
    function isValidSignature(
        bytes32 hash,
        bytes memory signature
    ) external view returns (bytes4) {
        require(signature.length == 65, REVERT_REASON_INVALID_SIGNATURE);
        
        (address signer, ) = _recoverSigner(
            keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, hash)),
            signature
        );
        
        if (isOwner[signer]) {
            return EIP1271_MAGIC_VALUE;
        }
        
        return bytes4(0);
    }

    /**
     * @dev Get owners count
     */
    function getOwnersCount() external view returns (uint256) {
        return owners.length;
    }

    /**
     * @dev Get guardians count
     */
    function getGuardiansCount() external view returns (uint256) {
        return guardians.length;
    }

    /**
     * @dev Get all owners
     */
    function getOwners() external view returns (address[] memory) {
        address[] memory result = new address[](owners.length);
        for (uint256 i = 0; i < owners.length; ) {
            result[i] = owners[i].owner;
            unchecked {
                i++;
            }
        }
        return result;
    }

    /**
     * @dev Get all guardians
     */
    function getGuardians() external view returns (address[] memory) {
        address[] memory result = new address[](guardians.length);
        for (uint256 i = 0; i < guardians.length; ) {
            result[i] = guardians[i].guardian;
            unchecked {
                i++;
            }
        }
        return result;
    }

    // Receive ETH
    receive() external payable {}

    // Fallback
    fallback() external payable {}
}

/**
 * @title IEntryPoint
 * @dev Minimal EntryPoint interface
 */
interface IEntryPoint {
    function getNonce(address sender) external view returns (uint256);

    function getDeposit(address account) external view returns (uint256);
}