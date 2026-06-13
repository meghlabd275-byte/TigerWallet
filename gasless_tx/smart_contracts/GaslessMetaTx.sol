// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title GaslessMetaTx
 * @dev EIP-2771 Gasless Transaction Contract
 * @dev Supports meta-transactions with relayers and sponsors
 */
contract GaslessMetaTx {
    // Events
    event MetaTransactionExecuted(
        address indexed user,
        address indexed relayer,
        bytes32 indexed hash,
        uint256 nonce
    );
    event RelayerAdded(address indexed relayer);
    event RelayerRemoved(address indexed relayer);
    event SponsorAdded(address indexed sponsor);
    event SponsorRemoved(address indexed sponsor);
    event GasPolicyUpdated(uint256 gasPrice, uint256 maxGas);
    event WhitelistUpdated(address indexed user, bool allowed);
    event TrustedForwarderUpdated(address indexed forwarder);

    // Constants
    bytes4 internal constant MAGIC_BYTES = 0x5190485e;

    // State
    mapping(address => bool) public relayers;
    mapping(address => bool) public sponsors;
    mapping(address => uint256) public nonces;
    mapping(address => uint256) public deposits;
    mapping(address => mapping(address => uint256)) public sponsorAllowances;
    mapping(address => bool) public whitelisted;
    mapping(address => bool) public trustedForwarders;

    // Gas policy
    uint256 public gasPrice = 30 gwei;
    uint256 public maxGas = 500000;
    uint256 public gasRefundPercent = 110;

    // Domain separator
    bytes32 public DOMAIN_SEPARATOR;

    // Chain ID
    uint256 public chainId;

    // Contract name
    string public constant name = "TigerWallet Gasless";
    string public constant version = "1.0.0";

    /**
     * @dev Constructor
     */
    constructor() {
        chainId = block.chainid;
        _updateDomainSeparator();
    }

    /**
     * @dev Update domain separator
     */
    function _updateDomainSeparator() internal {
        DOMAIN_SEPARATOR = keccak256(
            abi.encode(
                keccak256(
                    "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
                ),
                keccak256(bytes(name)),
                keccak256(bytes(version)),
                chainId,
                address(this)
            )
        );
    }

    /**
     * @dev Verify relayer
     */
    modifier onlyRelayer() {
        require(relayers[msg.sender], "Not relayer");
        _;
    }

    /**
     * @dev Execute meta transaction
     * @param from User address
     * @param to Target address
     * @param value Ether value
     * @param data Call data
     * @param gasLimit Gas limit
     * @param relayer Relayer address
     * @param signature User signature
     */
    function executeMetaTransaction(
        address from,
        address to,
        uint256 value,
        bytes calldata data,
        uint256 gasLimit,
        address relayer,
        bytes calldata signature
    ) external payable onlyRelayer returns (bytes memory) {
        // Verify signature
        require(
            _verifySignature(from, to, value, data, gasLimit, relayer, signature),
            "Invalid signature"
        );

        // Check gas policy
        require(gasLimit <= maxGas, "Gas limit exceeded");

        // Execute call
        (bool success, bytes memory result) = to.call{value: value, gas: gasLimit}(data);

        // Refund relayer
        uint256 gasUsed = gasLimit - gasleft();
        uint256 refundAmount = gasUsed * tx.gasprice * gasRefundPercent / 100;

        if (msg.value > refundAmount) {
            payable(relayer).send(msg.value - refundAmount);
        }

        // Update nonce
        nonces[from]++;

        emit MetaTransactionExecuted(from, relayer, keccak256(data), nonces[from]);

        return result;
    }

    /**
     * @dev Verify signature
     */
    function _verifySignature(
        address from,
        address to,
        uint256 value,
        bytes calldata data,
        uint256 gasLimit,
        address relayer,
        bytes calldata signature
    ) internal view returns (bool) {
        bytes32 domainSeparator = DOMAIN_SEPARATOR;
        
        if (chainId != block.chainid) {
            domainSeparator = keccak256(
                abi.encode(
                    keccak256(
                        "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
                    ),
                    keccak256(bytes(name)),
                    keccak256(bytes(version)),
                    block.chainid,
                    address(this)
                )
            );
        }

        bytes32 hash = keccak256(
            abi.encodePacked(
                "\x19\x01",
                domainSeparator,
                keccak256(
                    abi.encode(
                        keccak256(
                            "MetaTransaction(address from,address to,uint256 value,bytes data,uint256 gasLimit,address relayer,uint256 nonce)"
                        ),
                        from,
                        to,
                        value,
                        keccak256(data),
                        gasLimit,
                        relayer,
                        nonces[from]
                    )
                )
            )
        );

        // Recover signer
        (bytes32 r, bytes32 s, uint8 v) = _splitSignature(signature);
        address signer = ecrecover(hash, v, r, s);

        return signer == from;
    }

    /**
     * @dev Split signature
     */
    function _splitSignature(
        bytes calldata signature
    ) internal pure returns (bytes32 r, bytes32 s, uint8 v) {
        require(signature.length == 65, "Invalid signature");
        
        assembly {
            r := calldataload(signature.offset)
            s := calldataload(add(signature.offset, 32))
            v := byte(0, calldataload(add(signature.offset, 64)))
        }
    }

    /**
     * @dev Add relayer
     */
    function addRelayer(address relayer) external {
        require(relayer != address(0), "Zero address");
        relayers[relayer] = true;
        emit RelayerAdded(relayer);
    }

    /**
     * @dev Remove relayer
     */
    function removeRelayer(address relayer) external {
        relayers[relayer] = false;
        emit RelayerRemoved(relayer);
    }

    /**
     * @dev Add sponsor
     */
    function addSponsor(address sponsor) external {
        require(sponsor != address(0), "Zero address");
        sponsors[sponsor] = true;
        emit SponsorAdded(sponsor);
    }

    /**
     * @dev Remove sponsor
     */
    function removeSponsor(address sponsor) external {
        sponsors[sponsor] = false;
        emit SponsorRemoved(sponsor);
    }

    /**
     * @dev Set sponsor allowance
     */
    function setSponsorAllowance(
        address sponsor,
        address user,
        uint256 allowance
    ) external {
        sponsorAllowances[sponsor][user] = allowance;
    }

    /**
     * @dev Update gas policy
     */
    function setGasPolicy(uint256 _gasPrice, uint256 _maxGas, uint256 _gasRefundPercent) external {
        gasPrice = _gasPrice;
        maxGas = _maxGas;
        gasRefundPercent = _gasRefundPercent;
        emit GasPolicyUpdated(_gasPrice, _maxGas);
    }

    /**
     * @dev Whitelist user
     */
    function whitelistUser(address user, bool allowed) external {
        whitelisted[user] = allowed;
        emit WhitelistUpdated(user, allowed);
    }

    /**
     * @dev Set trusted forwarder
     */
    function setTrustedForwarder(address forwarder, bool allowed) external {
        trustedForwarders[forwarder] = allowed;
        emit TrustedForwarderUpdated(forwarder);
    }

    /**
     * @dev Get nonce
     */
    function getNonce(address user) external view returns (uint256) {
        return nonces[user];
    }

    /**
     * @dev Get deposit
     */
    function getDeposit(address user) external view returns (uint256) {
        return deposits[user];
    }

    /**
     * @dev Is trusted forwarder
     */
    function isTrustedForwarder(address forwarder) external view returns (bool) {
        return trustedForwarders[forwarder];
    }

    receive() external payable {
        deposits[msg.sender] += msg.value;
    }
}