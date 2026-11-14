package chainsdk

// pathutil.go — BIP-44 path helpers shared by every non-EVM caller.

import (
	"strconv"
	"strings"
)

func WithAccountIndex(path string, index int) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return path
	}
	last := parts[len(parts)-1]
	if strings.HasSuffix(last, "'") {
		parts[len(parts)-1] = strconv.Itoa(index) + "'"
	} else {
		parts[len(parts)-1] = strconv.Itoa(index)
	}
	return strings.Join(parts, "/")
}
