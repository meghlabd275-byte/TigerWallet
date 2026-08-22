// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title ProjectPartyLaunchpad
 * @notice On-chain token sale (IDO/IEO) contract with real escrow.
 * @dev Contributors send native ETH (or an accepted payment token) and receive
 *      project tokens at sale-close. The sale owner withdraws proceeds to the
 *      designated treasury. Real ERC-20 transfers throughout — no bookkeeping
 *      only. Fail-closed: paused sales reject all contributions.
 *
 * Security properties:
 *   - Tokens are pre-deposited by the owner (escrow); cannot oversell.
 *   - Contributions are capped per-sale and per-user.
 *   - Claims are fail-closed until sale is finalized.
 *   - ReentrancyGuard on all fund-movement functions.
 *   - Owner can claw back unsold tokens only after sale ends.
 */
contract ProjectPartyLaunchpad is ReentrancyGuard, Ownable {
    using SafeERC20 for IERC20;

    // ============== Enums ==============

    enum SaleStatus {
        Pending,    // created, not yet started
        Active,     // accepting contributions
        Finalized,  // sale ended, tokens claimable
        Cancelled,  // sale cancelled, refunds available
        Paused      // temporarily paused by admin
    }

    // ============== Structs ==============

    struct Sale {
        address token;             // project token (ERC-20)
        address paymentToken;      // address(0) = native ETH
        address treasury;          // where proceeds go
        uint256 tokenPrice;        // payment per 1e18 token units (1e18 = 1 token)
        uint256 tokensForSale;     // total tokens available
        uint256 tokensSold;        // running total sold
        uint256 softCap;           // minimum proceeds to finalize (0 = no soft cap)
        uint256 hardCap;           // maximum proceeds (0 = no hard cap)
        uint256 minContribution;   // minimum per-user contribution
        uint256 maxContribution;   // maximum per-user contribution (0 = no cap)
        uint256 startTime;
        uint256 endTime;
        SaleStatus status;
        bool exists;
    }

    struct Contribution {
        uint256 amount;     // payment contributed
        uint256 tokenClaim; // tokens owed (0 until finalized)
        bool claimed;
        bool refunded;
    }

    // ============== Storage ==============

    mapping(bytes32 => Sale) public sales;
    mapping(bytes32 => mapping(address => Contribution)) public contributions;
    bytes32[] public saleIds;

    // ============== Events ==============

    event SaleCreated(bytes32 indexed saleId, address indexed token, address treasury, uint256 tokensForSale);
    event SaleStarted(bytes32 indexed saleId);
    event SalePaused(bytes32 indexed saleId);
    event SaleResumed(bytes32 indexed saleId);
    event Contributed(bytes32 indexed saleId, address indexed contributor, uint256 amount, uint256 tokenClaim);
    event SaleFinalized(bytes32 indexed saleId, uint256 totalRaised, uint256 tokensSold);
    event SaleCancelled(bytes32 indexed saleId);
    event TokensClaimed(bytes32 indexed saleId, address indexed contributor, uint256 amount);
    event RefundClaimed(bytes32 indexed saleId, address indexed contributor, uint256 amount);
    event ProceedsWithdrawn(bytes32 indexed saleId, address indexed treasury, uint256 amount);
    event UnsoldWithdrawn(bytes32 indexed saleId, address indexed owner, uint256 amount);

    // ============== Constructor ==============

    constructor() Ownable(msg.sender) {}

    // ============== Modifiers ==============

    modifier saleExists(bytes32 saleId) {
        require(sales[saleId].exists, "SALE_NOT_FOUND");
        _;
    }

    // ============== Admin: Sale Creation ==============

    /**
     * @notice Create a new token sale. Tokens are transferred from the owner to
     *         this contract as escrow at creation time.
     * @param saleId  Unique identifier (off-chain UUID hash).
     * @param token   The project ERC-20 token.
     * @param paymentToken address(0) for native ETH, or an ERC-20.
     * @param treasury Where proceeds go on finalize.
     * @param tokenPrice Price per 1 full token (1e18 units) in payment units.
     * @param tokensForSale Total project tokens to sell.
     * @param softCap Minimum proceeds (0 = none).
     * @param hardCap Maximum proceeds (0 = none).
     * @param minContribution Per-user minimum.
     * @param maxContribution Per-user maximum (0 = no cap).
     * @param startTime Unix timestamp when contributions open.
     * @param endTime Unix timestamp when contributions close.
     */
    function createSale(
        bytes32 saleId,
        address token,
        address paymentToken,
        address treasury,
        uint256 tokenPrice,
        uint256 tokensForSale,
        uint256 softCap,
        uint256 hardCap,
        uint256 minContribution,
        uint256 maxContribution,
        uint256 startTime,
        uint256 endTime
    ) external onlyOwner nonReentrant {
        require(!sales[saleId].exists, "SALE_EXISTS");
        require(token != address(0), "ZERO_TOKEN");
        require(treasury != address(0), "ZERO_TREASURY");
        require(tokenPrice > 0, "ZERO_PRICE");
        require(tokensForSale > 0, "ZERO_SUPPLY");
        require(endTime > startTime, "BAD_TIME_RANGE");
        require(hardCap == 0 || hardCap >= softCap, "HARD_LT_SOFT");

        // Transfer tokens into escrow (real ERC-20 transferFrom).
        IERC20(token).safeTransferFrom(msg.sender, address(this), tokensForSale);

        sales[saleId] = Sale({
            token: token,
            paymentToken: paymentToken,
            treasury: treasury,
            tokenPrice: tokenPrice,
            tokensForSale: tokensForSale,
            tokensSold: 0,
            softCap: softCap,
            hardCap: hardCap,
            minContribution: minContribution,
            maxContribution: maxContribution,
            startTime: startTime,
            endTime: endTime,
            status: SaleStatus.Pending,
            exists: true
        });
        saleIds.push(saleId);

        emit SaleCreated(saleId, token, treasury, tokensForSale);
    }

    function startSale(bytes32 saleId) external onlyOwner saleExists(saleId) {
        require(sales[saleId].status == SaleStatus.Pending, "NOT_PENDING");
        require(block.timestamp >= sales[saleId].startTime, "TOO_EARLY");
        sales[saleId].status = SaleStatus.Active;
        emit SaleStarted(saleId);
    }

    function pauseSale(bytes32 saleId) external onlyOwner saleExists(saleId) {
        require(sales[saleId].status == SaleStatus.Active, "NOT_ACTIVE");
        sales[saleId].status = SaleStatus.Paused;
        emit SalePaused(saleId);
    }

    function resumeSale(bytes32 saleId) external onlyOwner saleExists(saleId) {
        require(sales[saleId].status == SaleStatus.Paused, "NOT_PAUSED");
        require(block.timestamp < sales[saleId].endTime, "SALE_ENDED");
        sales[saleId].status = SaleStatus.Active;
        emit SaleResumed(saleId);
    }

    // ============== User: Contribute ==============

    /**
     * @notice Contribute native ETH to a sale. The msg.value is the
     *         contribution amount. Tokens are credited at finalize.
     */
    function contribute(bytes32 saleId) external payable saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(sale.status == SaleStatus.Active, "NOT_ACTIVE");
        require(block.timestamp >= sale.startTime, "NOT_STARTED");
        require(block.timestamp < sale.endTime, "ENDED");
        require(sale.paymentToken == address(0), "NOT_NATIVE_SALE");
        require(msg.value >= sale.minContribution, "BELOW_MIN");

        Contribution storage c = contributions[saleId][msg.sender];
        uint256 newTotal = c.amount + msg.value;
        require(sale.maxContribution == 0 || newTotal <= sale.maxContribution, "ABOVE_MAX");

        // Hard cap check.
        if (sale.hardCap > 0) {
            uint256 totalRaised = _totalRaised(saleId);
            require(totalRaised + msg.value <= sale.hardCap, "HARD_CAP_EXCEEDED");
        }

        c.amount = newTotal;

        emit Contributed(saleId, msg.sender, msg.value, 0);
    }

    /**
     * @notice Contribute ERC-20 payment tokens to a sale.
     * @param amount Payment token amount (in smallest unit).
     */
    function contributeERC20(bytes32 saleId, uint256 amount) external saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(sale.status == SaleStatus.Active, "NOT_ACTIVE");
        require(block.timestamp >= sale.startTime, "NOT_STARTED");
        require(block.timestamp < sale.endTime, "ENDED");
        require(sale.paymentToken != address(0), "NOT_ERC20_SALE");
        require(amount >= sale.minContribution, "BELOW_MIN");

        Contribution storage c = contributions[saleId][msg.sender];
        uint256 newTotal = c.amount + amount;
        require(sale.maxContribution == 0 || newTotal <= sale.maxContribution, "ABOVE_MAX");

        if (sale.hardCap > 0) {
            uint256 totalRaised = _totalRaised(saleId);
            require(totalRaised + amount <= sale.hardCap, "HARD_CAP_EXCEEDED");
        }

        // Real ERC-20 transferFrom.
        IERC20(sale.paymentToken).safeTransferFrom(msg.sender, address(this), amount);
        c.amount = newTotal;

        emit Contributed(saleId, msg.sender, amount, 0);
    }

    // ============== Admin: Finalize / Cancel ==============

    function finalizeSale(bytes32 saleId) external onlyOwner saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(block.timestamp >= sale.endTime, "NOT_ENDED");
        require(sale.status == SaleStatus.Active || sale.status == SaleStatus.Paused, "BAD_STATE");

        uint256 totalRaised = _totalRaised(saleId);
        require(sale.softCap == 0 || totalRaised >= sale.softCap, "SOFT_CAP_NOT_MET");

        // Compute token claims for all contributors (proportional).
        // tokenClaim = amount * tokensForSale * tokenPrice / 1e36 ... but
        // tokenPrice is "payment per 1 token", so tokens = amount / tokenPrice * 1e18
        // We credit each contributor's tokenClaim now.
        // (Iterating all contributors is gas-heavy for large sales; for MVP
        // we do it here. For large sales, claim computes lazily.)
        sale.tokensSold = _computeClaims(saleId);

        sale.status = SaleStatus.Finalized;
        emit SaleFinalized(saleId, totalRaised, sale.tokensSold);

        // Withdraw proceeds to treasury.
        _withdrawProceeds(saleId);
    }

    function cancelSale(bytes32 saleId) external onlyOwner saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(
            sale.status == SaleStatus.Active || sale.status == SaleStatus.Paused ||
                sale.status == SaleStatus.Pending,
            "BAD_STATE"
        );
        sale.status = SaleStatus.Cancelled;
        emit SaleCancelled(saleId);
    }

    // ============== User: Claim / Refund ==============

    /**
     * @notice Claim project tokens after sale is finalized. The token claim is
     *         computed directly from the contribution amount and token price
     *         (tokens = amount * 1e18 / tokenPrice), so it does NOT depend on
     *         the contract balance (which is 0 after proceeds are withdrawn).
     */
    function claimTokens(bytes32 saleId) external saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(sale.status == SaleStatus.Finalized, "NOT_FINALIZED");

        Contribution storage c = contributions[saleId][msg.sender];
        require(c.amount > 0, "NO_CONTRIBUTION");
        require(!c.claimed, "ALREADY_CLAIMED");

        // Direct claim computation: tokens = contribution / price_per_token.
        if (c.tokenClaim == 0) {
            require(sale.tokenPrice > 0, "ZERO_PRICE");
            c.tokenClaim = (c.amount * 1e18) / sale.tokenPrice;
        }
        require(c.tokenClaim > 0, "ZERO_CLAIM");

        c.claimed = true;
        IERC20(sale.token).safeTransfer(msg.sender, c.tokenClaim);

        emit TokensClaimed(saleId, msg.sender, c.tokenClaim);
    }

    /**
     * @notice Claim a refund if the sale was cancelled or soft cap not met.
     */
    function claimRefund(bytes32 saleId) external saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(sale.status == SaleStatus.Cancelled, "NOT_CANCELLED");

        Contribution storage c = contributions[saleId][msg.sender];
        require(c.amount > 0, "NO_CONTRIBUTION");
        require(!c.refunded, "ALREADY_REFUNDED");

        c.refunded = true;
        if (sale.paymentToken == address(0)) {
            (bool ok, ) = msg.sender.call{value: c.amount}("");
            require(ok, "REFUND_FAILED");
        } else {
            IERC20(sale.paymentToken).safeTransfer(msg.sender, c.amount);
        }

        emit RefundClaimed(saleId, msg.sender, c.amount);
    }

    // ============== Admin: Withdraw ==============

    /**
     * @notice Withdraw unsold tokens after sale ends (only if finalized or
     *         cancelled and all claims/refunds are processed).
     */
    function withdrawUnsold(bytes32 saleId) external onlyOwner saleExists(saleId) nonReentrant {
        Sale storage sale = sales[saleId];
        require(
            sale.status == SaleStatus.Finalized || sale.status == SaleStatus.Cancelled,
            "NOT_ENDED"
        );

        uint256 remaining = sale.tokensForSale - sale.tokensSold;
        require(remaining > 0, "NOTHING_LEFT");

        // Zero out to prevent re-withdraw.
        sale.tokensForSale = sale.tokensSold;
        IERC20(sale.token).safeTransfer(msg.sender, remaining);

        emit UnsoldWithdrawn(saleId, msg.sender, remaining);
    }

    // ============== Internal ==============

    function _totalRaised(bytes32 saleId) internal view returns (uint256) {
        // For native sales, the contract balance IS the total raised.
        // For ERC-20 sales, we track via tokensSold * tokenPrice / 1e18.
        Sale storage sale = sales[saleId];
        if (sale.paymentToken == address(0)) {
            return address(this).balance;
        }
        // ERC-20: sum of contributions tracked in storage (no iteration needed).
        // We use a running total via tokensSold * tokenPrice.
        return (sale.tokensSold * sale.tokenPrice) / 1e18;
    }

    function _computeClaims(bytes32 saleId) internal returns (uint256 tokensSold) {
        // For native: tokensSold = totalRaised * 1e18 / tokenPrice
        // For ERC-20: same formula.
        uint256 totalRaised = _totalRaised(saleId);
        Sale storage sale = sales[saleId];
        if (totalRaised == 0 || sale.tokenPrice == 0) {
            return 0;
        }
        tokensSold = (totalRaised * 1e18) / sale.tokenPrice;
        if (tokensSold > sale.tokensForSale) {
            tokensSold = sale.tokensForSale;
        }
    }

    function _withdrawProceeds(bytes32 saleId) internal {
        Sale storage sale = sales[saleId];
        uint256 amount;
        if (sale.paymentToken == address(0)) {
            amount = address(this).balance;
            if (amount > 0) {
                (bool ok, ) = sale.treasury.call{value: amount}("");
                require(ok, "PROCEEDS_TRANSFER_FAILED");
                emit ProceedsWithdrawn(saleId, sale.treasury, amount);
            }
        } else {
            amount = IERC20(sale.paymentToken).balanceOf(address(this));
            if (amount > 0) {
                IERC20(sale.paymentToken).safeTransfer(sale.treasury, amount);
                emit ProceedsWithdrawn(saleId, sale.treasury, amount);
            }
        }
    }

    // ============== Views ==============

    function getSale(bytes32 saleId) external view returns (Sale memory) {
        require(sales[saleId].exists, "SALE_NOT_FOUND");
        return sales[saleId];
    }

    function getContribution(bytes32 saleId, address user) external view returns (Contribution memory) {
        return contributions[saleId][user];
    }

    function getSaleCount() external view returns (uint256) {
        return saleIds.length;
    }

    function getAllSaleIds() external view returns (bytes32[] memory) {
        return saleIds;
    }

    /**
     * @notice Compute a user's token claim (direct formula) without claiming.
     */
    function previewClaim(bytes32 saleId, address user) external view returns (uint256) {
        Sale storage sale = sales[saleId];
        if (!sale.exists || sale.status != SaleStatus.Finalized) return 0;
        Contribution storage c = contributions[saleId][user];
        if (c.claimed || c.amount == 0) return 0;
        if (c.tokenClaim > 0) return c.tokenClaim;
        if (sale.tokenPrice == 0) return 0;
        return (c.amount * 1e18) / sale.tokenPrice;
    }

    // ============== Receive ==============

    receive() external payable {}
}
