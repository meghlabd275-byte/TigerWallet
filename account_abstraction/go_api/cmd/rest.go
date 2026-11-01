// REST surface for the ERC-4337 bundler.
//
// The bundler speaks JSON-RPC natively (see handleRPC), but the wallet front end
// drives it through a small REST facade. This file implements that facade by
// translating each REST call into the same internal handler the JSON-RPC path
// uses, so there is exactly one implementation of every operation.
//
// Everything here reads from or writes to the chain. Counterfactual account
// addresses come from the deployed factory's own `getAddress`/`getAccountAddress`
// view, deployment state comes from `eth_getCode`, balances come from the node,
// and paymaster approvals are signed with the configured sponsor key over the
// EntryPoint-computed user operation hash. Nothing is synthesised: a missing
// dependency is an error, never a placeholder value.
package main

import (
        "context"
        "crypto/ecdsa"
        "encoding/json"
        "errors"
        "fmt"
        "math/big"
        "net/http"
        "os"
        "strconv"
        "strings"
        "time"

        "github.com/ethereum/go-ethereum"
        "github.com/ethereum/go-ethereum/accounts/abi"
        "github.com/ethereum/go-ethereum/common"
        "github.com/ethereum/go-ethereum/common/hexutil"
        "github.com/ethereum/go-ethereum/crypto"
        "github.com/gin-gonic/gin"
)

// Factory flavours. The canonical `SimpleAccountFactory` ships with the
// EntryPoint and exposes `getAddress`; TigerWallet's own factory exposes
// `getAccountAddress`. Both deploy through `createAccount(address,uint256)`.
const (
        factoryModeCanonical   = "canonical"
        factoryModeTigerWallet = "tigerwallet"
)

// paymasterGasLimits are the per-user-operation gas allowances advertised in
// `paymasterAndData` when requesting sponsorship.
const (
        defaultPaymasterVerificationGas = uint64(150_000)
        defaultPaymasterPostOpGas       = uint64(50_000)
        defaultSponsorshipWindow        = time.Hour
)

var (
        addressType abi.Type
        uint256Type abi.Type
)

func init() {
        addressType, _ = abi.NewType("address", "", nil)
        uint256Type, _ = abi.NewType("uint256", "", nil)
}

// methodSelector returns the 4-byte selector for a canonical ABI signature.
func methodSelector(signature string) []byte {
        return crypto.Keccak256([]byte(signature))[:4]
}

// packArgs ABI-encodes the given values against their types.
func packArgs(values ...interface{}) ([]byte, error) {
        arguments := make(abi.Arguments, 0, len(values))
        for _, value := range values {
                var argType abi.Type
                switch value.(type) {
                case common.Address:
                        argType = addressType
                case *big.Int:
                        argType = uint256Type
                default:
                        return nil, fmt.Errorf("unsupported ABI argument type %T", value)
                }
                arguments = append(arguments, abi.Argument{Type: argType})
        }
        return arguments.Pack(values...)
}

// encodeCall builds the calldata for a method call.
func encodeCall(signature string, values ...interface{}) ([]byte, error) {
        payload, err := packArgs(values...)
        if err != nil {
                return nil, err
        }
        return append(methodSelector(signature), payload...), nil
}

// factoryConfig describes the account factory a deployment uses.
type factoryConfig struct {
        Address common.Address
        Mode    string
}

// Factory returns the configured account factory.
func (api *BundlerAPI) Factory() *factoryConfig {
        if api.config.AccountFactory == (common.Address{}) {
                return nil
        }
        return &factoryConfig{Address: api.config.AccountFactory, Mode: api.config.FactoryMode}
}

// addressMethod is the view that returns the counterfactual account address.
func (f *factoryConfig) addressMethod() string {
        if f.Mode == factoryModeTigerWallet {
                return "getAccountAddress(address,uint256)"
        }
        return "getAddress(address,uint256)"
}

// counterfactualAddress asks the deployed factory for the account address that
// `owner` with `salt` will occupy. The factory computes it deterministically
// from the account implementation address and the creation code hash, so this
// is the authoritative answer rather than a locally reimplemented prediction.
func (api *BundlerAPI) counterfactualAddress(ctx context.Context, owner common.Address, salt *big.Int) (common.Address, error) {
        factory := api.Factory()
        if factory == nil {
                return common.Address{}, errors.New("ACCOUNT_FACTORY is not configured")
        }

        data, err := encodeCall(factory.addressMethod(), owner, salt)
        if err != nil {
                return common.Address{}, fmt.Errorf("cannot encode the factory call: %w", err)
        }

        result, err := api.ethCall(ctx, factory.Address, data)
        if err != nil {
                return common.Address{}, err
        }

        if len(result) < 32 {
                return common.Address{}, errors.New("the factory returned no address")
        }

        return common.BytesToAddress(result[12:32]), nil
}

// ethCall performs a read-only contract call.
func (api *BundlerAPI) ethCall(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
        message := ethereum.CallMsg{To: &to, Data: data}
        result, err := api.client.CallContract(ctx, message, nil)
        if err != nil {
                return nil, fmt.Errorf("the call to %s failed: %w", to.Hex(), err)
        }
        return result, nil
}

// isDeployed reports whether the account has code on chain.
func (api *BundlerAPI) isDeployed(ctx context.Context, account common.Address) (bool, error) {
        code, err := api.client.CodeAt(ctx, account, nil)
        if err != nil {
                return false, fmt.Errorf("cannot read the account code: %w", err)
        }
        return len(code) > 0, nil
}

// entryPointNonce reads the account's nonce from the EntryPoint.
func (api *BundlerAPI) entryPointNonce(ctx context.Context, sender common.Address) (*big.Int, error) {
        data, err := encodeCall("getNonce(address,uint192)", sender, big.NewInt(0))
        if err != nil {
                return nil, err
        }

        result, err := api.ethCall(ctx, api.config.EntryPoint, data)
        if err != nil {
                return nil, err
        }

        if len(result) < 32 {
                return nil, errors.New("the entry point returned no nonce")
        }

        return new(big.Int).SetBytes(result[:32]), nil
}

// ---------------------------------------------------------------------------
// REST handlers
// ---------------------------------------------------------------------------

// smartAccountView is the account state returned to the wallet front end.
type smartAccountView struct {
        Address    string `json:"address"`
        Owner      string `json:"owner"`
        Factory    string `json:"factory"`
        Nonce      string `json:"nonce"`
        IsDeployed bool   `json:"isDeployed"`
        Balance    string `json:"balance"`
}

// handleListEntryPoints serves GET /chains/:chainId/entry-points.
//
// The bundler serves one chain, so the path segment is checked against the
// node's own chain id rather than ignored.
func (api *BundlerAPI) handleListEntryPoints(c *gin.Context) {
        requested := strings.TrimSpace(c.Param("chainId"))

        ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
        defer cancel()

        chainID, err := api.client.ChainID(ctx)
        if err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("cannot read the chain id: %v", err)})
                return
        }

        if requested != "" {
                parsed, err := strconv.ParseUint(requested, 10, 64)
                if err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": "chainId must be a decimal integer"})
                        return
                }
                if parsed != chainID.Uint64() {
                        c.JSON(http.StatusNotFound, gin.H{
                                "error": fmt.Sprintf("this bundler serves chain %s, not %d", chainID.String(), parsed),
                        })
                        return
                }
        }

        c.JSON(http.StatusOK, []string{api.config.EntryPoint.Hex()})
}

// handleCreateSmartAccount serves POST /wallet.
//
// It returns the counterfactual account for the owner. Deployment itself happens
// on the first user operation: the EntryPoint reads `initCode` and calls the
// factory, which is how ERC-4337 accounts are created. The bundler never holds
// the owner's key, so it cannot and does not deploy eagerly.
func (api *BundlerAPI) handleCreateSmartAccount(c *gin.Context) {
        var request struct {
                Owner string `json:"owner"`
                Salt  string `json:"salt"`
        }
        if err := c.ShouldBindJSON(&request); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "the request body is malformed"})
                return
        }

        if !common.IsHexAddress(request.Owner) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "owner must be a hex address"})
                return
        }
        owner := common.HexToAddress(request.Owner)

        salt, err := parseOptionalUint(request.Salt)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
        defer cancel()

        account, err := api.counterfactualAddress(ctx, owner, salt)
        if err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
                return
        }

        view, err := api.accountView(ctx, account, owner)
        if err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, view)
}

// handleGetSmartAccount serves GET /wallet/:owner.
//
// The path segment is the account *owner*, matching how the wallet front end
// calls it: the counterfactual account for that owner (salt 0) is returned.
func (api *BundlerAPI) handleGetSmartAccount(c *gin.Context) {
        raw := strings.TrimSpace(c.Param("owner"))
        if !common.IsHexAddress(raw) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "owner must be a hex address"})
                return
        }
        owner := common.HexToAddress(raw)

        salt, err := parseOptionalUint(c.Query("salt"))
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
        defer cancel()

        account, err := api.counterfactualAddress(ctx, owner, salt)
        if err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
                return
        }

        view, err := api.accountView(ctx, account, owner)
        if err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, view)
}

// accountView assembles the on-chain state of an account.
func (api *BundlerAPI) accountView(ctx context.Context, account, owner common.Address) (*smartAccountView, error) {
        factory := api.Factory()

        balance, err := api.client.BalanceAt(ctx, account, nil)
        if err != nil {
                return nil, fmt.Errorf("cannot read the account balance: %w", err)
        }

        nonce := new(big.Int)
        if deployed, err := api.isDeployed(ctx, account); err != nil {
                return nil, err
        } else if deployed {
                nonce, err = api.entryPointNonce(ctx, account)
                if err != nil {
                        return nil, err
                }
        }

        deployed, err := api.isDeployed(ctx, account)
        if err != nil {
                return nil, err
        }

        factoryAddress := ""
        if factory != nil {
                factoryAddress = factory.Address.Hex()
        }

        return &smartAccountView{
                Address:    account.Hex(),
                Owner:      owner.Hex(),
                Factory:    factoryAddress,
                Nonce:      nonce.String(),
                IsDeployed: deployed,
                Balance:    balance.String(),
        }, nil
}

// handlePaymasterSponsorship serves POST /paymaster/sponsorship.
//
// It signs the EntryPoint-computed user operation hash with the configured
// sponsor key and returns the assembled `paymasterAndData` blob. Without a
// configured paymaster address and signer this fails closed; it never returns an
// approval the paymaster would reject.
func (api *BundlerAPI) handlePaymasterSponsorship(c *gin.Context) {
        var op UserOperation
        if err := c.ShouldBindJSON(&op); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "the user operation is malformed"})
                return
        }

        if api.config.PaymasterAddress == (common.Address{}) {
                c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PAYMASTER_ADDRESS is not configured"})
                return
        }
        if api.paymasterSigner == nil {
                c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PAYMASTER_SIGNER_KEY is not configured"})
                return
        }

        abiOp, err := op.toABI()
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
        defer cancel()

        chainID, err := api.client.ChainID(ctx)
        if err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("cannot read the chain id: %v", err)})
                return
        }

        userOpHash := api.userOpHash(abiOp, chainID)

        // The paymaster validates over the EIP-191 prefixed hash, so the
        // signature must be produced over the same digest.
        digest := crypto.Keccak256(
                []byte("\x19Ethereum Signed Message:\n32"),
                userOpHash.Bytes(),
        )

        signature, err := crypto.Sign(digest, api.paymasterSigner)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot sign the sponsorship"})
                return
        }
        // crypto.Sign returns v in {0,1}; the paymaster recovers through
        // ECDSA.recover, which accepts both forms, but 27/28 is the canonical
        // Ethereum encoding.
        signature[64] += 27

        now := uint64(time.Now().Unix())
        validUntil := now + uint64(api.config.SponsorshipWindow.Seconds())

        paymasterData := make([]byte, 0, 20+16+16+65+6+6)
        paymasterData = append(paymasterData, api.config.PaymasterAddress.Bytes()...)
        paymasterData = append(paymasterData, leftPadUint128(api.config.PaymasterVerificationGas)...)
        paymasterData = append(paymasterData, leftPadUint128(api.config.PaymasterPostOpGas)...)
        paymasterData = append(paymasterData, signature...)
        paymasterData = append(paymasterData, uint48Bytes(validUntil)...)
        paymasterData = append(paymasterData, uint48Bytes(now)...)

        c.JSON(http.StatusOK, gin.H{
                "enabled":        true,
                "sponsorAddress": api.config.PaymasterAddress.Hex(),
                "validUntil":     validUntil,
                "signature":      hexutil.Encode(paymasterData),
        })
}

// ---------------------------------------------------------------------------
// REST aliases for the JSON-RPC methods
// ---------------------------------------------------------------------------

// handleRESTSendUserOperation serves POST /rpc/eth_sendUserOperation.
//
// The body is a bare user operation; it is wrapped into the parameter array the
// JSON-RPC handler expects so both paths share one implementation.
func (api *BundlerAPI) handleRESTSendUserOperation(c *gin.Context) {
        api.dispatchREST(c, "eth_sendUserOperation")
}

// handleRESTEstimateGas serves POST /rpc/eth_estimateGas.
func (api *BundlerAPI) handleRESTEstimateGas(c *gin.Context) {
        api.dispatchREST(c, "eth_estimateUserOperationGas")
}

// handleRESTGetUserOperationByHash serves POST /rpc/eth_getUserOperationByHash.
func (api *BundlerAPI) handleRESTGetUserOperationByHash(c *gin.Context) {
        api.dispatchREST(c, "eth_getUserOperationByHash")
}

// handleRESTGetUserOperationReceipt serves
// GET|POST /rpc/eth_getUserOperationReceipt/:hash.
func (api *BundlerAPI) handleRESTGetUserOperationReceipt(c *gin.Context) {
        hash := c.Param("hash")
        if hash == "" {
                var body struct {
                        Hash string `json:"hash"`
                }
                _ = c.ShouldBindJSON(&body)
                hash = body.Hash
        }
        if strings.TrimSpace(hash) == "" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "a user operation hash is required"})
                return
        }

        api.serveMethod(c, "eth_getUserOperationReceipt", []json.RawMessage{mustJSON(hash)})
}

// dispatchREST reads a bare-object body and forwards it as a single parameter.
func (api *BundlerAPI) dispatchREST(c *gin.Context, method string) {
        var raw json.RawMessage
        if err := c.ShouldBindJSON(&raw); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "the request body is malformed"})
                return
        }

        api.serveMethod(c, method, []json.RawMessage{raw})
}

// mustJSON marshals a value that is already known to be serialisable.
func mustJSON(value interface{}) json.RawMessage {
        encoded, err := json.Marshal(value)
        if err != nil {
                return json.RawMessage("null")
        }
        return encoded
}

// parseOptionalUint reads a decimal or 0x-prefixed quantity, defaulting to zero.
func parseOptionalUint(value string) (*big.Int, error) {
        value = strings.TrimSpace(value)
        if value == "" {
                return big.NewInt(0), nil
        }

        if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
                parsed, err := hexutil.DecodeBig(value)
                if err != nil {
                        return nil, errors.New("the value is not a valid hex quantity")
                }
                return parsed, nil
        }

        parsed, ok := new(big.Int).SetString(value, 10)
        if !ok {
                return nil, errors.New("the value is not a valid decimal quantity")
        }
        return parsed, nil
}

// leftPadUint128 encodes a uint64 as a 16-byte big-endian word.
func leftPadUint128(value uint64) []byte {
        out := make([]byte, 16)
        for i := 0; i < 8; i++ {
                out[15-i] = byte(value >> (8 * uint(i)))
        }
        return out
}

// uint48Bytes encodes a uint64 as a 6-byte big-endian value.
func uint48Bytes(value uint64) []byte {
        out := make([]byte, 6)
        for i := 0; i < 6; i++ {
                out[5-i] = byte(value >> (8 * uint(i)))
        }
        return out
}

// loadPaymasterSigner reads the sponsor key from the environment.
//
// A missing or malformed key disables sponsorship rather than producing an
// approval the paymaster would reject.
func loadPaymasterSigner() (*ecdsa.PrivateKey, error) {
        key := strings.TrimSpace(os.Getenv("PAYMASTER_SIGNER_KEY"))
        if key == "" {
                return nil, errors.New("PAYMASTER_SIGNER_KEY is not set")
        }
        return crypto.HexToECDSA(strings.TrimPrefix(key, "0x"))
}
