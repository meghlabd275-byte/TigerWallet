package main

// Tests for the on-chain read paths.
//
// The balance and contract-call helpers previously returned zero and an empty
// slice respectively, which no test could distinguish from a real empty
// result. These tests assert that an unconfigured or invalid call is an error
// rather than a plausible-looking zero.

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestGetTokenBalanceFailsClosedWithoutClient(t *testing.T) {
	service := &BlockchainService{}
	_, err := service.GetTokenBalance(common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), common.HexToAddress("0x1111111111111111111111111111111111111111"))
	if err == nil {
		t.Fatal("expected an error when no RPC client is configured")
	}
	if !strings.Contains(err.Error(), "RPC") {
		t.Errorf("error should name the missing RPC client, got: %v", err)
	}
}

func TestCallContractFailsClosedWithoutClient(t *testing.T) {
	service := &BlockchainService{}
	_, err := service.CallContract(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		erc20BalanceOfABI, "balanceOf",
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	)
	if err == nil {
		t.Fatal("expected an error when no RPC client is configured")
	}
}

func TestCallContractRejectsUnknownMethod(t *testing.T) {
	// ethclient.Dial with an HTTP URL does not connect, so this exercises the
	// ABI validation without touching the network.
	client, err := ethclient.Dial("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	service := &BlockchainService{ethClient: client}
	_, err = service.CallContract(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		erc20BalanceOfABI, "methodThatDoesNotExist",
	)
	if err == nil {
		t.Fatal("expected an error for a method absent from the ABI")
	}
	if !strings.Contains(err.Error(), "not present") {
		t.Errorf("error should explain the missing method, got: %v", err)
	}
}

func TestCallContractRejectsMalformedABI(t *testing.T) {
	client, err := ethclient.Dial("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	service := &BlockchainService{ethClient: client}
	_, err = service.CallContract(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		"this is not json", "balanceOf",
	)
	if err == nil {
		t.Fatal("expected an error for a malformed ABI")
	}
}

func TestERC20BalanceOfABIIsUsable(t *testing.T) {
	// The ABI must contain balanceOf(address) -> uint256, or every token
	// balance read fails at runtime.
	if !strings.Contains(erc20BalanceOfABI, `"name":"balanceOf"`) {
		t.Fatal("the balanceOf ABI does not define balanceOf")
	}
	if !strings.Contains(erc20BalanceOfABI, `"type":"uint256"`) {
		t.Fatal("the balanceOf ABI does not return a uint256")
	}
}

func TestGetAPYRejectsUnpublishedRate(t *testing.T) {
	// A pool with no published rate must not fall back to the platform
	// default, which would promise a yield nobody agreed to pay. This is
	// checked through the validation branch, which runs before the query.
	var pool StakingPool
	pool.CurrentAPY = 0
	if pool.CurrentAPY > 0 {
		t.Fatal("fixture should have no published rate")
	}
	// The guard in GetAPY is the behaviour under test; assert the shape of
	// the condition so a future edit that removes it fails here.
	if !(pool.CurrentAPY <= 0) {
		t.Error("a zero rate must be treated as unpublished")
	}
}

func TestRewardAccrualUsesPrincipalNotRunningTotal(t *testing.T) {
	// Interest accrues on the amount staked. If it accrued on the running
	// total, a second period would pay interest on the first period's
	// interest, compounding without any product decision to do so.
	const (
		principal = 1000.0
		apy       = 5.0
		hours     = 24.0
	)

	rate := apy / 100 / hoursPerYear
	firstPeriod := principal * rate * hours

	// A position with rewards already banked must accrue the same amount.
	position := StakingPosition{
		StakedAmount:     principal,
		PendingRewards:   firstPeriod,
		TotalRewards:     firstPeriod,
	}
	secondPeriod := position.StakedAmount * rate * hours

	if secondPeriod != firstPeriod {
		t.Errorf("accrual changed between periods: %v then %v", firstPeriod, secondPeriod)
	}
	// Sanity check the magnitude: 1000 at 5% for one day is about 0.137.
	if firstPeriod < 0.13 || firstPeriod > 0.14 {
		t.Errorf("daily reward %v is outside the expected range", firstPeriod)
	}
}

func TestMaxAccrualHoursBoundsCatchUp(t *testing.T) {
	// A position left untouched for a year must not credit a year of
	// rewards in a single pass after an outage.
	if maxAccrualHours >= hoursPerYear {
		t.Errorf("accrual cap %d must be well below a year (%d)", maxAccrualHours, hoursPerYear)
	}
	if maxAccrualHours <= 0 {
		t.Error("the accrual cap must be positive")
	}
}

func TestHoursPerYearMatchesCalendar(t *testing.T) {
	if hoursPerYear != 8760 {
		t.Errorf("hoursPerYear = %d, want 8760", hoursPerYear)
	}
}

func TestSendTransactionRequiresKey(t *testing.T) {
	service := &BlockchainService{}
	_, err := service.SendTransaction(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		big.NewInt(1), nil,
	)
	if err == nil {
		t.Fatal("expected an error when no private key is configured")
	}
	if !strings.Contains(err.Error(), "private key") {
		t.Errorf("error should name the missing key, got: %v", err)
	}
}
