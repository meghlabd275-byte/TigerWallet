// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface ITigerSwapRouter {
    function factory() external view returns (address);
    function WETH() external view returns (address);
}
