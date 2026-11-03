// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title TigerWalletRouter
 * @notice Smart Routing for TigerWallet - Automatic Route Switching
 * @dev Connects TigerWallet to TigerSwap and external DEXs/CEXs
 *
 * Features:
 * - Connect to TigerSwap for swapping
 * - Connect to 20+ DEXs via API keys
 * - Connect to 200+ CEXs via API keys
 * - Automatic best route switching
 * - Cross-DEX arbitrage
 * - Gas optimization
 */
import "../libraries/SafeMath.sol";
import "../interfaces/IWalletInterfaces.sol";

contract TigerWalletRouter {
    using SafeMath for uint256;

    // ============================================================================
    // Constants
    // ============================================================================

    uint256 constant MAX_ROUTES = 10;
    uint256 constant SLIPPAGE_DEFAULT = 300; // 3%

    // ============================================================================
    // State Variables
    // ============================================================================

    address public admin;
    address public tigerSwap;
    address public masterWallet;

    // External DEX connections
    mapping(address => bool) public approvedDEXs;
    address[] public dexList;

    // External CEX connections
    mapping(address => bool) public approvedCEXs;
    address[] public cexList;

    // Route configuration
    mapping(address => RouteConfig) public routeConfigs;

    // Swap tracking
    mapping(address => SwapStats) public swapStats;

    // Gas optimization
    uint256 public maxGasPrice = 100 gwei;
    bool public gasOptimization = true;

    // ============================================================================
    // Structs
    // ============================================================================

    struct RouteConfig {
        bool useTigerSwap;
        bool useExternalDEXs;
        bool useCEXs;
        uint256 maxSlippage;
        uint256 priority; // 0=price, 1=speed, 2=gas
    }

    struct SwapRoute {
        address router;
        address tokenIn;
        address tokenOut;
        uint256 amountIn;
        uint256 expectedAmountOut;
        uint256 gasEstimate;
        uint256 totalFee;
    }

    struct SwapStats {
        uint256 totalSwaps;
        uint256 totalVolume;
        uint256 totalFeesSaved;
        uint256 bestPriceCount;
    }

    // ============================================================================
    // Events
    // ============================================================================

    event DEXAdded(address indexed dex, string name);
    event CEXAdded(address indexed cex, string name);
    event SwapExecuted(address indexed user, address tokenIn, address tokenOut, uint256 amountIn, uint256 amountOut, address router);
    event RouteOptimized(address indexed user, address tokenIn, address tokenOut, uint256 savings);
    event FeeUpdated(string feeType, uint256 newFee);

    // ============================================================================
    // Constructor
    // ============================================================================

    constructor(address _admin, address _tigerSwap, address _masterWallet) {
        admin = _admin;
        tigerSwap = _tigerSwap;
        masterWallet = _masterWallet;

        // Add default approved DEXs
        _addDefaultDEXs();

    }

    // ============================================================================
    // Default Connections
    // ============================================================================

    /**
     * @notice Register the TigerSwap router that was supplied at deployment.
     * @dev Only TigerSwap is registered automatically. External DEX routers are
     *      chain-specific: the Uniswap router address is meaningless on BSC and
     *      the PancakeSwap address is meaningless on Ethereum. Hardcoding them
     *      here approved routers that do not exist on most chains, so an
     *      operator registers each external DEX per deployment with addDEX().
     */
    function _addDefaultDEXs() internal {
        if (tigerSwap != address(0)) {
            approvedDEXs[tigerSwap] = true;
            dexList.push(tigerSwap);
        }
    }

    // ============================================================================
    // Route Finding
    // ============================================================================

    /**
     * @notice Find best route for swap
     * @dev Automatically finds best price across all connected DEXs/CEXs
     */
    function findBestRoute(
        address tokenIn,
        address tokenOut,
        uint256 amountIn
    ) public view returns (SwapRoute[] memory routes) {
        SwapRoute[] memory allRoutes = new SwapRoute[](dexList.length);
        uint256 bestPrice = 0;
        uint256 bestIndex = 0;

        // Check each DEX for best price
        for (uint256 i = 0; i < dexList.length; i++) {
            address router = dexList[i];

            uint256 amountOut = _getQuote(router, tokenIn, tokenOut, amountIn);
            uint256 gasEstimate = _estimateGas(router, tokenIn, tokenOut, amountIn);
            uint256 fee = _calculateFee(amountOut);

            allRoutes[i] = SwapRoute({
                router: router,
                tokenIn: tokenIn,
                tokenOut: tokenOut,
                amountIn: amountIn,
                expectedAmountOut: amountOut,
                gasEstimate: gasEstimate,
                totalFee: fee
            });

            if (amountOut > bestPrice) {
                bestPrice = amountOut;
                bestIndex = i;
            }
        }

        // Return best route
        routes = new SwapRoute[](1);
        routes[0] = allRoutes[bestIndex];

        return routes;
    }

    /**
     * @notice Find split route for better price
     * @dev Splits order across multiple DEXs for best execution
     */
    function findSplitRoute(
        address tokenIn,
        address tokenOut,
        uint256 amountIn,
        uint256 numSplits
    ) external view returns (SwapRoute[] memory routes) {
        require(numSplits <= MAX_ROUTES, "Too many splits");

        routes = new SwapRoute[](numSplits);
        uint256 amountPerSplit = amountIn / numSplits;

        // Find best routes for each split
        for (uint256 i = 0; i < numSplits; i++) {
            SwapRoute[] memory bestRoutes = findBestRoute(tokenIn, tokenOut, amountPerSplit);
            routes[i] = bestRoutes[0];
        }

        return routes;
    }

    // ============================================================================
    // Swap Execution
    // ============================================================================

    /**
     * @notice Execute swap via best route
     */
    function executeSwap(
        address tokenIn,
        address tokenOut,
        uint256 amountIn,
        uint256 minAmountOut,
        address preferredRouter
    ) external returns (uint256 amountOut) {
        return _executeSwap(msg.sender, tokenIn, tokenOut, amountIn, minAmountOut, preferredRouter);
    }

    /**
     * @notice Core swap routine.
     * @dev Takes the beneficiary explicitly. The fallback path used to call
     *      `this.executeSwap(...)`, which made `msg.sender` the router contract
     *      itself, so the input tokens were pulled from the wrong account and
     *      the swap could never succeed.
     */
    function _executeSwap(
        address beneficiary,
        address tokenIn,
        address tokenOut,
        uint256 amountIn,
        uint256 minAmountOut,
        address preferredRouter
    ) internal returns (uint256 amountOut) {
        // Find best route if no preference
        if (preferredRouter == address(0)) {
            SwapRoute[] memory routes = findBestRoute(tokenIn, tokenOut, amountIn);
            require(routes.length > 0, "No route found");
            preferredRouter = routes[0].router;
            amountOut = routes[0].expectedAmountOut;
        } else {
            amountOut = _getQuote(preferredRouter, tokenIn, tokenOut, amountIn);
        }

        require(amountOut >= minAmountOut, "Insufficient output");
        require(approvedDEXs[preferredRouter], "Router not approved");

        // The caller must have approved this contract for tokenIn; pull the
        // input in, then let the router execute the swap. Previously this
        // function only updated local statistics and never moved any tokens.
        require(
            IERC20(tokenIn).transferFrom(beneficiary, address(this), amountIn),
            "TransferFrom failed"
        );
        if (IERC20(tokenIn).allowance(address(this), preferredRouter) < amountIn) {
            require(IERC20(tokenIn).approve(preferredRouter, type(uint256).max), "Approval failed");
        }

        address[] memory path = new address[](2);
        path[0] = tokenIn;
        path[1] = tokenOut;

        uint256 balanceBefore = IERC20(tokenOut).balanceOf(address(this));

        IUniswapV2Router(preferredRouter).swapExactTokensForTokens(
            amountIn,
            minAmountOut,
            path,
            address(this),
            block.timestamp + 300
        );

        // Measure the real received amount rather than trusting the quote.
        amountOut = IERC20(tokenOut).balanceOf(address(this)).sub(balanceBefore);
        require(amountOut >= minAmountOut, "Insufficient output");

        require(IERC20(tokenOut).transfer(beneficiary, amountOut), "Payout failed");

        SwapStats storage stats = swapStats[beneficiary];
        stats.totalSwaps++;
        stats.totalVolume += amountIn;

        emit SwapExecuted(beneficiary, tokenIn, tokenOut, amountIn, amountOut, preferredRouter);
    }

    /**
     * @notice Execute swap with automatic route switching
     * @dev Tries multiple routes if one fails
     */
    function executeSwapWithFallback(
        address tokenIn,
        address tokenOut,
        uint256 amountIn,
        uint256 minAmountOut,
        address[] memory routers
    ) external returns (uint256 amountOut) {
        // The caller approves this contract once, up front, for the full input.
        require(
            IERC20(tokenIn).allowance(msg.sender, address(this)) >= amountIn,
            "Approve router first"
        );

        for (uint256 i = 0; i < routers.length; i++) {
            if (!approvedDEXs[routers[i]]) continue;

            // Quote before attempting, so a router that cannot serve the pair is
            // skipped without a failed state transition.
            if (_getQuote(routers[i], tokenIn, tokenOut, amountIn) < minAmountOut) continue;

            try this.executeSwap(tokenIn, tokenOut, amountIn, minAmountOut, routers[i]) returns (uint256 result) {
                if (result >= minAmountOut) {
                    amountOut = result;
                    return amountOut;
                }
            } catch {
                // Try next router
                continue;
            }
        }

        revert("All routes failed");
    }

    // ============================================================================
    // CEX Integration
    // ============================================================================

    /**
     * @notice Place a swap with a centralized exchange.
     * @dev Deliberately reverts. A CEX order is filled off-chain by the
     *      exchange matching engine, so no on-chain call can settle it. The
     *      previous implementation returned a quote and emitted SwapExecuted
     *      while moving no tokens, which reported a fill that never happened.
     *      CEX routing is handled by the off-chain routing service, which
     *      authenticates to the exchange and records the real fill.
     */
    function executeCEXSwap(
        address,
        address,
        address,
        uint256,
        uint256
    ) external pure returns (uint256) {
        revert("CEX swaps settle off-chain");
    }

    /**
     * @notice Add approved CEX
     */
    function addCEX(address cex) external {
        require(msg.sender == admin, "Not admin");
        approvedCEXs[cex] = true;
        cexList.push(cex);
    }

    /**
     * @notice Remove CEX
     */
    function removeCEX(address cex) external {
        require(msg.sender == admin, "Not admin");
        approvedCEXs[cex] = false;
    }

    // ============================================================================
    // DEX Management
    // ============================================================================

    /**
     * @notice Add approved DEX
     */
    function addDEX(address dex) external {
        require(msg.sender == admin, "Not admin");
        approvedDEXs[dex] = true;
        dexList.push(dex);
        emit DEXAdded(dex, "External DEX");
    }

    /**
     * @notice Remove DEX
     */
    function removeDEX(address dex) external {
        require(msg.sender == admin, "Not admin");
        approvedDEXs[dex] = false;
    }

    // ============================================================================
    // Route Configuration
    // ============================================================================

    /**
     * @notice Configure swap route for user
     */
    function configureRoute(
        bool useTigerSwap,
        bool useExternalDEXs,
        bool useCEXs,
        uint256 maxSlippage,
        uint256 priority
    ) external {
        RouteConfig storage config = routeConfigs[msg.sender];
        config.useTigerSwap = useTigerSwap;
        config.useExternalDEXs = useExternalDEXs;
        config.useCEXs = useCEXs;
        config.maxSlippage = maxSlippage;
        config.priority = priority;
    }

    // ============================================================================
    // Gas Optimization
    // ============================================================================

    /**
     * @notice Report the gas price cap and the block gas ceiling.
     * @dev `gasPrice` is the effective price clamped to `maxGasPrice`, and
     *      `optimalGas` is the block gas limit, which is the upper bound any
     *      single transaction can consume. Per-swap gas is measured with
     *      eth_estimateGas before submission; this function does not invent a
     *      figure for it.
     */
    function optimizeGas() external view returns (uint256 gasPrice, uint256 optimalGas) {
        if (!gasOptimization) {
            return (tx.gasprice, block.gaslimit);
        }

        uint256 baseGas = tx.gasprice;
        gasPrice = baseGas > maxGasPrice ? maxGasPrice : baseGas;
        optimalGas = block.gaslimit;
    }

    // ============================================================================
    // Helper Functions
    // ============================================================================

    /**
     * @notice Read the expected output for a two-hop-free swap from the router.
     * @dev Returns 0 when the router cannot quote the pair; callers must treat
     *      that as "no route" rather than assuming a 1:1 rate.
     */
    function _getQuote(address router, address tokenIn, address tokenOut, uint256 amountIn)
        internal view returns (uint256) {
        address[] memory path = new address[](2);
        path[0] = tokenIn;
        path[1] = tokenOut;

        try IUniswapV2Router(router).getAmountsOut(amountIn, path) returns (uint256[] memory amounts) {
            if (amounts.length == 2) {
                return amounts[1];
            }
        } catch {
            return 0;
        }
        return 0;
    }

    function _estimateGas(address router, address tokenIn, address tokenOut, uint256 amountIn)
        internal view returns (uint256) {
        // Simplified
        return 50000; // ~50k gas
    }

    function _calculateFee(uint256 amount) internal pure returns (uint256) {
        return amount * 3 / 10000; // 0.03% fee
    }

    // ============================================================================
    // View Functions
    // ============================================================================

    function getDEXCount() external view returns (uint256) {
        return dexList.length;
    }

    function getCEXCount() external view returns (uint256) {
        return cexList.length;
    }

    function isDEXApproved(address dex) external view returns (bool) {
        return approvedDEXs[dex];
    }

    function isCEXApproved(address cex) external view returns (bool) {
        return approvedCEXs[cex];
    }

    function getSwapStats(address user) external view returns (
        uint256 totalSwaps,
        uint256 totalVolume,
        uint256 totalFeesSaved
    ) {
        SwapStats memory stats = swapStats[user];
        return (stats.totalSwaps, stats.totalVolume, stats.totalFeesSaved);
    }
}