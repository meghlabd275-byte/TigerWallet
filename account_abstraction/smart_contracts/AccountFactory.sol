// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title AccountFactory
 * @dev ERC-4337 Account Factory Contract
 * @dev Creates smart accounts with deterministic addresses
 */
contract AccountFactory {
    // Event for account creation
    event AccountCreated(
        address indexed account,
        address indexed owner,
        uint256 salt
    );

    // Event for account initialization
    event AccountInitialized(
        address indexed account,
        address indexed entryPoint
    );

    // Implementation address
    address public implementation;

    // Entry point address
    address public entryPoint;

    // Account mapping
    mapping(address => bool) public isAccount;

    // Account bytecode hash (for CREATE2)
    bytes32 public accountBytecodeHash;

    // Minimum creation gas
    uint256 public constant MIN_CREATE_GAS = 100000;

    // Revert reasons
    string internal constant REVERT_REASON_ZERO_ADDRESS = "Zero address";
    string internal constant REVERT_REASON_INVALID_ENTRY_POINT = "Invalid entry point";
    string internal constant REVERT_REASON_DEPLOYMENT_FAILED = "Deployment failed";
    string internal constant REVERT_REASON_ALREADY_EXISTS = "Account already exists";

    /**
     * @dev Constructor
     * @param _implementation Implementation contract address
     * @param _entryPoint Entry point address
     */
    constructor(address _implementation, address _entryPoint) {
        require(_implementation != address(0), REVERT_REASON_ZERO_ADDRESS);
        require(_entryPoint != address(0), REVERT_REASON_ZERO_ADDRESS);

        implementation = _implementation;
        entryPoint = _entryPoint;

        // Compute bytecode hash for CREATE2
        accountBytecodeHash = keccak256(
            type(SmartAccount).creationCode
        );
    }

    /**
     * @dev Create account with salt
     * @param owner Initial owner address
     * @param salt Salt for deterministic address
     * @return account The created account address
     */
    function createAccount(
        address owner,
        uint256 salt
    ) external returns (address account) {
        bytes32 saltBytes = keccak256(abi.encode(owner, salt));
        
        account = _deployAccount(
            type(SmartAccount).creationCode,
            saltBytes,
            owner
        );
        
        require(!isAccount[account], REVERT_REASON_ALREADY_EXISTS);
        isAccount[account] = true;
        
        emit AccountCreated(account, owner, salt);
    }

    /**
     * @dev Get account address before creation
     * @param owner Owner address
     * @param salt Salt
     * @return The predicted account address
     */
    function getAccountAddress(
        address owner,
        uint256 salt
    ) public view returns (address) {
        bytes32 saltBytes = keccak256(abi.encode(owner, salt));
        
        return
            address(
                uint160(
                    uint256(
                        keccak256(
                            abi.encodePacked(
                                bytes1(0xff),
                                address(this),
                                saltBytes,
                                accountBytecodeHash
                            )
                        )
                    )
                )
            );
    }

    /**
     * @dev Internal deployment function
     */
    function _deployAccount(
        bytes memory creationCode,
        bytes32 salt,
        address owner
    ) internal returns (address) {
        bytes memory initCode = abi.encodePacked(
            creationCode,
            abi.encode(owner, entryPoint)
        );
        
        address account;
        assembly {
            account := create2(
                0,
                add(initCode, 0x20),
                mload(initCode),
                salt
            )
            
            if iszero(account) {
                revert(add(initCode, 0x20), mload(initCode))
            }
        }
        
        return account;
    }

    /**
     * @dev Create account with initialization
     * @param owner Owner address
     * @param salt Salt
     * @param owners Additional owners
     * @param threshold Required threshold
     * @return account Created account address
     */
    function createAccountWithInit(
        address owner,
        uint256 salt,
        address[] memory owners,
        uint256 threshold
    ) external returns (address account) {
        // Deploy account
        account = createAccount(owner, salt);
        
        // Initialize account
        SmartAccount(account).initialize(
            owners,
            threshold,
            entryPoint
        );
        
        emit AccountInitialized(account, entryPoint);
    }

    /**
     * @dev Batch create accounts
     * @param owners Array of owners
     * @param salts Array of salts
     * @return accounts Array of created account addresses
     */
    function batchCreate(
        address[] memory owners,
        uint256[] memory salts
    ) external returns (address[] memory accounts) {
        require(owners.length == salts.length, "Length mismatch");
        
        accounts = new address[](owners.length);
        
        for (uint256 i = 0; i < owners.length; ) {
            accounts[i] = createAccount(owners[i], salts[i]);
            unchecked {
                i++;
            }
        }
    }

    /**
     * @dev Get all deployed accounts for owner
     * @param owner Owner address
     * @return Array of account addresses
     */
    function getAccounts(
        address owner
    ) external view returns (address[] memory) {
        // This would need indexing in production
        // Simplified for demonstration
        address[] memory result = new address[](1);
        result[0] = getAccountAddress(owner, 0);
        return result;
    }
}

/**
 * @title SmartAccount
 * @dev Minimal SmartAccount interface for factory
 */
interface SmartAccount {
    function initialize(
        address[] memory owners,
        uint256 threshold,
        address entryPoint
    ) external;
}