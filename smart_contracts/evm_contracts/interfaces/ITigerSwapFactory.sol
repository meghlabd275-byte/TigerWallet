// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface ITigerSwapFactory {
    event PairCreated(address indexed token0, address indexed token1, address pair, uint256);

    function feeTo() external view returns (address);
    function feeToSetter() external view returns (address);
    function allPairsLength() external view returns (uint256);
    function createPair(address tokenA, address tokenB) external returns (address pair);
    function getPair(address tokenA, address tokenB) external view returns (address pair);
    function getPairAddress(address tokenA, address tokenB) external view returns (address);
    function setFeeTo(address) external;
    function setFeeToSetter(address) external;
}
