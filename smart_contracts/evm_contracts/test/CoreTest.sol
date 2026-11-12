// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../contracts/TigerToken.sol";
import "../contracts/Factory.sol";
import "../contracts/Pair.sol";
import "../contracts/Router.sol";

/**
 * @title TigerSwap Core Test Suite
 * @notice Tests for the TigerSwap DEX core contracts.
 */
contract TigerSwapCoreTest is Test {
    TigerToken public tokenA;
    TigerToken public tokenB;
    TigerSwapFactory public factory;
    TigerSwapRouter public router;

    address public deployer = address(this);
    address public user1 = address(0xBEEF);
    address public user2 = address(0xCAFE);

    uint256 constant MAX_SUPPLY = 1000000000 * 10**18;

    function setUp() public {
        tokenA = new TigerToken();
        tokenB = new TigerToken();
        factory = new TigerSwapFactory(deployer);
        router = new TigerSwapRouter(address(factory), address(0));

        // Fund the test users from the deployer allocation.
        tokenA.transfer(user1, 100000e18);
        tokenB.transfer(user1, 100000e18);
        tokenA.transfer(user2, 100000e18);
        tokenB.transfer(user2, 100000e18);
    }

    // ==================== Token ====================

    function testTokenDeployment() public {
        assertEq(tokenA.name(), "TigerSwap");
        assertEq(tokenA.symbol(), "TIGER");
        assertEq(tokenA.decimals(), 18);
        assertEq(tokenA.owner(), deployer);
        // 10% team + 5% investor + 5% treasury.
        assertEq(tokenA.totalSupply(), MAX_SUPPLY * 20 / 100);
    }

    function testTokenTransfer() public {
        uint256 before = tokenA.balanceOf(user2);
        tokenA.transfer(user2, 1000e18);
        assertEq(tokenA.balanceOf(user2), before + 1000e18);
    }

    function testTokenApproveAndTransferFrom() public {
        tokenA.approve(user2, 500e18);
        assertEq(tokenA.allowance(deployer, user2), 500e18);

        uint256 before = tokenA.balanceOf(user1);
        vm.prank(user2);
        tokenA.transferFrom(deployer, user1, 500e18);
        assertEq(tokenA.balanceOf(user1), before + 500e18);
        assertEq(tokenA.allowance(deployer, user2), 0);
    }

    function testTokenTransferExceedingBalanceReverts() public {
        vm.prank(user1);
        vm.expectRevert();
        tokenA.transfer(user2, 100000e18 + 1);
    }

    function testMintRespectsMaxSupply() public {
        tokenA.addMinter(deployer);
        uint256 remaining = MAX_SUPPLY - tokenA.totalSupply();
        tokenA.mint(user1, remaining);
        assertEq(tokenA.totalSupply(), MAX_SUPPLY);

        vm.expectRevert("TigerToken: MAX_SUPPLY_EXCEEDED");
        tokenA.mint(user1, 1);
    }

    function testOnlyMinterCanMint() public {
        vm.prank(user1);
        vm.expectRevert("TigerToken: NOT_MINTER");
        tokenA.mint(user1, 1e18);
    }

    // ==================== Factory ====================

    function testFactoryDeployment() public {
        assertEq(factory.feeToSetter(), deployer);
        assertEq(factory.feeTo(), address(0));
        assertEq(factory.allPairsLength(), 0);
    }

    function testCreatePair() public {
        address pair = factory.createPair(address(tokenA), address(tokenB));
        assertTrue(pair != address(0));
        assertEq(factory.allPairsLength(), 1);
        // The lookup is order independent.
        assertEq(factory.getPairAddress(address(tokenA), address(tokenB)), pair);
        assertEq(factory.getPairAddress(address(tokenB), address(tokenA)), pair);
    }

    function testCreatePairRejectsIdenticalTokens() public {
        vm.expectRevert("TigerSwap: IDENTICAL_ADDRESSES");
        factory.createPair(address(tokenA), address(tokenA));
    }

    function testCreatePairRejectsDuplicate() public {
        factory.createPair(address(tokenA), address(tokenB));
        vm.expectRevert("TigerSwap: PAIR_EXISTS");
        factory.createPair(address(tokenB), address(tokenA));
    }

    function testSetFeeToIsGovernanceGated() public {
        vm.prank(user1);
        vm.expectRevert("TigerSwap: FORBIDDEN");
        factory.setFeeTo(user1);

        factory.setFeeTo(user1);
        assertEq(factory.feeTo(), user1);
    }

    function testSetFeeToSetterIsGovernanceGated() public {
        vm.prank(user1);
        vm.expectRevert("TigerSwap: FORBIDDEN");
        factory.setFeeToSetter(user1);

        factory.setFeeToSetter(user1);
        assertEq(factory.feeToSetter(), user1);
    }

    // ==================== Pair ====================

    function testPairIsInitializedInAscendingOrder() public {
        address pairAddress = factory.createPair(address(tokenA), address(tokenB));
        TigerSwapPair pair = TigerSwapPair(pairAddress);

        assertEq(pair.factory(), address(factory));
        assertTrue(pair.token0() < pair.token1());
        assertEq(pair.isLocked(), false);
        assertEq(pair.totalSupply(), 0);
    }

    function testMintLiquidityDirect() public {
        address pairAddress = factory.createPair(address(tokenA), address(tokenB));
        TigerSwapPair pair = TigerSwapPair(pairAddress);

        tokenA.transfer(pairAddress, 10000e18);
        tokenB.transfer(pairAddress, 10000e18);

        uint256 liquidity = pair.mint(user1);
        // First deposit mints sqrt(amount0 * amount1) minus the locked minimum.
        assertEq(liquidity, _sqrt(10000e18 * 10000e18) - 1000);
        assertEq(pair.balanceOf(user1), liquidity);
        assertEq(pair.totalSupply(), liquidity + 1000);

        (uint112 r0, uint112 r1,) = pair.getReserves();
        assertEq(uint256(r0), 10000e18);
        assertEq(uint256(r1), 10000e18);
    }

    function testBurnReturnsUnderlyingTokens() public {
        address pairAddress = factory.createPair(address(tokenA), address(tokenB));
        TigerSwapPair pair = TigerSwapPair(pairAddress);

        tokenA.transfer(pairAddress, 10000e18);
        tokenB.transfer(pairAddress, 10000e18);
        uint256 liquidity = pair.mint(user1);

        // burn() settles whatever LP the pair holds, so the owner must send the
        // LP in first. This is exactly what Router.removeLiquidity does.
        vm.prank(user1);
        pair.transfer(pairAddress, liquidity);
        uint256 beforeA = tokenA.balanceOf(user1);
        uint256 beforeB = tokenB.balanceOf(user1);

        (uint256 amount0, uint256 amount1) = pair.burn(user1);
        assertEq(tokenA.balanceOf(user1) - beforeA, amount0);
        assertEq(tokenB.balanceOf(user1) - beforeB, amount1);
        assertTrue(amount0 > 0 && amount1 > 0);
    }

    function testRouterRemoveLiquidity() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        uint256 liquidity = TigerSwapPair(pairAddress).balanceOf(deployer);
        assertTrue(liquidity > 0);

        uint256 beforeA = tokenA.balanceOf(user1);
        uint256 beforeB = tokenB.balanceOf(user1);

        // LP tokens are ERC20, so the router pulls them with transferFrom.
        TigerSwapPair(pairAddress).approve(address(router), liquidity);
        (uint256 amountA, uint256 amountB) = router.removeLiquidity(
            address(tokenA),
            address(tokenB),
            liquidity,
            0,
            0,
            user1,
            block.timestamp + 1 hours
        );

        assertTrue(amountA > 0 && amountB > 0);
        assertEq(tokenA.balanceOf(user1) - beforeA, amountA);
        assertEq(tokenB.balanceOf(user1) - beforeB, amountB);
    }

    function testLpTransferRequiresAllowance() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        TigerSwapPair pair = TigerSwapPair(pairAddress);
        uint256 liquidity = pair.balanceOf(deployer);

        vm.prank(user1);
        vm.expectRevert("TigerSwap: INSUFFICIENT_ALLOWANCE");
        pair.transferFrom(deployer, user1, liquidity);
    }

    function testLpTransferMovesBalance() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        TigerSwapPair pair = TigerSwapPair(pairAddress);
        uint256 liquidity = pair.balanceOf(deployer);

        pair.transfer(user1, liquidity);
        assertEq(pair.balanceOf(user1), liquidity);
        assertEq(pair.balanceOf(deployer), 0);
    }

    // ==================== Router ====================

    function _seedLiquidity(uint256 amountA, uint256 amountB) internal returns (address pair) {
        pair = factory.createPair(address(tokenA), address(tokenB));
        tokenA.approve(address(router), amountA);
        tokenB.approve(address(router), amountB);
        router.addLiquidity(
            address(tokenA),
            address(tokenB),
            amountA,
            amountB,
            0,
            0,
            deployer,
            block.timestamp + 1 hours
        );
    }

    function testRouterAddLiquidity() public {
        address pair = _seedLiquidity(10000e18, 10000e18);

        assertEq(tokenA.balanceOf(pair), 10000e18);
        assertEq(tokenB.balanceOf(pair), 10000e18);
        assertTrue(TigerSwapPair(pair).balanceOf(deployer) > 0);
    }

    function testRouterAddLiquidityRespectsMinimums() public {
        factory.createPair(address(tokenA), address(tokenB));
        tokenA.approve(address(router), 10000e18);
        tokenB.approve(address(router), 10000e18);

        vm.expectRevert("TigerSwapRouter: INSUFFICIENT_A_AMOUNT");
        router.addLiquidity(
            address(tokenA),
            address(tokenB),
            10000e18,
            10000e18,
            10001e18,
            0,
            deployer,
            block.timestamp + 1 hours
        );
    }

    function testRouterRejectsExpiredDeadline() public {
        factory.createPair(address(tokenA), address(tokenB));
        tokenA.approve(address(router), 1000e18);
        tokenB.approve(address(router), 1000e18);

        vm.expectRevert("TigerSwapRouter: EXPIRED");
        router.addLiquidity(
            address(tokenA),
            address(tokenB),
            1000e18,
            1000e18,
            0,
            0,
            deployer,
            block.timestamp - 1
        );
    }

    function testGetAmountOutFollowsFeeFormula() public {
        uint256 amountIn = 1000e18;
        uint256 reserveIn = 10000e18;
        uint256 reserveOut = 10000e18;

        uint256 expected = (amountIn * 997 * reserveOut) / (reserveIn * 1000 + amountIn * 997);
        assertEq(router.getAmountOut(amountIn, reserveIn, reserveOut), expected);
    }

    function testGetAmountInRoundTrips() public {
        uint256 reserveIn = 10000e18;
        uint256 reserveOut = 10000e18;
        uint256 desiredOut = 500e18;

        uint256 amountIn = router.getAmountIn(desiredOut, reserveIn, reserveOut);
        assertTrue(router.getAmountOut(amountIn, reserveIn, reserveOut) >= desiredOut);
    }

    function testGetAmountOutRejectsZeroInput() public {
        vm.expectRevert("TigerSwapRouter: INSUFFICIENT_INPUT_AMOUNT");
        router.getAmountOut(0, 1000e18, 1000e18);
    }

    function testSwapExactTokensForTokens() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        TigerSwapPair pair = TigerSwapPair(pairAddress);
        (uint112 r0Before, uint112 r1Before,) = pair.getReserves();

        // Reserves are ordered by token address, which is unrelated to the path
        // order the caller chose.
        bool aIsToken0 = address(tokenA) == pair.token0();
        uint256 reserveInBefore = aIsToken0 ? r0Before : r1Before;
        uint256 reserveOutBefore = aIsToken0 ? r1Before : r0Before;

        uint256 amountIn = 100e18;
        uint256 expectedOut = router.getAmountOut(amountIn, reserveInBefore, reserveOutBefore);

        uint256 beforeBal = tokenB.balanceOf(user1);
        vm.prank(user1);
        tokenA.approve(address(router), amountIn);
        vm.prank(user1);
        uint256[] memory amounts = router.swapExactTokensForTokens(
            amountIn,
            0,
            _path(address(tokenA), address(tokenB)),
            user1,
            block.timestamp + 1 hours
        );

        assertEq(amounts[amounts.length - 1], expectedOut);
        assertEq(tokenB.balanceOf(user1) - beforeBal, expectedOut);

        (uint112 r0After, uint112 r1After,) = pair.getReserves();
        uint256 reserveInAfter = aIsToken0 ? r0After : r1After;
        uint256 reserveOutAfter = aIsToken0 ? r1After : r0After;
        assertEq(reserveInAfter, reserveInBefore + amountIn);
        assertEq(reserveOutAfter, reserveOutBefore - expectedOut);
    }

    function testSwapReverseDirection() public {
        // Exercises the opposite token ordering, which is the branch that used
        // to request the input amount as the swap output.
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        TigerSwapPair pair = TigerSwapPair(pairAddress);
        (uint112 r0Before, uint112 r1Before,) = pair.getReserves();

        bool bIsToken0 = address(tokenB) == pair.token0();
        uint256 reserveInBefore = bIsToken0 ? r0Before : r1Before;
        uint256 reserveOutBefore = bIsToken0 ? r1Before : r0Before;

        uint256 amountIn = 250e18;
        uint256 expectedOut = router.getAmountOut(amountIn, reserveInBefore, reserveOutBefore);

        uint256 beforeBal = tokenA.balanceOf(user1);
        vm.prank(user1);
        tokenB.approve(address(router), amountIn);
        vm.prank(user1);
        uint256[] memory amounts = router.swapExactTokensForTokens(
            amountIn,
            0,
            _path(address(tokenB), address(tokenA)),
            user1,
            block.timestamp + 1 hours
        );

        assertEq(amounts[amounts.length - 1], expectedOut);
        assertEq(tokenA.balanceOf(user1) - beforeBal, expectedOut);

        (uint112 r0After, uint112 r1After,) = pair.getReserves();
        uint256 reserveInAfter = bIsToken0 ? r0After : r1After;
        uint256 reserveOutAfter = bIsToken0 ? r1After : r0After;
        assertEq(reserveInAfter, reserveInBefore + amountIn);
        assertEq(reserveOutAfter, reserveOutBefore - expectedOut);
    }

    function testSwapRespectsMinimumOutput() public {
        _seedLiquidity(10000e18, 10000e18);
        uint256 amountIn = 100e18;

        vm.prank(user1);
        tokenA.approve(address(router), amountIn);
        vm.prank(user1);
        vm.expectRevert("TigerSwapRouter: INSUFFICIENT_OUTPUT");
        router.swapExactTokensForTokens(
            amountIn,
            type(uint256).max,
            _path(address(tokenA), address(tokenB)),
            user1,
            block.timestamp + 1 hours
        );
    }

    function testPairSwapRejectsEmptyOutput() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        vm.expectRevert("TigerSwap: INSUFFICIENT_OUTPUT_AMOUNT");
        TigerSwapPair(pairAddress).swap(0, 0, user1);
    }

    function testPairSwapRejectsInsufficientLiquidity() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        vm.expectRevert("TigerSwap: INSUFFICIENT_LIQUIDITY");
        TigerSwapPair(pairAddress).swap(10000e18, 0, user1);
    }

    function testSwapRequiresInputAmount() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        // Withdrawing without paying anything in must revert.
        vm.expectRevert("TigerSwap: INSUFFICIENT_INPUT_AMOUNT");
        TigerSwapPair(pairAddress).swap(1e18, 0, user1);
    }

    function testPairUnlockedAfterOperations() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        TigerSwapPair pair = TigerSwapPair(pairAddress);
        assertEq(pair.isLocked(), false);

        vm.prank(user1);
        tokenA.approve(address(router), 100e18);
        vm.prank(user1);
        router.swapExactTokensForTokens(
            100e18,
            0,
            _path(address(tokenA), address(tokenB)),
            user1,
            block.timestamp + 1 hours
        );

        // The reentrancy lock must always be released, or the pool is bricked.
        assertEq(pair.isLocked(), false);
    }

    function testSkimRecoversDonatedTokens() public {
        address pairAddress = _seedLiquidity(10000e18, 10000e18);
        TigerSwapPair pair = TigerSwapPair(pairAddress);

        tokenA.transfer(pairAddress, 500e18);
        uint256 before = tokenA.balanceOf(user1);
        pair.skim(user1);
        assertEq(tokenA.balanceOf(user1) - before, 500e18);
    }

    // ==================== Helpers ====================

    function _path(address a, address b) internal pure returns (address[] memory path) {
        path = new address[](2);
        path[0] = a;
        path[1] = b;
    }

    function _sqrt(uint256 y) internal pure returns (uint256 z) {
        if (y > 3) {
            z = y;
            uint256 x = y / 2 + 1;
            while (x < z) {
                z = x;
                x = (y / x + x) / 2;
            }
        } else if (y != 0) {
            z = 1;
        }
    }
}
