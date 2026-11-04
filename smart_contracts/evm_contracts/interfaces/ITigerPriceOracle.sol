// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title ITigerPriceOracle
 * @notice Price feed interface consumed by the TigerWallet order book and
 *         perpetual contracts.
 * @dev The signature set is the Chainlink AggregatorV3 interface, so a Tiger
 *      feed and a Chainlink feed are interchangeable at the call site. The
 *      answer is always scaled by `decimals()`, which callers must read rather
 *      than assume, and `updatedAt` must be checked against a staleness bound
 *      before the answer is trusted.
 */
interface ITigerPriceOracle {
    /// @notice Number of decimals the `answer` from latestRoundData is scaled by.
    function decimals() external view returns (uint8);

    /**
     * @notice Latest price for the feed.
     * @return roundId        Round this answer belongs to.
     * @return answer         Price, scaled by decimals(). Reverts-free: callers
     *                        must reject a non-positive answer.
     * @return startedAt      Round start timestamp.
     * @return updatedAt      Timestamp the answer was published; compare
     *                        against block.timestamp to detect a stale feed.
     * @return answeredInRound Round the answer was computed in.
     */
    function latestRoundData()
        external
        view
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound);
}
