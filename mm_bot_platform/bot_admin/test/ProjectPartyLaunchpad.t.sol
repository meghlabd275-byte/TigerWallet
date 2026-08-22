// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test} from "forge-std/Test.sol";
import {ProjectPartyLaunchpad} from "../src/ProjectPartyLaunchpad.sol";
import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// @notice Mirrors OZ v5 Ownable's custom error for testing
error OwnableUnauthorizedAccount(address account);

/// @title A simple mintable ERC-20 for testing the launchpad
contract TestToken is ERC20 {
    constructor(string memory name, string memory symbol) ERC20(name, symbol) {}

    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }
}

contract ProjectPartyLaunchpadTest is Test {
    ProjectPartyLaunchpad internal launchpad;
    TestToken internal projectToken;
    TestToken internal paymentToken;

    address internal owner = address(0x0FF1CE);
    address internal treasury = address(0x7E4A5E);
    address internal alice = address(0xA11CE);
    address internal bob = address(0xB0B);
    address internal attacker = address(0xBAD);

    bytes32 internal constant SALE_NATIVE = keccak256("sale_native");
    bytes32 internal constant SALE_ERC20 = keccak256("sale_erc20");

    function setUp() public {
        vm.prank(owner);
        launchpad = new ProjectPartyLaunchpad();

        projectToken = new TestToken("ProjectX", "PXT");
        paymentToken = new TestToken("USDT", "USDT");

        // Fund owner with project tokens.
        projectToken.mint(owner, 1_000_000 * 1e18);
        // Approve the launchpad to pull tokens from owner.
        vm.startPrank(owner);
        projectToken.approve(address(launchpad), type(uint256).max);
        vm.stopPrank();

        // Fund contributors.
        paymentToken.mint(alice, 100_000 * 1e18);
        paymentToken.mint(bob, 100_000 * 1e18);
        vm.startPrank(alice);
        paymentToken.approve(address(launchpad), type(uint256).max);
        vm.stopPrank();
        vm.startPrank(bob);
        paymentToken.approve(address(launchpad), type(uint256).max);
        vm.stopPrank();

        vm.deal(alice, 200 ether);
        vm.deal(bob, 100 ether);
    }

    // ---------------------------------------------------------------------
    // Sale creation
    // ---------------------------------------------------------------------

    function test_CreateNativeSale() public {
        uint256 tokensForSale = 100_000 * 1e18;
        uint256 startTime = block.timestamp;
        uint256 endTime = startTime + 1 days;

        vm.prank(owner);
        launchpad.createSale(
            SALE_NATIVE,
            address(projectToken),
            address(0), // native ETH
            treasury,
            0.01 ether, // 0.01 ETH per token
            tokensForSale,
            0, // no soft cap
            2000 ether, // hard cap
            0.01 ether, // min contribution
            100 ether, // max contribution
            startTime,
            endTime
        );

        // Tokens were escrowed into the launchpad.
        assertEq(projectToken.balanceOf(address(launchpad)), tokensForSale);

        ProjectPartyLaunchpad.Sale memory sale = launchpad.getSale(SALE_NATIVE);
        assertEq(sale.token, address(projectToken));
        assertEq(sale.paymentToken, address(0));
        assertEq(sale.treasury, treasury);
        assertEq(sale.tokensForSale, tokensForSale);
        assertTrue(sale.exists);
        assertEq(uint256(sale.status), uint256(ProjectPartyLaunchpad.SaleStatus.Pending));
    }

    function test_CreateSaleRejectsNonOwner() public {
        vm.expectRevert(abi.encodeWithSelector(OwnableUnauthorizedAccount.selector, attacker));
        vm.prank(attacker);
        launchpad.createSale(
            SALE_NATIVE, address(projectToken), address(0), treasury,
            1, 100e18, 0, 0, 1, 0, block.timestamp, block.timestamp + 1 days
        );
    }

    function test_CreateSaleRejectsDuplicateId() public {
        _createNativeSale();
        vm.expectRevert("SALE_EXISTS");
        vm.prank(owner);
        launchpad.createSale(
            SALE_NATIVE, address(projectToken), address(0), treasury,
            0.01 ether, 100e18, 0, 0, 0.01 ether, 0,
            block.timestamp, block.timestamp + 1 days
        );
    }

    function test_CreateSaleRejectsBadTimeRange() public {
        vm.expectRevert("BAD_TIME_RANGE");
        vm.prank(owner);
        launchpad.createSale(
            SALE_NATIVE, address(projectToken), address(0), treasury,
            0.01 ether, 100e18, 0, 0, 0.01 ether, 0,
            block.timestamp + 100, block.timestamp
        );
    }

    // ---------------------------------------------------------------------
    // Lifecycle: start → contribute → finalize → claim
    // ---------------------------------------------------------------------

    function _createNativeSale() internal {
        vm.prank(owner);
        launchpad.createSale(
            SALE_NATIVE,
            address(projectToken),
            address(0),
            treasury,
            0.01 ether, // 0.01 ETH per token => 1 token per 0.01 ETH
            100_000 * 1e18, // 100k tokens
            0, // no soft cap
            2000 ether, // hard cap
            0.01 ether, // min contribution
            100 ether, // max contribution
            block.timestamp,
            block.timestamp + 1 days
        );
    }

    function test_FullNativeSaleLifecycle() public {
        _createNativeSale();

        // Start the sale.
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);
        assertEq(uint256(launchpad.getSale(SALE_NATIVE).status), uint256(ProjectPartyLaunchpad.SaleStatus.Active));

        // Alice contributes 1 ETH => should get 100 tokens (1 / 0.01).
        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);

        ProjectPartyLaunchpad.Contribution memory c = launchpad.getContribution(SALE_NATIVE, alice);
        assertEq(c.amount, 1 ether);

        // Bob contributes 2 ETH => 200 tokens.
        vm.prank(bob);
        launchpad.contribute{value: 2 ether}(SALE_NATIVE);
        assertEq(launchpad.getContribution(SALE_NATIVE, bob).amount, 2 ether);

        // Fast-forward to end time.
        vm.warp(block.timestamp + 1 days + 1);

        // Finalize.
        vm.prank(owner);
        launchpad.finalizeSale(SALE_NATIVE);

        ProjectPartyLaunchpad.Sale memory sale = launchpad.getSale(SALE_NATIVE);
        assertEq(uint256(sale.status), uint256(ProjectPartyLaunchpad.SaleStatus.Finalized));
        // tokensSold = totalRaised * 1e18 / tokenPrice = 3 ether * 1e18 / 0.01 ether = 300e18
        assertEq(sale.tokensSold, 300 * 1e18);

        // Proceeds went to treasury.
        assertEq(treasury.balance, 3 ether);

        // Alice claims 100 tokens.
        uint256 aliceBalBefore = projectToken.balanceOf(alice);
        vm.prank(alice);
        launchpad.claimTokens(SALE_NATIVE);
        assertEq(projectToken.balanceOf(alice) - aliceBalBefore, 100 * 1e18);

        // Bob claims 200 tokens.
        uint256 bobBalBefore = projectToken.balanceOf(bob);
        vm.prank(bob);
        launchpad.claimTokens(SALE_NATIVE);
        assertEq(projectToken.balanceOf(bob) - bobBalBefore, 200 * 1e18);

        // Double-claim reverts.
        vm.expectRevert("ALREADY_CLAIMED");
        vm.prank(alice);
        launchpad.claimTokens(SALE_NATIVE);
    }

    // ---------------------------------------------------------------------
    // Constraints: min/max contribution, hard cap
    // ---------------------------------------------------------------------

    function test_ContributeRejectsBelowMin() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);

        vm.expectRevert("BELOW_MIN");
        vm.prank(alice);
        launchpad.contribute{value: 0.001 ether}(SALE_NATIVE); // below 0.01 min
    }

    function test_ContributeRejectsAboveMax() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);

        // Max is 100 ether; contribute 100 + then 1 more should revert.
        vm.prank(alice);
        launchpad.contribute{value: 100 ether}(SALE_NATIVE);
        vm.expectRevert("ABOVE_MAX");
        vm.prank(alice);
        launchpad.contribute{value: 0.01 ether}(SALE_NATIVE);
    }

    function test_ContributeRejectsWhenNotActive() public {
        _createNativeSale();
        // Still pending (not started).
        vm.expectRevert("NOT_ACTIVE");
        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);
    }

    function test_ContributeRejectsWhenPaused() public {
        _createNativeSale();
        vm.startPrank(owner);
        launchpad.startSale(SALE_NATIVE);
        launchpad.pauseSale(SALE_NATIVE);
        vm.stopPrank();
        vm.expectRevert("NOT_ACTIVE");
        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);
    }

    // ---------------------------------------------------------------------
    // Cancel + refund
    // ---------------------------------------------------------------------

    function test_CancelAndRefund() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);

        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);

        vm.prank(owner);
        launchpad.cancelSale(SALE_NATIVE);
        assertEq(uint256(launchpad.getSale(SALE_NATIVE).status), uint256(ProjectPartyLaunchpad.SaleStatus.Cancelled));

        uint256 aliceBalBefore = alice.balance;
        vm.prank(alice);
        launchpad.claimRefund(SALE_NATIVE);
        // Alice got her 1 ETH back (minus gas, which is 0 in test).
        assertEq(alice.balance - aliceBalBefore, 1 ether);

        // Double-refund reverts.
        vm.expectRevert("ALREADY_REFUNDED");
        vm.prank(alice);
        launchpad.claimRefund(SALE_NATIVE);
    }

    function test_ClaimRefundRejectsWhenNotCancelled() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);
        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);
        vm.expectRevert("NOT_CANCELLED");
        vm.prank(alice);
        launchpad.claimRefund(SALE_NATIVE);
    }

    // ---------------------------------------------------------------------
    // ERC-20 sale
    // ---------------------------------------------------------------------

    function test_ERC20SaleLifecycle() public {
        uint256 tokensForSale = 100_000 * 1e18;
        uint256 pricePerToken = 1 * 1e18; // 1 USDT per token
        uint256 startTime = block.timestamp;
        uint256 endTime = startTime + 1 days;

        vm.prank(owner);
        launchpad.createSale(
            SALE_ERC20,
            address(projectToken),
            address(paymentToken),
            treasury,
            pricePerToken,
            tokensForSale,
            0,
            0,
            1e18, // min 1 USDT
            1000 * 1e18, // max 1000 USDT
            startTime,
            endTime
        );

        vm.prank(owner);
        launchpad.startSale(SALE_ERC20);

        // Alice contributes 50 USDT => 50 tokens.
        vm.prank(alice);
        launchpad.contributeERC20(SALE_ERC20, 50 * 1e18);

        assertEq(launchpad.getContribution(SALE_ERC20, alice).amount, 50 * 1e18);
        // Payment tokens are escrowed in the launchpad.
        assertEq(paymentToken.balanceOf(address(launchpad)), 50 * 1e18);

        // Warp to end + finalize.
        vm.warp(block.timestamp + 1 days + 1);
        vm.prank(owner);
        launchpad.finalizeSale(SALE_ERC20);

        // Proceeds went to treasury.
        assertEq(paymentToken.balanceOf(treasury), 50 * 1e18);

        // Alice claims 50 tokens.
        uint256 before = projectToken.balanceOf(alice);
        vm.prank(alice);
        launchpad.claimTokens(SALE_ERC20);
        assertEq(projectToken.balanceOf(alice) - before, 50 * 1e18);
    }

    function test_ERC20ContributeRejectsOnNativeSale() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);
        vm.expectRevert("NOT_ERC20_SALE");
        vm.prank(alice);
        launchpad.contributeERC20(SALE_NATIVE, 1e18);
    }

    function test_NativeContributeRejectsOnERC20Sale() public {
        vm.prank(owner);
        launchpad.createSale(
            SALE_ERC20, address(projectToken), address(paymentToken), treasury,
            1e18, 100e18, 0, 0, 1e18, 0, block.timestamp, block.timestamp + 1 days
        );
        vm.prank(owner);
        launchpad.startSale(SALE_ERC20);
        vm.expectRevert("NOT_NATIVE_SALE");
        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_ERC20);
    }

    // ---------------------------------------------------------------------
    // Withdraw unsold
    // ---------------------------------------------------------------------

    function test_WithdrawUnsoldAfterFinalize() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);

        // Only 1 ETH contributed (100 tokens) out of 100k.
        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);

        vm.warp(block.timestamp + 1 days + 1);
        vm.prank(owner);
        launchpad.finalizeSale(SALE_NATIVE);

        vm.prank(alice);
        launchpad.claimTokens(SALE_NATIVE);

        // Withdraw unsold.
        uint256 ownerBalBefore = projectToken.balanceOf(owner);
        vm.prank(owner);
        launchpad.withdrawUnsold(SALE_NATIVE);
        assertEq(projectToken.balanceOf(owner) - ownerBalBefore, 100_000 * 1e18 - 100 * 1e18);
    }

    function test_WithdrawUnsoldRejectsWhenActive() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);
        vm.expectRevert("NOT_ENDED");
        vm.prank(owner);
        launchpad.withdrawUnsold(SALE_NATIVE);
    }

    // ---------------------------------------------------------------------
    // Preview claim
    // ---------------------------------------------------------------------

    function test_PreviewClaim() public {
        _createNativeSale();
        vm.prank(owner);
        launchpad.startSale(SALE_NATIVE);

        vm.prank(alice);
        launchpad.contribute{value: 1 ether}(SALE_NATIVE);

        // Before finalize, preview = 0.
        assertEq(launchpad.previewClaim(SALE_NATIVE, alice), 0);

        vm.warp(block.timestamp + 1 days + 1);
        vm.prank(owner);
        launchpad.finalizeSale(SALE_NATIVE);

        // After finalize: 1 ETH / 0.01 ETH * 1e18 = 100e18 tokens.
        assertEq(launchpad.previewClaim(SALE_NATIVE, alice), 100 * 1e18);
    }

    // ---------------------------------------------------------------------
    // View functions
    // ---------------------------------------------------------------------

    function test_GetSaleCount() public {
        assertEq(launchpad.getSaleCount(), 0);
        _createNativeSale();
        assertEq(launchpad.getSaleCount(), 1);
    }

    function test_GetAllSaleIds() public {
        _createNativeSale();
        bytes32[] memory ids = launchpad.getAllSaleIds();
        assertEq(ids.length, 1);
        assertEq(ids[0], SALE_NATIVE);
    }
}
