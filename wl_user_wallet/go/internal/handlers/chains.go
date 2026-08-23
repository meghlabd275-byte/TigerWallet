package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/wl-user-wallet/internal/chains"
)

// GET /chains — canonical 186-chain registry. Optional ?type=evm|nonevm filter.
func (s *Svc) GetChains(c *gin.Context) {
	chainType := strings.ToLower(strings.TrimSpace(c.Query("type")))
	var list []chains.ChainConfig
	switch chainType {
	case "evm":
		list = chains.ListChainsByType("evm")
	case "nonevm", "non-evm", "non_evm":
		// Non-EVM chains are stored with their own chain_type labels; serve all
		// non-EVM entries by filtering out "evm".
		all := chains.ListSupportedChains()
		for _, ch := range all {
			if !ch.IsEVM() {
				list = append(list, ch)
			}
		}
	default:
		list = chains.ListSupportedChains()
	}
	c.JSON(http.StatusOK, gin.H{
		"chains":       list,
		"count":        len(list),
		"evm_count":    chains.EVMChainCount(),
		"nonevm_count": chains.NonEVMChainCount(),
	})
}
