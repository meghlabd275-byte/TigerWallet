// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../contracts/governance/EmergencyPause.sol";
import "../contracts/governance/EmergencyCouncil.sol";
import "../contracts/governance/UpgradeManager.sol";
import "../contracts/governance/InsuranceVault.sol";
import "../contracts/governance/TimelockController.sol";

/**
 * @title Governance and security-control test suite
 * @notice Covers the pause switch, the emergency council, the upgrade manager,
 *         the insurance vault and the timelock.
 *
 * These contracts shipped with no tests while still being compiled by the dex
 * profile, so the access-control defects below went unnoticed. The rejection
 * cases are the point of this suite: for every privileged entry point there is
 * a test that a non-privileged caller cannot reach it.
 */
contract GovernanceTest is Test {
    /// The insurance vault pays out with .transfer(), so the test contract has
    /// to be able to receive ether.
    receive() external payable {}

    address internal owner = address(this);
    address internal alice = address(0xA11CE);
    address internal bob = address(0xB0B);
    address internal mallory = address(0xBAD);

    // =====================================================================
    // EmergencyPause
    // =====================================================================

    EmergencyPause internal pauseCtl;

    function setUp() public {
        pauseCtl = new EmergencyPause();
    }

    function testEmergencyPauseDeployerIsGuardian() public view {
        assertTrue(pauseCtl.guardians(owner));
        assertEq(pauseCtl.owner(), owner);
    }

    function testEmergencyPauseNonGuardianCannotPause() public {
        vm.prank(mallory);
        vm.expectRevert("Not guardian");
        pauseCtl.pause("unauthorised");
        assertFalse(pauseCtl.paused());
    }

    function testEmergencyPauseGuardianCanPause() public {
        pauseCtl.pause("maintenance");
        assertTrue(pauseCtl.paused());
    }

    /// The original contract let any caller unpause, which makes the pause
    /// meaningless as a safety control.
    function testEmergencyPauseNonGuardianCannotUnpause() public {
        pauseCtl.pause("maintenance");
        vm.prank(mallory);
        vm.expectRevert("Not guardian");
        pauseCtl.unpause();
        assertTrue(pauseCtl.paused());
    }

    function testEmergencyPauseGuardianCanUnpause() public {
        pauseCtl.pause("maintenance");
        pauseCtl.unpause();
        assertFalse(pauseCtl.paused());
    }

    function testEmergencyPauseNonGuardianCannotActivateEmergency() public {
        vm.prank(mallory);
        vm.expectRevert("Not guardian");
        pauseCtl.activateEmergency("panic");
        assertFalse(pauseCtl.emergencyActive());
    }

    function testEmergencyPauseUnpauseBlockedWhileEmergencyActive() public {
        pauseCtl.activateEmergency("panic");
        vm.expectRevert("Emergency active");
        pauseCtl.unpause();
    }

    /// Lifting an emergency is owner-only so a compromised guardian key cannot
    /// undo the freeze.
    function testEmergencyPauseGuardianCannotDeactivateEmergency() public {
        pauseCtl.activateEmergency("panic");
        pauseCtl.setGuardian(alice, true);
        vm.prank(alice);
        vm.expectRevert("Not owner");
        pauseCtl.deactivateEmergency();
        assertTrue(pauseCtl.emergencyActive());
    }

    function testEmergencyPauseOwnerCanDeactivateEmergency() public {
        pauseCtl.activateEmergency("panic");
        pauseCtl.deactivateEmergency();
        assertFalse(pauseCtl.emergencyActive());
    }

    function testEmergencyPauseNonGuardianCannotSetPauseDelay() public {
        vm.prank(mallory);
        vm.expectRevert("Not guardian");
        pauseCtl.setPauseDelay(keccak256("withdraw"), 1 days);
    }

    function testEmergencyPauseOnlyOwnerManagesGuardians() public {
        vm.prank(mallory);
        vm.expectRevert("Not owner");
        pauseCtl.setGuardian(mallory, true);
        assertFalse(pauseCtl.guardians(mallory));
    }

    function testEmergencyPauseGrantedGuardianCanPause() public {
        pauseCtl.setGuardian(alice, true);
        vm.prank(alice);
        pauseCtl.pause("guardian action");
        assertTrue(pauseCtl.paused());
    }

    function testEmergencyPauseRevokedGuardianCannotPause() public {
        pauseCtl.setGuardian(alice, true);
        pauseCtl.setGuardian(alice, false);
        vm.prank(alice);
        vm.expectRevert("Not guardian");
        pauseCtl.pause("revoked");
    }

    function testEmergencyPauseTransferOwnershipAddsGuardian() public {
        pauseCtl.transferOwnership(bob);
        assertEq(pauseCtl.owner(), bob);
        assertTrue(pauseCtl.guardians(bob));
    }

    function testEmergencyPauseCanExecuteFollowsGlobalPause() public {
        bytes32 fnId = keccak256("withdraw");

        // Not paused: executable.
        assertTrue(pauseCtl.canExecute(fnId));

        // A plain pause blocks everything until it is lifted.
        pauseCtl.pause("maintenance");
        assertFalse(pauseCtl.canExecute(fnId));

        pauseCtl.unpause();
        assertTrue(pauseCtl.canExecute(fnId));
    }

    /// An emergency is an explicit override: while it is active canExecute
    /// returns true even though the protocol is paused, so a guardian can still
    /// run the recovery function the emergency was raised for.
    function testEmergencyPauseCanExecuteOverridesGlobalPause() public {
        bytes32 fnId = keccak256("withdraw");
        pauseCtl.pause("maintenance");
        assertFalse(pauseCtl.canExecute(fnId));

        pauseCtl.activateEmergency("incident");
        assertTrue(pauseCtl.paused());
        assertTrue(pauseCtl.canExecute(fnId));
    }

    /// The per-function delay only takes effect once a start timestamp exists
    /// for that function, and nothing in this contract ever writes one, so a
    /// delay set through setPauseDelay never gates on its own.
    function testEmergencyPauseDelayWithoutTimestampDoesNotGate() public {
        bytes32 fnId = keccak256("withdraw");
        pauseCtl.setPauseDelay(fnId, 1 days);
        assertEq(pauseCtl.pauseDelays(fnId), 1 days);
        assertEq(pauseCtl.pauseTimestamps(fnId), 0);
        assertTrue(pauseCtl.canExecute(fnId));
    }

    // =====================================================================
    // EmergencyCouncil
    // =====================================================================

    EmergencyCouncil internal council;

    function setUpCouncil() internal {
        council = new EmergencyCouncil();
    }

    function testCouncilNonMemberCannotPropose() public {
        setUpCouncil();
        vm.prank(mallory);
        vm.expectRevert("Not member");
        council.createProposal("take over", "");
    }

    /// addMember used to be fully public: a caller could join the council,
    /// vote for their own proposal, and execute it alone.
    function testCouncilNonOwnerCannotAddMember() public {
        setUpCouncil();
        vm.prank(mallory);
        vm.expectRevert("Not owner");
        council.addMember(mallory);
        assertFalse(council.councilMembers(mallory));
    }

    function testCouncilNonOwnerCannotRemoveMember() public {
        setUpCouncil();
        vm.prank(mallory);
        vm.expectRevert("Not owner");
        council.removeMember(owner);
        assertTrue(council.councilMembers(owner));
    }

    function testCouncilNonOwnerCannotChangeQuorum() public {
        setUpCouncil();
        vm.prank(mallory);
        vm.expectRevert("Not owner");
        council.setQuorum(1);
        assertEq(council.quorum(), 3);
    }

    function testCouncilVotingAndExecution() public {
        setUpCouncil();
        council.addMember(alice);
        council.addMember(bob);

        uint256 id = council.createProposal("recover funds", "");
        council.vote(id, true);
        vm.prank(alice);
        council.vote(id, true);
        vm.prank(bob);
        council.vote(id, true);

        // Voting is still open, so execution is premature.
        vm.expectRevert("Voting ongoing");
        council.executeProposal(id);

        vm.warp(block.timestamp + 1 days);
        council.executeProposal(id);

        (, , , uint256 votesFor, , , , bool executed, ) = council.proposals(id);
        assertEq(votesFor, 3);
        assertTrue(executed);
    }

    /// A proposal that never reaches quorum cannot be executed, however long it
    /// is left open.
    function testCouncilBelowQuorumCannotExecute() public {
        setUpCouncil();
        council.addMember(alice);

        uint256 id = council.createProposal("recover funds", "");
        council.vote(id, true);
        vm.prank(alice);
        council.vote(id, true);

        vm.warp(block.timestamp + 1 days);
        vm.expectRevert("No quorum");
        council.executeProposal(id);
    }

    function testCouncilDoubleVoteRejected() public {
        setUpCouncil();
        council.addMember(alice);
        uint256 id = council.createProposal("x", "");
        vm.prank(alice);
        council.vote(id, true);
        vm.prank(alice);
        vm.expectRevert("Already voted");
        council.vote(id, true);
    }

    function testCouncilRemovedMemberCannotVote() public {
        setUpCouncil();
        council.addMember(alice);
        council.removeMember(alice);
        vm.prank(alice);
        vm.expectRevert("Not member");
        council.vote(1, true);
    }

    function testCouncilCannotExecuteWhileVotingOngoing() public {
        setUpCouncil();
        council.addMember(alice);
        council.addMember(bob);
        uint256 id = council.createProposal("x", "");
        council.vote(id, true);
        vm.prank(alice);
        council.vote(id, true);
        vm.prank(bob);
        council.vote(id, true);

        vm.expectRevert("Voting ongoing");
        council.executeProposal(id);
    }

    // =====================================================================
    // UpgradeManager
    // =====================================================================

    UpgradeManager internal upgrader;

    function setUpUpgrader() internal {
        upgrader = new UpgradeManager();
    }

    /// proposeUpgrade used to have no guard at all, so anyone could register an
    /// upgrade pointing at an implementation of their choosing.
    function testUpgradeNonGovernorCannotPropose() public {
        setUpUpgrader();
        vm.prank(mallory);
        vm.expectRevert("Not governor");
        upgrader.proposeUpgrade(address(0xDEAD));
    }

    function testUpgradeNonGovernorCannotVote() public {
        setUpUpgrader();
        bytes32 id = upgrader.proposeUpgrade(address(0xDEAD));
        vm.prank(mallory);
        vm.expectRevert("Not governor");
        upgrader.vote(id);
    }

    function testUpgradeNonGovernorCannotExecute() public {
        setUpUpgrader();
        bytes32 id = upgrader.proposeUpgrade(address(0xDEAD));
        vm.warp(block.timestamp + 1 days);
        vm.prank(mallory);
        vm.expectRevert("Not governor");
        upgrader.executeUpgrade(id);
    }

    function testUpgradeNonOwnerCannotSetGovernor() public {
        setUpUpgrader();
        vm.prank(mallory);
        vm.expectRevert("Not owner");
        upgrader.setGovernor(mallory, true);
        assertFalse(upgrader.governors(mallory));
    }

    function testUpgradeNonOwnerCannotSetQuorum() public {
        setUpUpgrader();
        vm.prank(mallory);
        vm.expectRevert("Not owner");
        upgrader.setQuorum(1);
        assertEq(upgrader.quorum(), 3);
    }

    function testUpgradeZeroImplementationRejected() public {
        setUpUpgrader();
        vm.expectRevert("Zero implementation");
        upgrader.proposeUpgrade(address(0));
    }

    function testUpgradeFullLifecycle() public {
        setUpUpgrader();
        upgrader.setGovernor(alice, true);
        upgrader.setGovernor(bob, true);

        bytes32 id = upgrader.proposeUpgrade(address(0xDEAD));
        upgrader.vote(id);
        vm.prank(alice);
        upgrader.vote(id);

        // Below quorum.
        vm.warp(block.timestamp + 1 days);
        vm.expectRevert("No quorum");
        upgrader.executeUpgrade(id);

        vm.prank(bob);
        upgrader.vote(id);
        upgrader.executeUpgrade(id);

        (, , uint256 votes, , bool executed, ) = upgrader.upgrades(id);
        assertEq(votes, 3);
        assertTrue(executed);
    }

    function testUpgradeCannotExecuteBeforeVotingPeriodEnds() public {
        setUpUpgrader();
        upgrader.setGovernor(alice, true);
        upgrader.setGovernor(bob, true);

        bytes32 id = upgrader.proposeUpgrade(address(0xDEAD));
        upgrader.vote(id);
        vm.prank(alice);
        upgrader.vote(id);
        vm.prank(bob);
        upgrader.vote(id);

        vm.expectRevert("Voting ongoing");
        upgrader.executeUpgrade(id);
    }

    function testUpgradeRevokedGovernorCannotVote() public {
        setUpUpgrader();
        upgrader.setGovernor(alice, true);
        upgrader.setGovernor(alice, false);
        bytes32 id = upgrader.proposeUpgrade(address(0xDEAD));
        vm.prank(alice);
        vm.expectRevert("Not governor");
        upgrader.vote(id);
    }

    // =====================================================================
    // InsuranceVault
    // =====================================================================

    InsuranceVault internal vault;

    function setUpVault() internal {
        vault = new InsuranceVault(owner);
    }

    function testVaultDepositAndWithdraw() public {
        setUpVault();
        vault.deposit{value: 10 ether}();
        assertEq(vault.deposits(owner), 10 ether);
        assertEq(vault.totalDeposits(), 10 ether);

        vault.withdraw(4 ether);
        assertEq(vault.deposits(owner), 6 ether);
        assertEq(vault.totalWithdrawals(), 4 ether);
    }

    function testVaultWithdrawRejectsMoreThanDeposited() public {
        setUpVault();
        vault.deposit{value: 1 ether}();
        vm.expectRevert("Insufficient");
        vault.withdraw(2 ether);
    }

    /// Withdrawals used to ignore the pause flag, so a governance freeze did
    /// not actually stop an exit from the fund.
    function testVaultWithdrawBlockedWhilePaused() public {
        setUpVault();
        vault.deposit{value: 5 ether}();
        vault.pause();

        vm.expectRevert("Paused");
        vault.withdraw(1 ether);

        vault.unpause();
        vault.withdraw(1 ether);
        assertEq(vault.deposits(owner), 4 ether);
    }

    function testVaultDepositBlockedWhilePaused() public {
        setUpVault();
        vault.pause();
        vm.expectRevert("Paused");
        vault.deposit{value: 1 ether}();
    }

    function testVaultOnlyGovernanceCanPause() public {
        setUpVault();
        vm.prank(mallory);
        vm.expectRevert("Not governance");
        vault.pause();
        assertFalse(vault.paused());
    }

    function testVaultOnlyGovernanceCanPayCoverage() public {
        setUpVault();
        vault.deposit{value: 10 ether}();
        vm.prank(mallory);
        vm.expectRevert("Not governance");
        vault.payCoverage(mallory, 1 ether);
    }

    function testVaultPayCoverageRespectsMaxCoverage() public {
        setUpVault();
        vault.deposit{value: 10 ether}();
        vault.setMaxCoverage(1 ether);

        vm.expectRevert("Exceeds max");
        vault.payCoverage(alice, 2 ether);

        uint256 before = alice.balance;
        vault.payCoverage(alice, 1 ether);
        assertEq(alice.balance, before + 1 ether);
    }

    function testVaultZeroCoverageRejected() public {
        setUpVault();
        vm.expectRevert("Zero coverage");
        vault.setMaxCoverage(0);
    }

    // =====================================================================
    // TimelockController
    // =====================================================================

    TimelockController internal timelock;

    function setUpTimelock() internal {
        timelock = new TimelockController(1 days);
    }

    function testTimelockNonAdminCannotQueue() public {
        setUpTimelock();
        vm.prank(mallory);
        vm.expectRevert("Not admin");
        timelock.queueTransaction(alice, 0, "");
    }

    function testTimelockNonAdminCannotCancel() public {
        setUpTimelock();
        bytes32 id = timelock.queueTransaction(alice, 0, "");
        vm.prank(mallory);
        vm.expectRevert("Not admin");
        timelock.cancelTransaction(id);
    }

    /// getTransactionHash must return the same hash queueTransaction produced;
    /// the nonce was previously captured after incrementing, so the two never
    /// agreed.
    function testTimelockQueueHashMatchesPrediction() public {
        setUpTimelock();
        bytes32 predicted = timelock.getTransactionHash(alice, 0, "");
        bytes32 queued = timelock.queueTransaction(alice, 0, "");
        assertEq(queued, predicted);
    }

    function testTimelockQueuedNonceMatchesHashCommitment() public {
        setUpTimelock();
        bytes32 id = timelock.queueTransaction(alice, 0, "");
        (, , , uint256 storedNonce, , , ) = timelock.transactions(id);
        assertEq(storedNonce, 0);
        assertEq(timelock.nonces(alice), 1);
    }

    function testTimelockExecuteBeforeDelayReverts() public {
        setUpTimelock();
        bytes32 id = timelock.queueTransaction(alice, 0, "");
        vm.expectRevert("Too early");
        timelock.executeTransaction(id);
    }

    function testTimelockExecuteAfterDelaySucceeds() public {
        setUpTimelock();
        bytes32 id = timelock.queueTransaction(alice, 0, "");
        vm.warp(block.timestamp + 1 days);
        timelock.executeTransaction(id);

        (, , , , , bool executed, ) = timelock.transactions(id);
        assertTrue(executed);
    }

    function testTimelockCannotExecuteTwice() public {
        setUpTimelock();
        bytes32 id = timelock.queueTransaction(alice, 0, "");
        vm.warp(block.timestamp + 1 days);
        timelock.executeTransaction(id);

        vm.expectRevert("Executed");
        timelock.executeTransaction(id);
    }

    function testTimelockCancelledCannotExecute() public {
        setUpTimelock();
        bytes32 id = timelock.queueTransaction(alice, 0, "");
        timelock.cancelTransaction(id);
        vm.warp(block.timestamp + 1 days);
        vm.expectRevert("Cancelled");
        timelock.executeTransaction(id);
    }

    function testTimelockExecutesCallWithValue() public {
        setUpTimelock();
        vm.deal(address(timelock), 5 ether);

        bytes32 id = timelock.queueTransaction(bob, 1 ether, "");
        vm.warp(block.timestamp + 1 days);
        uint256 before = bob.balance;
        timelock.executeTransaction(id);
        assertEq(bob.balance, before + 1 ether);
    }

    function testTimelockUnknownHashReverts() public {
        setUpTimelock();
        vm.expectRevert("Not queued");
        timelock.executeTransaction(keccak256("unknown"));
    }
}
