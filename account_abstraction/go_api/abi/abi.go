// Package abi implements the ERC-4337 user operation encoding.
//
// The EntryPoint packs the gas fields into two bytes32 words (v0.7) or keeps
// them as separate words (v0.6), and hashes the operation over the packed form.
// The off-chain encoder has to match exactly, or the hash a client signs will
// not be the hash the contract computes and verification will fail.
package abi

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EntryPointV06 is the canonical ERC-4337 v0.6 EntryPoint address, deployed on
// every major network at the same address.
var EntryPointV06 = common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789")

// EntryPointV07 is the canonical ERC-4337 v0.7 EntryPoint address.
var EntryPointV07 = common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032")

// EntryPointV08 is the canonical ERC-4337 v0.8 EntryPoint address.
var EntryPointV08 = common.HexToAddress("0x4337084D9E255Ff0702461CF8895CE9E3b5Ff108")

// UserOperation holds the fields of a user operation.
type UserOperation struct {
	Sender               common.Address
	Nonce                *big.Int
	InitCode             []byte
	CallData             []byte
	CallGasLimit         *big.Int
	VerificationGasLimit *big.Int
	PreVerificationGas   *big.Int
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	PaymasterAndData     []byte
	Signature            []byte
}

// word encodes a big integer as a 32-byte big-endian word.
func word(v *big.Int) []byte {
	out := make([]byte, 32)
	if v == nil || v.Sign() == 0 {
		return out
	}

	b := v.Bytes()
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(out[32-len(b):], b)
	return out
}

// wordUint128 encodes a value as the low half of a 32-byte word.
func wordUint128(v *big.Int) []byte {
	out := make([]byte, 32)
	if v == nil || v.Sign() == 0 {
		return out
	}

	b := v.Bytes()
	if len(b) > 16 {
		b = b[len(b)-16:]
	}
	copy(out[32-len(b):], b)
	return out
}

// addressWord encodes an address as a 32-byte word.
func addressWord(a common.Address) []byte {
	return word(new(big.Int).SetBytes(a.Bytes()))
}

// packUint128Pair packs two uint128 values, high first.
func packUint128Pair(high, low *big.Int) []byte {
	out := make([]byte, 32)
	copy(out[:16], wordUint128(high)[16:])
	copy(out[16:], wordUint128(low)[16:])
	return out
}

// encodeDynamicBytes encodes a bytes value as a length word followed by the
// data, right-padded to a whole number of words.
func encodeDynamicBytes(data []byte) []byte {
	out := make([]byte, 0, 32+((len(data)+31)/32)*32)
	out = append(out, word(new(big.Int).SetUint64(uint64(len(data))))...)
	out = append(out, data...)

	padding := (32 - len(data)%32) % 32
	out = append(out, make([]byte, padding)...)
	return out
}

// packV07 builds the v0.7 inner hash preimage:
//
//	keccak256(abi.encode(sender, nonce, keccak(initCode), keccak(callData),
//	          accountGasLimits, preVerificationGas, gasFees,
//	          keccak(paymasterAndData)))
func packV07(op *UserOperation) []byte {
	accountGasLimits := packUint128Pair(op.VerificationGasLimit, op.CallGasLimit)
	gasFees := packUint128Pair(op.MaxPriorityFeePerGas, op.MaxFeePerGas)

	var inner []byte
	inner = append(inner, addressWord(op.Sender)...)
	inner = append(inner, word(op.Nonce)...)
	inner = append(inner, crypto.Keccak256(op.InitCode)...)
	inner = append(inner, crypto.Keccak256(op.CallData)...)
	inner = append(inner, accountGasLimits...)
	inner = append(inner, word(op.PreVerificationGas)...)
	inner = append(inner, gasFees...)
	inner = append(inner, crypto.Keccak256(op.PaymasterAndData)...)

	return inner
}

// packV06 builds the v0.6 inner hash preimage, where every gas field keeps its
// own word:
//
//	keccak256(abi.encode(sender, nonce, keccak(initCode), keccak(callData),
//	          callGasLimit, verificationGasLimit, preVerificationGas,
//	          maxFeePerGas, maxPriorityFeePerGas, keccak(paymasterAndData)))
func packV06(op *UserOperation) []byte {
	var inner []byte
	inner = append(inner, addressWord(op.Sender)...)
	inner = append(inner, word(op.Nonce)...)
	inner = append(inner, crypto.Keccak256(op.InitCode)...)
	inner = append(inner, crypto.Keccak256(op.CallData)...)
	inner = append(inner, word(op.CallGasLimit)...)
	inner = append(inner, word(op.VerificationGasLimit)...)
	inner = append(inner, word(op.PreVerificationGas)...)
	inner = append(inner, word(op.MaxFeePerGas)...)
	inner = append(inner, word(op.MaxPriorityFeePerGas)...)
	inner = append(inner, crypto.Keccak256(op.PaymasterAndData)...)

	return inner
}

// UserOpHashV07 computes the v0.7 user operation hash:
// keccak256(abi.encode(keccak256(packedFields), entryPoint, chainID)).
func UserOpHashV07(op *UserOperation, entryPoint common.Address, chainID *big.Int) common.Hash {
	var outer []byte
	outer = append(outer, crypto.Keccak256(packV07(op))...)
	outer = append(outer, addressWord(entryPoint)...)
	outer = append(outer, word(chainID)...)

	return common.BytesToHash(crypto.Keccak256(outer))
}

// UserOpHashV06 computes the v0.6 user operation hash. The final hashing step
// is the same as v0.7; only the packed fields differ.
func UserOpHashV06(op *UserOperation, entryPoint common.Address, chainID *big.Int) common.Hash {
	var outer []byte
	outer = append(outer, crypto.Keccak256(packV06(op))...)
	outer = append(outer, addressWord(entryPoint)...)
	outer = append(outer, word(chainID)...)

	return common.BytesToHash(crypto.Keccak256(outer))
}

// Selectors, as keccak256 of the canonical signatures.
var (
	// GetUserOpHashV07Selector is getUserOpHash for the v0.7 tuple.
	GetUserOpHashV07Selector = crypto.Keccak256([]byte(
		"getUserOpHash((address,uint256,bytes,bytes,bytes32,uint256,bytes32,bytes,bytes))"))[:4]

	// GetUserOpHashV06Selector is getUserOpHash for the v0.6 tuple.
	GetUserOpHashV06Selector = crypto.Keccak256([]byte(
		"getUserOpHash((address,uint256,bytes,bytes,uint256,uint256,uint256,uint256,uint256,bytes,bytes))"))[:4]

	// HandleOpsV07Selector is handleOps for the v0.7 tuple array.
	HandleOpsV07Selector = crypto.Keccak256([]byte(
		"handleOps((address,uint256,bytes,bytes,bytes32,uint256,bytes32,bytes,bytes)[],address)"))[:4]

	// HandleOpsV06Selector is handleOps for the v0.6 tuple array.
	HandleOpsV06Selector = crypto.Keccak256([]byte(
		"handleOps((address,uint256,bytes,bytes,uint256,uint256,uint256,uint256,uint256,bytes,bytes)[],address)"))[:4]
)

// encodeOpTupleV07 encodes one v0.7 packed user operation as an ABI tuple. A
// tuple with dynamic members is a head of nine words followed by the dynamic
// contents, with each offset measured from the start of the tuple.
func encodeOpTupleV07(op *UserOperation) []byte {
	const headWords = 9
	headLen := headWords * 32

	initCode := encodeDynamicBytes(op.InitCode)
	callData := encodeDynamicBytes(op.CallData)
	paymasterAndData := encodeDynamicBytes(op.PaymasterAndData)
	signature := encodeDynamicBytes(op.Signature)

	out := make([]byte, 0, headLen+len(initCode)+len(callData)+len(paymasterAndData)+len(signature))

	out = append(out, addressWord(op.Sender)...)
	out = append(out, word(op.Nonce)...)

	offset := headLen
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)
	offset += len(initCode)
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)
	offset += len(callData)
	out = append(out, packUint128Pair(op.VerificationGasLimit, op.CallGasLimit)...)
	out = append(out, word(op.PreVerificationGas)...)
	out = append(out, packUint128Pair(op.MaxPriorityFeePerGas, op.MaxFeePerGas)...)
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)
	offset += len(paymasterAndData)
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)

	out = append(out, initCode...)
	out = append(out, callData...)
	out = append(out, paymasterAndData...)
	out = append(out, signature...)

	return out
}

// encodeOpTupleV06 encodes one v0.6 user operation as an ABI tuple.
func encodeOpTupleV06(op *UserOperation) []byte {
	const headWords = 11
	headLen := headWords * 32

	initCode := encodeDynamicBytes(op.InitCode)
	callData := encodeDynamicBytes(op.CallData)
	paymasterAndData := encodeDynamicBytes(op.PaymasterAndData)
	signature := encodeDynamicBytes(op.Signature)

	out := make([]byte, 0, headLen+len(initCode)+len(callData)+len(paymasterAndData)+len(signature))

	out = append(out, addressWord(op.Sender)...)
	out = append(out, word(op.Nonce)...)

	offset := headLen
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)
	offset += len(initCode)
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)
	offset += len(callData)

	out = append(out, word(op.CallGasLimit)...)
	out = append(out, word(op.VerificationGasLimit)...)
	out = append(out, word(op.PreVerificationGas)...)
	out = append(out, word(op.MaxFeePerGas)...)
	out = append(out, word(op.MaxPriorityFeePerGas)...)

	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)
	offset += len(paymasterAndData)
	out = append(out, word(new(big.Int).SetUint64(uint64(offset)))...)

	out = append(out, initCode...)
	out = append(out, callData...)
	out = append(out, paymasterAndData...)
	out = append(out, signature...)

	return out
}

// encodeTupleArray encodes an array of encoded tuples as a dynamic ABI
// parameter body: a length word followed by one offset per element and then the
// element contents.
func encodeTupleArray(tuples [][]byte) []byte {
	body := word(new(big.Int).SetUint64(uint64(len(tuples))))

	elementOffset := len(tuples) * 32
	for _, tuple := range tuples {
		body = append(body, word(new(big.Int).SetUint64(uint64(elementOffset)))...)
		elementOffset += len(tuple)
	}
	for _, tuple := range tuples {
		body = append(body, tuple...)
	}

	return body
}

// encodeHandleOps builds calldata for handleOps(tuple[] ops, address beneficiary).
func encodeHandleOps(tuples [][]byte, selector []byte, beneficiary common.Address) ([]byte, error) {
	if len(tuples) == 0 {
		return nil, fmt.Errorf("handleOps requires at least one operation")
	}

	body := encodeTupleArray(tuples)

	out := make([]byte, 0, 4+64+len(body))
	out = append(out, selector...)
	// The array is a dynamic parameter, so the head holds its offset. With two
	// parameters the head is two words, placing the array at 0x40.
	out = append(out, word(new(big.Int).SetUint64(64))...)
	out = append(out, addressWord(beneficiary)...)
	out = append(out, body...)

	return out, nil
}

// EncodeHandleOpsV07 builds v0.7 handleOps calldata.
func EncodeHandleOpsV07(ops []*UserOperation, beneficiary common.Address) ([]byte, error) {
	tuples := make([][]byte, len(ops))
	for i, op := range ops {
		if op == nil {
			return nil, fmt.Errorf("operation %d is nil", i)
		}
		tuples[i] = encodeOpTupleV07(op)
	}

	return encodeHandleOps(tuples, HandleOpsV07Selector, beneficiary)
}

// EncodeHandleOpsV06 builds v0.6 handleOps calldata.
func EncodeHandleOpsV06(ops []*UserOperation, beneficiary common.Address) ([]byte, error) {
	tuples := make([][]byte, len(ops))
	for i, op := range ops {
		if op == nil {
			return nil, fmt.Errorf("operation %d is nil", i)
		}
		tuples[i] = encodeOpTupleV06(op)
	}

	return encodeHandleOps(tuples, HandleOpsV06Selector, beneficiary)
}
