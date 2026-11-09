// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../TigerOrderBook.sol";
import "./Mocks.sol";

/**
 * @notice Behavioural tests for TigerOrderBook.
 * @dev The contract holds collateral, so the tests assert on balances and
 *      positions rather than only on events: a fill that reports the right
 *      price but moves the wrong amount of collateral is still a failure.
 */
contract TigerOrderBookTest is Test {
    TigerOrderBook internal book;
    MockToken internal quote;
    MockOracle internal oracle;

    address internal constant ALICE = address(0xA11CE);
    address internal constant BOB = address(0xB0B);
    address internal constant CAROL = address(0xCA401);

    bytes32 internal constant MARKET = keccak256("ETH-USD");

    uint256 internal constant PRICE_1_8 = 1e18;      // 1 quote per unit, 18 dp
    uint256 internal constant MAKER_FEE_BPS = 10;
    uint256 internal constant TAKER_FEE_BPS = 30;
    uint256 internal constant MAX_LEVERAGE = 10;
    uint256 internal constant MAX_PRICE_AGE = 300;

    function setUp() public {
        oracle = new MockOracle(8);
        // 2000.00000000 with 8 decimals.
        oracle.setAnswer(2000e8);

        book = new TigerOrderBook(address(oracle));
        quote = new MockToken();

        book.createMarket(MARKET, address(0xE7), address(quote), MAX_LEVERAGE, MAX_PRICE_AGE, 50);

        address[3] memory traders = [ALICE, BOB, CAROL];
        for (uint256 i = 0; i < traders.length; ++i) {
            quote.mint(traders[i], 1_000_000e18);
            vm.prank(traders[i]);
            quote.approve(address(book), type(uint256).max);
            vm.prank(traders[i]);
            book.depositCollateral(MARKET, 100_000e18);
        }
    }

    // ==================== Helpers ====================

    function _placeLimit(address who, uint8 side, uint256 price, uint256 quantity, uint256 margin)
        internal
        returns (bytes32)
    {
        vm.prank(who);
        return book.createLimitOrder(MARKET, side, price, quantity, 0, margin);
    }

    function _orderMargin(uint256 quantity, uint256 price) internal pure returns (uint256) {
        return (quantity * price / 1e18) / MAX_LEVERAGE;
    }

    // ==================== Oracle ====================

    function test_price_is_read_from_the_oracle_and_scaled_to_18_decimals() public view {
        assertEq(book.getPrice(MARKET), 2000e18);
    }

    /// The previous implementation returned a constant here. A constant would
    /// settle every fill at a rate unrelated to the market.
    function test_price_follows_the_oracle() public {
        oracle.setAnswer(1234e8);
        assertEq(book.getPrice(MARKET), 1234e18);

        oracle.setAnswer(1e8);
        assertEq(book.getPrice(MARKET), 1e18);
    }

    function test_price_rejects_a_non_positive_answer() public {
        oracle.setAnswer(0);
        vm.expectRevert(bytes("INVALID_ORACLE_PRICE"));
        book.getPrice(MARKET);

        oracle.setAnswer(-1);
        vm.expectRevert(bytes("INVALID_ORACLE_PRICE"));
        book.getPrice(MARKET);
    }

    function test_price_rejects_a_stale_answer() public {
        vm.warp(block.timestamp + MAX_PRICE_AGE + 1);
        vm.expectRevert(bytes("STALE_ORACLE_PRICE"));
        book.getPrice(MARKET);
    }

    function test_price_accepts_an_answer_at_the_staleness_bound() public {
        uint256 published = block.timestamp;
        vm.warp(published + MAX_PRICE_AGE);
        assertEq(book.getPrice(MARKET), 2000e18);
    }

    function test_price_reverts_for_an_unknown_market() public {
        vm.expectRevert(bytes("UNKNOWN_MARKET"));
        book.getPrice(keccak256("NOPE"));
    }

    // ==================== Collateral ====================

    function test_deposit_credits_the_received_amount() public {
        assertEq(book.collateralBalance(ALICE), 100_000e18);

        vm.prank(ALICE);
        book.depositCollateral(MARKET, 500e18);
        assertEq(book.collateralBalance(ALICE), 100_500e18);
        assertEq(quote.balanceOf(address(book)), 300_500e18);
    }

    function test_withdraw_returns_collateral() public {
        uint256 before = quote.balanceOf(ALICE);

        vm.prank(ALICE);
        book.withdrawCollateral(MARKET, 1_000e18);

        assertEq(book.collateralBalance(ALICE), 99_000e18);
        assertEq(quote.balanceOf(ALICE), before + 1_000e18);
    }

    function test_withdraw_beyond_balance_reverts() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("INSUFFICIENT_COLLATERAL"));
        book.withdrawCollateral(MARKET, 100_001e18);
    }

    // ==================== Limit orders ====================

    function test_limit_order_escrows_margin_and_rests_on_the_book() public {
        uint256 quantity = 5e18;
        uint256 price = 2000e18;
        uint256 margin = _orderMargin(quantity, price);
        bytes32 hash = _placeLimit(ALICE, 1, price, quantity, margin);

        assertEq(book.collateralBalance(ALICE), 100_000e18 - margin);
        (uint256 bestBid,) = book.getBestBid(MARKET);
        assertEq(bestBid, price);

        (,,,,,,, uint256 escrowed, uint8 status) = book.getOrder(hash);
        assertEq(escrowed, margin);
        assertEq(status, 1); // STATUS_OPEN
    }

    function test_limit_order_requires_escrow_to_cover_the_leverage() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("INSUFFICIENT_MARGIN"));
        book.createLimitOrder(MARKET, 1, 2000e18, 5e18, 0, 1);
    }

    function test_bids_are_ordered_best_first_and_asks_best_first() public {
        _placeLimit(ALICE, 1, 1900e18, 1e18, 1_000e18);
        _placeLimit(BOB, 1, 2100e18, 1e18, 1_000e18);
        _placeLimit(CAROL, 1, 2000e18, 1e18, 1_000e18);

        (uint256 bestBid,) = book.getBestBid(MARKET);
        assertEq(bestBid, 2100e18, "highest bid is best");

        _placeLimit(ALICE, 2, 2200e18, 1e18, 1_000e18);
        _placeLimit(BOB, 2, 2000e18, 1e18, 1_000e18);
        _placeLimit(CAROL, 2, 2100e18, 1e18, 1_000e18);

        (uint256 bestAsk,) = book.getBestAsk(MARKET);
        assertEq(bestAsk, 2000e18, "lowest ask is best");
    }

    function test_cancel_returns_the_escrow_and_clears_the_level() public {
        uint256 margin = _orderMargin(1e18, 2000e18);
        bytes32 hash = _placeLimit(ALICE, 1, 2000e18, 1e18, margin);

        vm.prank(ALICE);
        book.cancelOrder(hash);

        assertEq(book.collateralBalance(ALICE), 100_000e18, "escrow returned");
        (uint256 bestBid,) = book.getBestBid(MARKET);
        assertEq(bestBid, 0, "level removed");

        (,,,,,,,, uint8 status) = book.getOrder(hash);
        assertEq(status, 3); // STATUS_CANCELLED
    }

    function test_cancel_by_a_non_owner_reverts() public {
        bytes32 hash = _placeLimit(ALICE, 1, 2000e18, 1e18, 1_000e18);

        vm.prank(BOB);
        vm.expectRevert(bytes("NOT_OWNER"));
        book.cancelOrder(hash);
    }

    function test_cancelling_twice_reverts() public {
        bytes32 hash = _placeLimit(ALICE, 1, 2000e18, 1e18, 1_000e18);

        vm.prank(ALICE);
        book.cancelOrder(hash);

        vm.prank(ALICE);
        vm.expectRevert(bytes("NOT_OPEN"));
        book.cancelOrder(hash);
    }

    // ==================== Matching ====================

    /// The core case: a taker consumes a resting level at the maker's price.
    function test_market_order_fills_against_the_resting_book() public {
        uint256 quantity = 2e18;
        uint256 makerMargin = _orderMargin(quantity, 1900e18);
        _placeLimit(ALICE, 2, 1900e18, quantity, makerMargin);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, quantity, 100, 2_000e18);

        // Alice sold at her resting price, so she is short at 1900.
        (int256 aliceSize, uint256 aliceEntry,,,) = book.getPosition(ALICE, MARKET);
        assertEq(aliceSize, -int256(quantity));
        assertEq(aliceEntry, 1900e18);

        (int256 bobSize, uint256 bobEntry,,,) = book.getPosition(BOB, MARKET);
        assertEq(bobSize, int256(quantity));
        assertEq(bobEntry, 1900e18);

        (uint256 bestAsk,) = book.getBestAsk(MARKET);
        assertEq(bestAsk, 0, "consumed level removed");
    }

    /// A taker will not cross past its slippage bound.
    function test_market_order_respects_slippage() public {
        _placeLimit(ALICE, 2, 2500e18, 1e18, 2_500e18);

        // 1% slippage from 2000 is 2020, well below the 2500 ask.
        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);

        (int256 bobSize,,,,) = book.getPosition(BOB, MARKET);
        assertEq(bobSize, 0, "no position opened");
        (uint256 bestAsk,) = book.getBestAsk(MARKET);
        assertEq(bestAsk, 2500e18, "ask untouched");
    }

    /// A market order that cannot fill is cancelled and its escrow released,
    /// rather than left locked in an order that can never rest.
    function test_unfilled_market_order_returns_its_escrow() public {
        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);

        assertEq(book.collateralBalance(BOB), 100_000e18, "escrow returned");
    }

    /// A partially filled market order also returns the unfilled remainder's
    /// escrow; leaving it locked would strand collateral.
    function test_partially_filled_market_order_returns_the_remainder_escrow() public {
        _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 5e18, 100, 1_000e18);

        (int256 bobSize,,,,) = book.getPosition(BOB, MARKET);
        assertEq(bobSize, 1e18, "only the resting quantity filled");

        // BOB escrowed 1000 and 200 of it became position margin for the filled
        // unit at 10x, so the rest must be back in free collateral.
        uint256 free = book.collateralBalance(BOB);
        assertGt(free, 99_000e18, "unfilled escrow returned");
        assertEq(quote.balanceOf(address(book)) >= free, true);
    }

    /// A taker larger than the book consumes every level it can and stops.
    /// This is the case that previously looped forever.
    function test_sweeping_multiple_levels_terminates() public {
        _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);
        _placeLimit(ALICE, 2, 2010e18, 1e18, 201e18);
        _placeLimit(ALICE, 2, 2020e18, 1e18, 202e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 3e18, 100, 3_000e18);

        (int256 bobSize,,,,) = book.getPosition(BOB, MARKET);
        assertEq(bobSize, 3e18, "all three levels consumed");
        assertEq(book.getFillCount(MARKET), 3);
    }

    /// Asking for more than the book holds must stop at the book, not spin.
    function test_market_order_larger_than_the_book_stops_at_the_book() public {
        _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 10e18, 100, 5_000e18);

        (int256 bobSize,,,,) = book.getPosition(BOB, MARKET);
        assertEq(bobSize, 1e18, "filled only what was available");
    }

    /// Two takers must not both consume the same resting level.
    function test_a_level_cannot_be_filled_twice() public {
        _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);
        vm.prank(CAROL);
        book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);

        (int256 carolSize,,,,) = book.getPosition(CAROL, MARKET);
        assertEq(carolSize, 0, "second taker found no liquidity");
    }

    /// A sell taker matches the best (highest) bid.
    function test_sell_taker_matches_the_best_bid() public {
        _placeLimit(ALICE, 1, 1900e18, 1e18, 190e18);
        _placeLimit(CAROL, 1, 2000e18, 1e18, 200e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 2, 1e18, 100, 1_000e18);

        (int256 aliceSize,,,,) = book.getPosition(ALICE, MARKET);
        assertEq(aliceSize, 0, "lower bid untouched");

        (int256 carolSize, uint256 carolEntry,,,) = book.getPosition(CAROL, MARKET);
        assertEq(carolSize, 1e18, "best bid filled");
        assertEq(carolEntry, 2000e18);
    }

    /// Fees are charged, and a fee cannot leave the position with negative
    /// margin.
    function test_fees_are_collected_from_the_fill() public {
        _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);

        assertGt(book.feePool(), 0, "fee charged");
    }

    /// A fill must record the orders that produced it, so history can be
    /// reconciled against the book.
    function test_fill_records_the_real_order_hashes() public {
        bytes32 makerHash = _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);

        vm.prank(BOB);
        bytes32 takerHash = book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);

        TigerOrderBook.Fill memory fill = book.getFill(MARKET, 0);
        assertEq(fill.makerOrderHash, makerHash);
        assertEq(fill.takerOrderHash, takerHash);
        assertEq(fill.maker, ALICE);
        assertEq(fill.taker, BOB);
        assertEq(fill.price, 2000e18);
        assertEq(fill.quantity, 1e18);
    }

    // ==================== Positions ====================

    function test_open_position_takes_margin_from_collateral() public {
        uint256 margin = 1_000e18;

        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 2e18, margin);

        assertEq(book.collateralBalance(ALICE), 100_000e18 - margin);

        (int256 size, uint256 entryPrice,, uint256 positionMargin,) = book.getPosition(ALICE, MARKET);
        assertEq(size, 2e18);
        assertEq(entryPrice, 2000e18);
        assertEq(positionMargin, margin);
    }

    function test_open_position_requires_enough_margin_for_the_leverage() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("INSUFFICIENT_MARGIN"));
        // 2 units at 2000 is 4000 notional; at 10x it needs 400 margin.
        book.openPosition(MARKET, 1, 2e18, 399e18);
    }

    function test_open_position_requires_free_collateral() public {
        vm.prank(ALICE);
        book.withdrawCollateral(MARKET, 99_500e18);

        vm.prank(ALICE);
        vm.expectRevert(bytes("INSUFFICIENT_COLLATERAL"));
        book.openPosition(MARKET, 1, 2e18, 1_000e18);
    }

    /// A profitable long realises the gain and can withdraw it.
    function test_closing_a_profitable_long_realises_the_gain() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        oracle.setAnswer(2100e8);

        vm.prank(ALICE);
        int256 realised = book.closePosition(MARKET, 0);

        // 1 unit moved from 2000 to 2100.
        assertEq(realised, 100e18);

        (, , , uint256 margin, ) = book.getPosition(ALICE, MARKET);
        assertEq(margin, 0, "position cleared");

        // Equity went back to free collateral: 100_000 - 1000 + 1000 + 100.
        assertEq(book.collateralBalance(ALICE), 100_100e18);
    }

    function test_closing_a_losing_long_returns_less_than_the_margin() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        oracle.setAnswer(1900e8);

        vm.prank(ALICE);
        int256 realised = book.closePosition(MARKET, 0);
        assertEq(realised, -100e18);

        assertEq(book.collateralBalance(ALICE), 99_900e18);
    }

    function test_closing_a_profitable_short_realises_the_gain() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 2, 1e18, 1_000e18);

        oracle.setAnswer(1900e8);

        vm.prank(ALICE);
        int256 realised = book.closePosition(MARKET, 0);
        assertEq(realised, 100e18);
    }

    function test_closing_a_losing_short_returns_less() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 2, 1e18, 1_000e18);

        oracle.setAnswer(2100e8);

        vm.prank(ALICE);
        int256 realised = book.closePosition(MARKET, 0);
        assertEq(realised, -100e18);
    }

    function test_partial_close_keeps_the_remaining_position() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 2e18, 2_000e18);

        vm.prank(ALICE);
        book.closePosition(MARKET, 1e18);

        (int256 size, uint256 entryPrice, , uint256 margin, ) = book.getPosition(ALICE, MARKET);
        assertEq(size, 1e18);
        assertEq(entryPrice, 2000e18);
        assertEq(margin, 1_000e18, "half the margin released");
    }

    function test_closing_without_a_position_reverts() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("NO_POSITION"));
        book.closePosition(MARKET, 0);
    }

    /// Adding to a position must not realise PnL or release margin, and the
    /// entry price becomes the size-weighted average of the two fills.
    function test_increasing_a_position_keeps_it_open() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        oracle.setAnswer(2100e8);

        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        (int256 size, uint256 entryPrice, , uint256 margin, ) = book.getPosition(ALICE, MARKET);
        assertEq(size, 2e18);
        // (2000 + 2100) / 2
        assertEq(entryPrice, 2050e18);
        assertEq(margin, 2_000e18);
    }

    // ==================== Liquidation ====================

    function test_a_healthy_position_cannot_be_liquidated() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        vm.prank(BOB);
        vm.expectRevert(bytes("ABOVE_MAINTENANCE"));
        book.liquidate(ALICE, MARKET);
    }

    function test_a_position_below_maintenance_is_liquidatable() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        assertFalse(book.isLiquidatable(ALICE, MARKET));

        // 1 unit at 2000 with 1000 margin. Equity is P - 1000 and maintenance is
        // 5% of notional, so the position becomes liquidatable below 1052.6.
        oracle.setAnswer(1050e8);

        assertTrue(book.isLiquidatable(ALICE, MARKET));

        uint256 insuranceBefore = book.insuranceBalance();

        vm.prank(BOB);
        book.liquidate(ALICE, MARKET);

        assertGt(book.insuranceBalance(), insuranceBefore, "penalty taken");
        (int256 size, , , , ) = book.getPosition(ALICE, MARKET);
        assertEq(size, 0, "position cleared");
    }

    /// A position far under water must not underflow when equity is negative.
    function test_liquidating_an_insolvent_position_does_not_underflow() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        // A loss larger than the whole margin.
        oracle.setAnswer(500e8);

        vm.prank(BOB);
        book.liquidate(ALICE, MARKET);

        (int256 size, , , , ) = book.getPosition(ALICE, MARKET);
        assertEq(size, 0);
    }

    function test_liquidating_without_a_position_reverts() public {
        vm.prank(BOB);
        vm.expectRevert(bytes("NO_POSITION"));
        book.liquidate(ALICE, MARKET);
    }

    // ==================== Access control ====================

    function test_only_governance_can_create_a_market() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("ONLY_GOVERNANCE"));
        book.createMarket(keccak256("OTHER"), address(0xE7), address(quote), 10, 300, 50);
    }

    function test_only_governance_can_set_fees() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("ONLY_GOVERNANCE"));
        book.setFees(1, 1);
    }

    function test_fees_are_capped() public {
        vm.expectRevert(bytes("FEE_TOO_HIGH"));
        book.setFees(501, 10);
    }

    function test_only_governance_can_replace_the_oracle() public {
        vm.prank(ALICE);
        vm.expectRevert(bytes("ONLY_GOVERNANCE"));
        book.setPriceOracle(address(0x1234));
    }

    function test_oracle_cannot_be_zero() public {
        vm.expectRevert(bytes("INVALID_ORACLE"));
        book.setPriceOracle(address(0));
    }

    function test_pausing_blocks_trading_but_not_withdrawal() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        book.setPaused(true);

        vm.prank(ALICE);
        vm.expectRevert(bytes("PAUSED"));
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        // Withdrawals must keep working while trading is halted, otherwise a
        // pause would trap user funds.
        vm.prank(ALICE);
        book.withdrawCollateral(MARKET, 1_000e18);
    }

    function test_only_the_fee_recipient_can_claim_fees() public {
        _placeLimit(ALICE, 2, 2000e18, 1e18, 200e18);
        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 1e18, 100, 1_000e18);

        vm.prank(BOB);
        vm.expectRevert(bytes("NOT_FEE_RECIPIENT"));
        book.claimFees(MARKET);

        // The deployer is the fee recipient.
        uint256 claimed = book.feePool();
        book.claimFees(MARKET);
        assertEq(quote.balanceOf(address(this)), claimed);
    }

    // ==================== Depth view ====================

    function test_depth_view_walks_outwards_from_the_best_price() public {
        _placeLimit(ALICE, 1, 1900e18, 1e18, 190e18);
        _placeLimit(BOB, 1, 2000e18, 2e18, 400e18);
        _placeLimit(CAROL, 1, 2100e18, 3e18, 630e18);

        (uint256[] memory prices, uint256[] memory quantities) = book.getOrderBookDepth(MARKET, 1, 3);

        assertEq(prices[0], 2100e18, "best bid first");
        assertEq(prices[1], 2000e18);
        assertEq(prices[2], 1900e18);
        assertEq(quantities[0], 3e18);
        assertEq(quantities[2], 1e18);
    }

    function test_depth_view_handles_an_empty_book() public view {
        (uint256[] memory prices,) = book.getOrderBookDepth(MARKET, 1, 5);
        assertEq(prices.length, 0);
    }

    function test_depth_view_does_not_exceed_the_book() public {
        _placeLimit(ALICE, 1, 1900e18, 1e18, 190e18);
        (uint256[] memory prices,) = book.getOrderBookDepth(MARKET, 1, 10);
        assertEq(prices.length, 1);
    }

    // ==================== Solvency ====================

    /// The contract must always hold at least what it owes: user collateral
    /// plus the fee and insurance pools. A shortfall would mean one account
    /// could not withdraw.
    function test_contract_stays_solvent_through_a_trading_session() public {
        // Alice rests an ask; Bob takes it; Carol opens a position.
        _placeLimit(ALICE, 2, 2000e18, 2e18, 400e18);

        vm.prank(BOB);
        book.createMarketOrder(MARKET, 1, 2e18, 100, 1_000e18);

        vm.prank(CAROL);
        book.openPosition(MARKET, 2, 1e18, 1_000e18);

        oracle.setAnswer(2100e8);

        vm.prank(CAROL);
        book.closePosition(MARKET, 0);

        _assertSolvent();
    }

    function test_contract_stays_solvent_after_a_liquidation() public {
        vm.prank(ALICE);
        book.openPosition(MARKET, 1, 1e18, 1_000e18);

        oracle.setAnswer(1050e8);

        vm.prank(BOB);
        book.liquidate(ALICE, MARKET);

        _assertSolvent();
    }

    function test_contract_stays_solvent_after_cancels() public {
        bytes32 a = _placeLimit(ALICE, 1, 1900e18, 1e18, 190e18);
        _placeLimit(BOB, 1, 2000e18, 1e18, 200e18);

        vm.prank(ALICE);
        book.cancelOrder(a);

        _assertSolvent();
    }

    function _assertSolvent() internal view {
        uint256 owed = book.feePool() + book.insuranceBalance();
        owed += book.collateralBalance(ALICE);
        owed += book.collateralBalance(BOB);
        owed += book.collateralBalance(CAROL);

        assertGe(quote.balanceOf(address(book)), owed, "contract holds less than it owes");
    }
}
