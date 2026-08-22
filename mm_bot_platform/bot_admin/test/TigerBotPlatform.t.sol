// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test} from "forge-std/Test.sol";
import {TigerBotPlatform} from "../src/TigerBotPlatform.sol";

/// @title TigerBotPlatform Test
/// @notice Real Foundry tests for the on-chain bot trading platform. No mocks:
///         role-gated governance, bot lifecycle, exchange management, fee
///         configuration, emergency controls, and the fail-closed trading path
///         (reverts without a real router). All assertions are against real
///         storage state.
contract TigerBotPlatformTest is Test {
    TigerBotPlatform internal platform;
    address internal admin = address(0xA11CE);
    address internal botOperator = address(0xB07);
    address internal client = address(0xC11E);
    address internal other = address(0x0DD);

    bytes32 internal constant ROLE_ADMIN = keccak256("ADMIN");
    bytes32 internal constant ROLE_BOT_OPERATOR = keccak256("BOT_OPERATOR");
    bytes32 internal constant ROLE_CLIENT = keccak256("CLIENT");

    function setUp() public {
        vm.prank(admin);
        platform = new TigerBotPlatform();
    }

    // ---------------------------------------------------------------------
    // Role management
    // ---------------------------------------------------------------------

    function test_DeployerIsAdmin() public view {
        assertEq(platform.getUserRole(admin), ROLE_ADMIN);
        assertTrue(platform.hasRole(admin, ROLE_ADMIN));
    }

    function test_OnlyGovernanceCanGrantRole() public {
        vm.expectRevert("ONLY_GOVERNANCE");
        vm.prank(other);
        platform.grantRole(botOperator, ROLE_BOT_OPERATOR);
    }

    function test_GrantRoleSetsMembership() public {
        vm.prank(admin);
        platform.grantRole(botOperator, ROLE_BOT_OPERATOR);
        assertEq(platform.getUserRole(botOperator), ROLE_BOT_OPERATOR);
        assertTrue(platform.hasRole(botOperator, ROLE_BOT_OPERATOR));
    }

    function test_CannotRevokeAdmin() public {
        vm.prank(admin);
        vm.expectRevert("CANNOT_REVOKE_ADMIN");
        platform.revokeRole(admin);
    }

    // ---------------------------------------------------------------------
    // Bot lifecycle
    // ---------------------------------------------------------------------

    function _createBotAsClient() internal returns (bytes32 botId) {
        vm.startPrank(admin);
        platform.grantRole(client, ROLE_CLIENT);
        vm.stopPrank();
        vm.prank(client);
        botId = platform.createBot(
            1, "MM-Bot", "market maker", address(0x1), address(0x2), 1e18, 10e18, 500, 2
        );
    }

    function test_CreateBotAsClient() public {
        bytes32 botId = _createBotAsClient();
        assertTrue(botId != bytes32(0), "botId not zero");
        TigerBotPlatform.BotView memory b = platform.getBot(botId);
        assertEq(b.owner, client);
        assertEq(b.botType, 1);
        assertTrue(b.isActive);
        assertFalse(b.isPaused);
        assertEq(b.minInvestment, 1e18);
        assertEq(b.maxInvestment, 10e18);

        bytes32[] memory userBots = platform.getUserBots(client);
        assertEq(userBots.length, 1);
        assertEq(userBots[0], botId);
        (uint256 totalBots, uint256 active, , , ) = platform.getPlatformStats();
        assertEq(totalBots, 1);
        assertEq(active, 1);
    }

    function test_CreateBotRejectsNoRole() public {
        vm.expectRevert("NO_ROLE");
        vm.prank(other);
        platform.createBot(1, "x", "y", address(0x1), address(0x2), 1, 2, 100, 1);
    }

    function test_CreateBotRejectsInvalidBounds() public {
        vm.startPrank(admin);
        platform.grantRole(client, ROLE_CLIENT);
        vm.stopPrank();
        vm.prank(client);
        vm.expectRevert("INVALID_MAX");
        platform.createBot(1, "x", "y", address(0x1), address(0x2), 10e18, 1e18, 500, 2);
    }

    function test_CreateBotRejectsInvalidType() public {
        vm.startPrank(admin);
        platform.grantRole(client, ROLE_CLIENT);
        vm.stopPrank();
        vm.prank(client);
        vm.expectRevert("INVALID_TYPE");
        platform.createBot(10, "x", "y", address(0x1), address(0x2), 1, 2, 100, 1);
    }

    function test_PauseAndStopBot() public {
        bytes32 botId = _createBotAsClient();
        // Bot starts active + running (isPaused=false).
        assertTrue(platform.getBot(botId).isActive);
        assertFalse(platform.getBot(botId).isPaused);

        // Admin pauses the bot.
        vm.prank(admin);
        platform.pauseBot(botId);
        assertTrue(platform.getBot(botId).isPaused);

        // Owner stops the bot (sets isPaused=true, emits BotStopped).
        vm.prank(client);
        platform.stopBot(botId);
        assertTrue(platform.getBot(botId).isPaused);
        assertTrue(platform.getBot(botId).isActive); // still active, just stopped
    }

    function test_DeleteBotOnlyOwnerOrAdmin() public {
        bytes32 botId = _createBotAsClient();
        vm.expectRevert("NOT_OWNER");
        vm.prank(other);
        platform.deleteBot(botId);

        vm.prank(client);
        platform.deleteBot(botId);
        assertFalse(platform.getBot(botId).isActive);
        (, uint256 active, , , ) = platform.getPlatformStats();
        assertEq(active, 0);
    }

    // ---------------------------------------------------------------------
    // Exchange management
    // ---------------------------------------------------------------------

    function test_AddExchangeAdminOnly() public {
        vm.expectRevert("ONLY_ADMIN");
        vm.prank(other);
        platform.addExchange(bytes32("uniswap"), "Uniswap", "v2", address(0x7a25), 300000, 1, 1000e18, 30);

        vm.prank(admin);
        platform.addExchange(bytes32("uniswap"), "Uniswap", "v2", address(0x7a25), 300000, 1, 1000e18, 30);
        TigerBotPlatform.ExchangeView memory e = platform.getExchange(bytes32("uniswap"));
        assertEq(e.router, address(0x7a25));
        assertTrue(e.isActive);
        assertEq(e.minTradeSize, 1);
        assertEq(e.maxTradeSize, 1000e18);

        bytes32[] memory all = platform.getAllExchanges();
        assertEq(all.length, 1);
    }

    function test_RemoveExchange() public {
        vm.startPrank(admin);
        platform.addExchange(bytes32("ex"), "Ex", "ep", address(0x1), 300000, 1, 1000e18, 30);
        platform.removeExchange(bytes32("ex"));
        vm.stopPrank();
        assertFalse(platform.getExchange(bytes32("ex")).isActive);
    }

    function test_ConnectToExchange() public {
        vm.prank(admin);
        platform.addExchange(bytes32("ex"), "E", "p", address(0x1), 300000, 1, 1000e18, 30);
        vm.prank(other);
        platform.connectToExchange(bytes32("ex"));
        assertTrue(platform.canTradeOnExchange(other, bytes32("ex")));
    }

    // ---------------------------------------------------------------------
    // Emergency controls
    // ---------------------------------------------------------------------

    function test_EmergencyModeBlocksNewBots() public {
        vm.startPrank(admin);
        platform.grantRole(client, ROLE_CLIENT);
        platform.enableEmergencyMode();
        vm.stopPrank();
        vm.prank(client);
        vm.expectRevert("EMERGENCY_MODE");
        platform.createBot(1, "x", "y", address(0x1), address(0x2), 1, 2, 100, 1);
    }

    function test_PauseNewBotsCreation() public {
        vm.startPrank(admin);
        platform.grantRole(client, ROLE_CLIENT);
        platform.pauseNewBotsCreation();
        vm.stopPrank();
        vm.prank(client);
        vm.expectRevert("BOTS_PAUSED");
        platform.createBot(1, "x", "y", address(0x1), address(0x2), 1, 2, 100, 1);
    }

    // ---------------------------------------------------------------------
    // Trading path - fail-closed (no real router wired in unit test)
    // ---------------------------------------------------------------------

    function test_ExecuteTradeRevertsOnUnknownExchange() public {
        bytes32 botId = _createBotAsClient();
        vm.prank(client);
        vm.expectRevert("EXCHANGE_NOT_ACTIVE");
        platform.executeTrade(botId, bytes32("nope"), address(0x1), address(0x2), 1e18, 0, "");
    }

    function test_ExecuteTradeRevertsWhenTradingPaused() public {
        vm.startPrank(admin);
        platform.grantRole(client, ROLE_CLIENT);
        platform.addExchange(bytes32("ex"), "E", "p", address(0x1), 300000, 1, 1000e18, 30);
        bytes32 botId = platform.createBot(
            1, "MM", "d", address(0x1), address(0x2), 1e18, 10e18, 500, 2
        );
        platform.pauseAllTrading();
        vm.stopPrank();
        vm.prank(client);
        vm.expectRevert("TRADING_PAUSED");
        platform.executeTrade(botId, bytes32("ex"), address(0x1), address(0x2), 1e18, 0, "");
    }

    function test_QuoteSwapRevertsOnBadExchange() public {
        vm.expectRevert("BAD_EXCHANGE");
        platform.quoteSwap(bytes32("nope"), address(0x1), address(0x2), 1e18);
    }

    // ---------------------------------------------------------------------
    // Fee configuration
    // ---------------------------------------------------------------------

    function test_UpdateProtocolFeeAdminOnly() public {
        vm.expectRevert("ONLY_ADMIN");
        vm.prank(other);
        platform.updateProtocolFee(50);
    }

    function test_UpdateProtocolFeeCapsAt5Percent() public {
        vm.prank(admin);
        vm.expectRevert("FEE_TOO_HIGH");
        platform.updateProtocolFee(501);
    }

    function test_SetFeeRecipientRejectsZeroAddress() public {
        vm.prank(admin);
        vm.expectRevert("ZERO_ADDRESS");
        platform.setFeeRecipient(address(0));
    }

    // ---------------------------------------------------------------------
    // Governance transfer
    // ---------------------------------------------------------------------

    function test_TransferGovernanceTwoStep() public {
        address newGov = address(0x605);
        vm.prank(admin);
        platform.transferGovernance(newGov);
        vm.prank(newGov);
        platform.acceptGovernance();
        vm.prank(newGov);
        platform.grantRole(botOperator, ROLE_BOT_OPERATOR);
        assertTrue(platform.hasRole(botOperator, ROLE_BOT_OPERATOR));
    }

    function test_AcceptGovernanceOnlyPending() public {
        address newGov = address(0x605);
        vm.prank(admin);
        platform.transferGovernance(newGov);
        // Wrong caller cannot accept.
        vm.expectRevert("NOT_PENDING");
        vm.prank(other);
        platform.acceptGovernance();
    }
}
