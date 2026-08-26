package main

// solana_broadcast.go — real Solana SystemProgram.transfer construction,
// Ed25519 signing, and broadcast via JSON-RPC. No fakes: the message is built
// from the canonical legacy layout, the blockhash is fetched from the live
// RPC, and the signed transaction is submitted with sendTransaction.

import (
        "bytes"
        "crypto/ed25519"
        "encoding/base64"
        "encoding/binary"
        "encoding/json"
        "fmt"
        "io"
        "math/big"
        "net/http"
        "time"
)

// solanaSystemProgram is the Solana System program id as base58.
const solanaSystemProgram = "11111111111111111111111111111111"

// solanaRPCCall performs a single JSON-RPC 2.0 call and writes the JSON result
// into out (if non-nil). Returns the raw result bytes.
func solanaRPCCall(rpcURL, method string, params []any, out any) error {
        body, err := json.Marshal(map[string]any{
                "jsonrpc": "2.0",
                "id":      1,
                "method":  method,
                "params":  params,
        })
        if err != nil {
                return err
        }
        req, err := http.NewRequest(http.MethodPost, rpcURL, bytes.NewReader(body))
        if err != nil {
                return err
        }
        req.Header.Set("Content-Type", "application/json")
        client := &http.Client{Timeout: 20 * time.Second}
        resp, err := client.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()
        raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
        if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("solana rpc HTTP %d: %s", resp.StatusCode, string(raw))
        }
        var envelope struct {
                Error *json.RawMessage `json:"error"`
                Result json.RawMessage `json:"result"`
        }
        if err := json.Unmarshal(raw, &envelope); err != nil {
                return err
        }
        if envelope.Error != nil {
                return fmt.Errorf("solana rpc error: %s", string(*envelope.Error))
        }
        if out != nil {
                if err := json.Unmarshal(envelope.Result, out); err != nil {
                        return err
                }
        }
        return nil
}

// fetchSolanaBlockhash returns the current blockhash from getLatestBlockhash.
func fetchSolanaBlockhash(rpcURL string) ([]byte, error) {
        var out struct {
                Value struct {
                        Blockhash string `json:"blockhash"`
                } `json:"value"`
        }
        if err := solanaRPCCall(rpcURL, "getLatestBlockhash", []any{}, &out); err != nil {
                return nil, err
        }
        if out.Value.Blockhash == "" {
                return nil, fmt.Errorf("empty blockhash from solana rpc")
        }
        raw, err := base58DecodeStrict(out.Value.Blockhash)
        if err != nil {
                return nil, fmt.Errorf("decode blockhash: %w", err)
        }
        if len(raw) != 32 {
                return nil, fmt.Errorf("invalid solana blockhash length %d", len(raw))
        }
        return raw, nil
}

// base58DecodeStrict decodes a base58 string, returning an error on invalid
// characters (unlike the lenient base58Decode helper used elsewhere).
func base58DecodeStrict(s string) ([]byte, error) {
        const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
        if s == "" {
                return []byte{}, nil
        }
        x := new(big.Int)
        base := big.NewInt(58)
        for i := 0; i < len(s); i++ {
                idx := bytes.IndexByte([]byte(alphabet), s[i])
                if idx < 0 {
                        return nil, fmt.Errorf("invalid base58 character %q", s[i])
                }
                x.Mul(x, base)
                x.Add(x, big.NewInt(int64(idx)))
        }
        decoded := x.Bytes()
        // Recover leading zero bytes ('1').
        nLeading := 0
        for nLeading < len(s) && s[nLeading] == '1' {
                nLeading++
        }
        out := make([]byte, nLeading+len(decoded))
        copy(out[nLeading:], decoded)
        return out, nil
}

// appendCompactU16 appends a compact-u16 encoded length.
func appendCompactU16(buf []byte, v uint16) []byte {
        switch {
        case v < 0x80:
                return append(buf, byte(v))
        case v < 0x4000:
                return append(buf, byte(v)|0x80, byte(v>>7))
        default:
                return append(buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14))
        }
}

// buildSolanaTransferMessage builds the unsigned legacy Solana message for a
// SystemProgram.transfer. Returns the message bytes (the exact bytes Ed25519
// signs).
func buildSolanaTransferMessage(fromPub, toPub ed25519.PublicKey, lamports uint64, blockhash []byte) ([]byte, error) {
        if len(fromPub) != ed25519.PublicKeySize {
                return nil, fmt.Errorf("invalid solana from pubkey length %d", len(fromPub))
        }
        if len(toPub) != ed25519.PublicKeySize {
                return nil, fmt.Errorf("invalid solana to pubkey length %d", len(toPub))
        }
        if len(blockhash) != 32 {
                return nil, fmt.Errorf("invalid solana blockhash length %d", len(blockhash))
        }
        sysPub, err := base58DecodeStrict(solanaSystemProgram)
        if err != nil {
                return nil, err
        }
        if len(sysPub) != ed25519.PublicKeySize {
                return nil, fmt.Errorf("invalid system program id length %d", len(sysPub))
        }

        var msg bytes.Buffer

        // Message header: num_required_signatures, num_readonly_signed, num_readonly_unsigned.
        msg.Write([]byte{1, 0, 1})

        // account_keys: [from (signer+writable), to (writable), system program (readonly)].
        msg.Write(appendCompactU16(nil, 3))
        msg.Write(fromPub)
        msg.Write(toPub)
        msg.Write(sysPub)

        // recent_blockhash (32 bytes).
        msg.Write(blockhash)

        // instructions: one SystemProgram.transfer.
        msg.Write(appendCompactU16(nil, 1))
        msg.WriteByte(2) // program_id_index = system program (index 2 in account_keys)
        // account indexes: from(0), to(1).
        msg.Write(appendCompactU16(nil, 2))
        msg.WriteByte(0)
        msg.WriteByte(1)
        // instruction data: u32 discriminator (2 = Transfer) + u64 lamports LE.
        var data bytes.Buffer
        var disc [4]byte
        binary.LittleEndian.PutUint32(disc[:], 2)
        data.Write(disc[:])
        var lam [8]byte
        binary.LittleEndian.PutUint64(lam[:], lamports)
        data.Write(lam[:])
        msg.Write(appendCompactU16(nil, uint16(data.Len())))
        msg.Write(data.Bytes())

        return msg.Bytes(), nil
}

// mwSolanaTransfer builds, signs, and broadcasts a real Solana
// SystemProgram.transfer. It returns the transaction id from sendTransaction.
func mwSolanaTransfer(seed []byte, derivationPath, toAddress, valueStr, rpcURL string) (string, error) {
        if rpcURL == "" {
                return "", fmt.Errorf("no solana rpc url")
        }
        privKeyBytes, err := slip10DeriveEd25519MW(seed, derivationPath)
        if err != nil {
                return "", fmt.Errorf("solana key derivation: %w", err)
        }
        privKey := ed25519.NewKeyFromSeed(privKeyBytes)
        fromPub := privKey.Public().(ed25519.PublicKey)

        toPubRaw, err := base58DecodeStrict(toAddress)
        if err != nil {
                return "", fmt.Errorf("invalid solana to address: %w", err)
        }
        if len(toPubRaw) != ed25519.PublicKeySize {
                return "", fmt.Errorf("invalid solana to address length %d", len(toPubRaw))
        }
        toPub := ed25519.PublicKey(toPubRaw)

        lamports, ok := new(big.Int).SetString(valueStr, 10)
        if !ok || lamports.Sign() < 0 {
                return "", fmt.Errorf("invalid solana lamports value %q", valueStr)
        }
        if !lamports.IsUint64() {
                return "", fmt.Errorf("solana lamports overflow %q", valueStr)
        }

        blockhash, err := fetchSolanaBlockhash(rpcURL)
        if err != nil {
                return "", fmt.Errorf("solana blockhash: %w", err)
        }
        msgBytes, err := buildSolanaTransferMessage(fromPub, toPub, lamports.Uint64(), blockhash)
        if err != nil {
                return "", err
        }
        sig := ed25519.Sign(privKey, msgBytes)

        // Assemble the signed transaction: compact-u16 signature count (1),
        // 64-byte signature, then the serialized message.
        var tx bytes.Buffer
        tx.Write(appendCompactU16(nil, 1))
        tx.Write(sig)
        tx.Write(msgBytes)

        encoded := base64.StdEncoding.EncodeToString(tx.Bytes())
        var txid string
        if err := solanaRPCCall(rpcURL, "sendTransaction", []any{encoded, map[string]any{"preflightCommitment": "confirmed"}}, &txid); err != nil {
                return "", fmt.Errorf("solana broadcast: %w", err)
        }
        if txid == "" {
                return "", fmt.Errorf("solana broadcast returned empty txid")
        }
        return txid, nil
}