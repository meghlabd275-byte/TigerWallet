// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title TigerWallet
 * @notice User Wallet - Complete Web3 Wallet Functionality
 * @dev Non-custodial wallet with 24-word seed phrase
 *
 * Features:
 * - 24-word BIP39 seed phrase
 * - Password protection
 * - Multi-chain support (EVM + Non-EVM)
 * - Send/Receive native & tokens
 * - Swap (integrated DEX)
 * - Claim airdrops
 * - Join campaigns
 * - Built-in DEX browser
 * - Provide liquidity
 * - Multi-sig transfers
 * - Create tokens
 * - Auto-signed by master wallet
 */
import "../libraries/SafeMath.sol";
import "../interfaces/IWalletInterfaces.sol";

contract TigerWallet {
    using SafeMath for uint256;

    // ============================================================================
    // Constants
    // ============================================================================

    uint256 constant MAX_CHAINS = 100;
    uint256 constant MAX_TOKENS = 10000;
    uint256 constant VERSION = 1;

    // ============================================================================
    // State Variables
    // ============================================================================

    address public masterWallet;
    address public walletFactory;

    // Wallet info
    bytes public encryptedSeed;
    bytes32 public walletHash;
    string public walletName;
    uint256 public createdAt;
    bool public isActive;

    // Security
    bytes32 public passwordHash;
    bool public isPasswordSet;
    bool public twoFactorEnabled;
    address public twoFactorAddress;

    // Chains supported
    mapping(uint256 => bool) public supportedChains;
    uint256[] public chainIds;

    // Balances
    mapping(uint256 => uint256) public nativeBalances;  // chainId -> balance
    mapping(uint256 => mapping(address => uint256)) public tokenBalances;  // chainId -> token -> balance

    // Pending transactions
    mapping(bytes32 => Transaction) public pendingTransactions;
    bytes32[] public pendingTxIds;
    uint256 public pendingTxCount = 0;

    // Transaction history
    Transaction[] public transactionHistory;
    uint256 public txHistoryCount = 0;

    // Auto-sign settings
    bool public autoSignEnabled;
    uint256 public autoSignLimit;
    mapping(address => bool) public allowedTokens;

    // Token factory used by createToken. Zero means the factory has not been
    // wired up yet, and createToken reverts rather than handing back an address
    // that is not a token.
    address public tokenFactory;

    // Multisig
    mapping(bytes32 => MultisigTx) public multisigTxs;
    bytes32[] public multisigTxIds;
    uint256 public requiredSignatures = 1;

    // ============================================================================
    // Structs
    // ============================================================================

    struct Transaction {
        bytes32 id;
        uint256 chainId;
        address from;
        address to;
        address token;
        uint256 amount;
        uint256 fee;
        uint256 timestamp;
        TxType txType;
        TxStatus status;
        bytes data;
    }

    enum TxType {
        Send,
        Receive,
        Swap,
        Liquidity,
        TokenCreate,
        AirdropClaim,
        CampaignJoin,
        Bridge,
        Multisig
    }

    enum TxStatus {
        Pending,
        Signed,
        Executed,
        Failed,
        Cancelled
    }

    struct MultisigTx {
        bytes32 id;
        address[] owners;
        uint256 confirmations;
        uint256 amount;
        address token;
        address to;
        uint256 timestamp;
        bool executed;
    }

    struct WalletConfig {
        string name;
        bool autoSign;
        uint256 autoSignLimit;
        address[] allowedTokens;
        uint256[] allowedChains;
    }

    // ============================================================================
    // Events
    // ============================================================================

    event WalletCreated(address indexed wallet, string name);
    event Deposit(address indexed from, uint256 amount, uint256 chainId);
    event Withdraw(address indexed to, uint256 amount, uint256 chainId, bytes32 txHash);
    event SwapExecuted(address indexed tokenIn, address indexed tokenOut, uint256 amountIn, uint256 amountOut);
    event LiquidityAdded(address indexed tokenA, address indexed tokenB, uint256 amountA, uint256 amountB);
    event TokenCreated(address indexed token, string name, string symbol, uint256 supply);
    event AirdropClaimed(address indexed airdrop, uint256 amount);
    event MultisigCreated(bytes32 indexed txId, address indexed creator);
    event MultisigExecuted(bytes32 indexed txId);
    event ChainAdded(uint256 indexed chainId);
    event ConfigUpdated(string field);

    // ============================================================================
    // Modifiers
    // ============================================================================

    modifier onlyMasterWallet() {
        require(msg.sender == masterWallet, "Not master wallet");
        _;
    }

    modifier onlyWithPassword() {
        require(isPasswordSet, "Password not set");
        _;
    }

    // ============================================================================
    // Constructor
    // ============================================================================

    constructor(
        address _masterWallet,
        address _walletFactory,
        bytes memory _encryptedSeed,
        bytes32 _walletHash,
        string memory _name
    ) {
        masterWallet = _masterWallet;
        walletFactory = _walletFactory;
        encryptedSeed = _encryptedSeed;
        walletHash = _walletHash;
        walletName = _name;
        createdAt = block.timestamp;
        isActive = true;

        emit WalletCreated(address(this), _name);
    }

    // ============================================================================
    // Security
    // ============================================================================

    /**
     * @notice Set wallet password
     */
    function setPassword(string memory password) external {
        require(msg.sender == masterWallet || tx.origin == address(this), "Not authorized");
        passwordHash = keccak256(abi.encodePacked(password));
        isPasswordSet = true;
        emit ConfigUpdated("password");
    }

    /**
     * @notice Verify password
     */
    function verifyPassword(string memory password) external view returns (bool) {
        return keccak256(abi.encodePacked(password)) == passwordHash;
    }

    /**
     * @notice Enable two-factor authentication
     */
    function enableTwoFactor(address _twoFactorAddress) external {
        require(msg.sender == masterWallet, "Not master wallet");
        twoFactorEnabled = true;
        twoFactorAddress = _twoFactorAddress;
        emit ConfigUpdated("twoFactor");
    }

    /**
     * @notice Disable two-factor authentication
     */
    function disableTwoFactor() external {
        require(msg.sender == masterWallet, "Not master wallet");
        twoFactorEnabled = false;
        twoFactorAddress = address(0);
    }

    // ============================================================================
    // Chain Management
    // ============================================================================

    /**
     * @notice Add supported chain
     */
    function addChain(uint256 chainId) external onlyMasterWallet {
        require(!supportedChains[chainId], "Chain already supported");
        supportedChains[chainId] = true;
        chainIds.push(chainId);
        emit ChainAdded(chainId);
    }

    /**
     * @notice Remove supported chain
     */
    function removeChain(uint256 chainId) external onlyMasterWallet {
        supportedChains[chainId] = false;
    }

    /**
     * @notice Check if chain is supported
     */
    function isChainSupported(uint256 chainId) external view returns (bool) {
        return supportedChains[chainId];
    }

    // ============================================================================
    // Balance Management
    // ============================================================================

    /**
     * @notice Deposit native token
     */
    function depositNative(uint256 chainId) external payable {
        require(supportedChains[chainId], "Chain not supported");
        nativeBalances[chainId] = nativeBalances[chainId].add(msg.value);
        emit Deposit(msg.sender, msg.value, chainId);

        // Record transaction
        _recordTransaction(chainId, address(0), address(this), address(0), msg.value, TxType.Receive);
    }

    /**
     * @notice Deposit token
     */
    function depositToken(uint256 chainId, address token, uint256 amount) external {
        require(supportedChains[chainId], "Chain not supported");
        require(token != address(0), "Invalid token");

        // Transfer token from user (would use IERC20 in production)
        tokenBalances[chainId][token] = tokenBalances[chainId][token].add(amount);
        emit Deposit(msg.sender, amount, chainId);

        _recordTransaction(chainId, address(0), address(this), token, amount, TxType.Receive);
    }

    /**
     * @notice Withdraw native token
     */
    function withdrawNative(uint256 chainId, address to, uint256 amount) external onlyMasterWallet {
        require(supportedChains[chainId], "Chain not supported");
        require(nativeBalances[chainId] >= amount, "Insufficient balance");

        nativeBalances[chainId] = nativeBalances[chainId].sub(amount);

        // Transfer to recipient
        (bool success, ) = to.call{value: amount}("");
        require(success, "Transfer failed");

        bytes32 txHash = keccak256(abi.encodePacked(block.timestamp, to, amount));
        emit Withdraw(to, amount, chainId, txHash);

        _recordTransaction(chainId, address(this), to, address(0), amount, TxType.Send);
    }

    /**
     * @notice Withdraw token
     */
    function withdrawToken(uint256 chainId, address to, address token, uint256 amount) external onlyMasterWallet {
        require(supportedChains[chainId], "Chain not supported");
        require(tokenBalances[chainId][token] >= amount, "Insufficient balance");

        tokenBalances[chainId][token] = tokenBalances[chainId][token].sub(amount);

        // Transfer token (would use IERC20 in production)

        _recordTransaction(chainId, address(this), to, token, amount, TxType.Send);
    }

    /**
     * @notice Get native balance
     */
    function getNativeBalance(uint256 chainId) external view returns (uint256) {
        return nativeBalances[chainId];
    }

    /**
     * @notice Get token balance
     */
    function getTokenBalance(uint256 chainId, address token) external view returns (uint256) {
        return tokenBalances[chainId][token];
    }

    // ============================================================================
    // Swap (Integrated DEX)
    // ============================================================================

    /**
     * @notice Swap tokens (integrated DEX)
     */
    function swap(
        uint256 chainId,
        address tokenIn,
        address tokenOut,
        uint256 amountIn,
        uint256 minAmountOut,
        address router
    ) external onlyMasterWallet returns (uint256 amountOut) {
        require(supportedChains[chainId], "Chain not supported");
        require(router != address(0), "Router not configured");
        require(tokenBalances[chainId][tokenIn] >= amountIn, "Insufficient balance");

        // Deduct input before the external call (checks-effects-interactions).
        tokenBalances[chainId][tokenIn] = tokenBalances[chainId][tokenIn].sub(amountIn);

        address[] memory path = new address[](2);
        path[0] = tokenIn;
        path[1] = tokenOut;

        // Quote first so an unproductive route reverts before any tokens move.
        uint256[] memory quoted = IUniswapV2Router(router).getAmountsOut(amountIn, path);
        require(quoted.length == 2, "Invalid quote");
        require(quoted[1] >= minAmountOut, "Slippage exceeded");

        uint256 balanceBefore = IERC20(tokenOut).balanceOf(address(this));

        _ensureAllowance(tokenIn, router, amountIn);

        IUniswapV2Router(router).swapExactTokensForTokens(
            amountIn,
            minAmountOut,
            path,
            address(this),
            block.timestamp + 300
        );

        // Measure the real output instead of trusting the quote: a
        // fee-on-transfer or rebasing token makes the two differ.
        amountOut = IERC20(tokenOut).balanceOf(address(this)).sub(balanceBefore);
        require(amountOut >= minAmountOut, "Insufficient output");

        tokenBalances[chainId][tokenOut] = tokenBalances[chainId][tokenOut].add(amountOut);

        emit SwapExecuted(tokenIn, tokenOut, amountIn, amountOut);

        _recordTransaction(chainId, address(this), router, tokenIn, amountIn, TxType.Swap);
    }

    /**
     * @notice Approve `spender` for at least `amount` of `token`.
     * @dev Tops the allowance up only when the current one is short, so repeated
     *      swaps do not emit a redundant approval each time.
     */
    function _ensureAllowance(address token, address spender, uint256 amount) internal {
        if (IERC20(token).allowance(address(this), spender) < amount) {
            require(IERC20(token).approve(spender, type(uint256).max), "Approval failed");
        }
    }

    // ============================================================================
    // Liquidity
    // ============================================================================

    /**
     * @notice Add liquidity to DEX
     */
    function addLiquidity(
        uint256 chainId,
        address tokenA,
        address tokenB,
        uint256 amountADesired,
        uint256 amountBDesired,
        address router
    ) external onlyMasterWallet returns (uint256 amountA, uint256 amountB) {
        require(supportedChains[chainId], "Chain not supported");
        require(router != address(0), "Router not configured");
        require(tokenBalances[chainId][tokenA] >= amountADesired, "Insufficient balance A");
        require(tokenBalances[chainId][tokenB] >= amountBDesired, "Insufficient balance B");

        // Deduct before the external call (checks-effects-interactions).
        tokenBalances[chainId][tokenA] = tokenBalances[chainId][tokenA].sub(amountADesired);
        tokenBalances[chainId][tokenB] = tokenBalances[chainId][tokenB].sub(amountBDesired);

        _ensureAllowance(tokenA, router, amountADesired);
        _ensureAllowance(tokenB, router, amountBDesired);

        // The router decides the final amounts (it rebalances against pool
        // ratios), so record what it actually consumed.
        (amountA, amountB, ) = IUniswapV2Router(router).addLiquidity(
            tokenA,
            tokenB,
            amountADesired,
            amountBDesired,
            0,
            0,
            address(this),
            block.timestamp + 300
        );
        require(amountA > 0 && amountB > 0, "No liquidity minted");

        // Refund whatever the router did not take.
        if (amountA < amountADesired) {
            tokenBalances[chainId][tokenA] = tokenBalances[chainId][tokenA].add(amountADesired.sub(amountA));
        }
        if (amountB < amountBDesired) {
            tokenBalances[chainId][tokenB] = tokenBalances[chainId][tokenB].add(amountBDesired.sub(amountB));
        }

        emit LiquidityAdded(tokenA, tokenB, amountA, amountB);

        _recordTransaction(chainId, address(this), router, tokenA, amountA, TxType.Liquidity);
    }

    // ============================================================================
    // Token Creation
    // ============================================================================

    /**
     * @notice Create new token
     */
    function createToken(
        string memory name,
        string memory symbol,
        uint256 totalSupply,
        uint8 decimals,
        uint256 chainId
    ) external onlyMasterWallet returns (address token) {
        require(supportedChains[chainId], "Chain not supported");
        require(tokenFactory != address(0), "Token factory not configured");
        require(totalSupply > 0, "Supply must be positive");

        // Deploy a real ERC-20 through the factory. Previously this returned
        // address(this), recording a supply against the wallet contract itself
        // and handing callers an address that is not a token.
        token = ITigerTokenFactory(tokenFactory).createToken(name, symbol, totalSupply, decimals);
        require(token != address(0), "Factory returned zero address");

        tokenBalances[chainId][token] = totalSupply;

        emit TokenCreated(token, name, symbol, totalSupply);
    }

    // ============================================================================
    // Airdrop & Campaigns
    // ============================================================================

    /**
     * @notice Claim airdrop
     */
    function claimAirdrop(address airdropContract, uint256 chainId) external onlyMasterWallet returns (uint256 amount) {
        require(supportedChains[chainId], "Chain not supported");
        require(airdropContract != address(0), "Airdrop contract not set");

        // Measure the real claim instead of crediting a fixed amount. The
        // previous implementation added a hardcoded 1000e18 on every call,
        // inflating the balance with tokens that were never received.
        uint256 balanceBefore = address(this).balance;
        (bool ok, ) = airdropContract.call(abi.encodeWithSignature("claim(address)", address(this)));
        require(ok, "Airdrop claim failed");
        amount = address(this).balance.sub(balanceBefore);
        require(amount > 0, "Airdrop paid nothing");

        nativeBalances[chainId] = nativeBalances[chainId].add(amount);

        emit AirdropClaimed(airdropContract, amount);

        _recordTransaction(chainId, airdropContract, address(this), address(0), amount, TxType.AirdropClaim);
    }

    /**
     * @notice Join campaign
     */
    function joinCampaign(address campaignContract, bytes memory data) external onlyMasterWallet {
        require(campaignContract != address(0), "Campaign contract not set");

        // Forward the caller-supplied payload to the campaign contract. The
        // previous version only wrote a local history row and never contacted
        // the campaign, so the join silently did nothing on-chain.
        (bool ok, ) = campaignContract.call(data);
        require(ok, "Campaign join failed");

        _recordTransaction(1, address(this), campaignContract, address(0), 0, TxType.CampaignJoin);
    }

    // ============================================================================
    // Multisig
    // ============================================================================

    /**
     * @notice Create multisig transaction
     */
    function createMultisigTx(
        address[] memory owners,
        address to,
        address token,
        uint256 amount,
        uint256 _requiredSignatures
    ) external onlyMasterWallet returns (bytes32 txId) {
        txId = keccak256(abi.encodePacked(block.timestamp, owners, to, amount));

        MultisigTx storage tx = multisigTxs[txId];
        tx.id = txId;
        tx.owners = owners;
        tx.to = to;
        tx.amount = amount;
        tx.token = token;
        tx.timestamp = block.timestamp;
        tx.executed = false;
        tx.confirmations = 0;

        multisigTxIds.push(txId);

        emit MultisigCreated(txId, msg.sender);
    }

    /**
     * @notice Confirm multisig transaction
     */
    function confirmMultisig(bytes32 txId) external {
        MultisigTx storage tx = multisigTxs[txId];
        require(!tx.executed, "Already executed");

        tx.confirmations++;

        if (tx.confirmations >= tx.owners.length) {
            _executeMultisig(txId);
        }
    }

    /**
     * @notice Execute multisig transaction
     */
    function _executeMultisig(bytes32 txId) internal {
        MultisigTx storage tx = multisigTxs[txId];
        require(!tx.executed, "Already executed");

        tx.executed = true;

        if (tx.token == address(0)) {
            (bool success, ) = tx.to.call{value: tx.amount}("");
            require(success, "Transfer failed");
        } else {
            tokenBalances[1][tx.token] = tokenBalances[1][tx.token].sub(tx.amount);
        }

        emit MultisigExecuted(txId);
    }

    // ============================================================================
    // Transaction History
    // ============================================================================

    function _recordTransaction(
        uint256 chainId,
        address from,
        address to,
        address token,
        uint256 amount,
        TxType txType
    ) internal {
        bytes32 txId = keccak256(abi.encodePacked(block.timestamp, txHistoryCount));

        Transaction memory tx = Transaction({
            id: txId,
            chainId: chainId,
            from: from,
            to: to,
            token: token,
            amount: amount,
            fee: 0,
            timestamp: block.timestamp,
            txType: txType,
            status: TxStatus.Executed,
            data: ""
        });

        transactionHistory.push(tx);
        txHistoryCount++;
    }

    function getTransactionHistory(uint256 start, uint256 count) external view returns (Transaction[] memory) {
        Transaction[] memory result = new Transaction[](count);
        for (uint256 i = 0; i < count; i++) {
            result[i] = transactionHistory[start + i];
        }
        return result;
    }

    // ============================================================================
    // Auto-Sign
    // ============================================================================

    /**
     * @notice Enable auto-sign
     */
    function enableAutoSign(uint256 _autoSignLimit) external onlyMasterWallet {
        autoSignEnabled = true;
        autoSignLimit = _autoSignLimit;
    }

    /**
     * @notice Disable auto-sign
     */
    function disableAutoSign() external onlyMasterWallet {
        autoSignEnabled = false;
    }

    // ============================================================================
    // Admin
    // ============================================================================

    function pause() external onlyMasterWallet {
        isActive = false;
    }

    function unpause() external onlyMasterWallet {
        isActive = true;
    }

    function setMasterWallet(address _masterWallet) external {
        require(msg.sender == masterWallet, "Not master wallet");
        masterWallet = _masterWallet;
    }

    /**
     * @notice Point createToken at the deployed token factory.
     * @dev Owner-only. Passing the zero address disables token creation again.
     */
    function setTokenFactory(address _factory) external onlyMasterWallet {
        tokenFactory = _factory;
        emit ConfigUpdated("tokenFactory");
    }
}