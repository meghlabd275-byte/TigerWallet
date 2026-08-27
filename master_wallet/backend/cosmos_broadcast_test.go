package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestCosmosChainMetaAndDecimals(t *testing.T) {
	chainIDStr, denom, decimals := cosmosChainMetaAndDecimals(9000000118) // Cosmos Hub
	if chainIDStr != "cosmoshub-4" || denom != "uatom" {
		t.Fatalf("unexpected meta for Cosmos Hub: %q %q", chainIDStr, denom)
	}
	if decimals != 6 {
		t.Fatalf("expected 6 decimals for ATOM, got %d", decimals)
	}

	_, _, decInj := cosmosChainMetaAndDecimals(9000073068) // Injective (18 decimals)
	if decInj != 18 {
		t.Fatalf("expected 18 decimals for INJ, got %d", decInj)
	}
}

func TestCosmosRegistryDirMapping(t *testing.T) {
	if dir := cosmosRegistryDir["osmosis-1"]; dir != "osmosis" {
		t.Fatalf("osmosis dir = %q", dir)
	}
	if dir := cosmosRegistryDir["cosmoshub-4"]; dir != "cosmoshub" {
		t.Fatalf("cosmoshub dir = %q", dir)
	}
}

func TestCosmosStdTxEnvelope(t *testing.T) {
	fee := cosmosFeeTroika{
		Amount: []map[string]string{{"denom": "uatom", "amount": "5000"}},
		Gas:    "200000",
	}
	msgs := []cosmosMsgTroika{{
		Type: "cosmos-sdk/MsgSend",
		Value: cosmosSendMsgOne{
			Amount:      []map[string]string{{"denom": "uatom", "amount": "1"}},
			FromAddress: "cosmos1sender",
			ToAddress:   "cosmos1receiver",
		},
	}}
	tx := cosmosStdTx{
		Type: "cosmos-sdk/StdTx",
		Value: cosmosStdTxValue{
			Msg:  msgs,
			Fee:  fee,
			Memo: "",
			Signatures: []cosmosStdSignature{{
				PubKey: map[string]string{
					"type":  "tendermint/PubKeySecp256k1",
					"value": base64.StdEncoding.EncodeToString(make([]byte, 33)),
				},
				Signature: base64.StdEncoding.EncodeToString(make([]byte, 64)),
			}},
		},
	}
	raw, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	s := string(raw)
	for _, want := range []string{"cosmos-sdk/StdTx", "cosmos-sdk/MsgSend", "tendermint/PubKeySecp256k1", "from_address", "to_address"} {
		if !strings.Contains(s, want) {
			t.Fatalf("stdTx envelope missing %q: %s", want, s)
		}
	}
}

func TestCosmosHumanToBaseDenom(t *testing.T) {
	v, err := cosmosHumanToBaseDenom("1.5", 6)
	if err != nil || v != "1500000" {
		t.Fatalf("1.5 ATOM (6 dec) = 1500000, got %q err=%v", v, err)
	}
	v2, err := cosmosHumanToBaseDenom("0.000001", 6)
	if err != nil || v2 != "1" {
		t.Fatalf("0.000001 ATOM = 1, got %q err=%v", v2, err)
	}
}
