package eth

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	DefaultVersion = "2.0"
)

// MethodId is the ID of the JSON-RPC method
type MethodId int

const (
	ETH_TXPoolStatus     MethodId = 0
	ETH_RawTransaction   MethodId = 1
	ETH_TransactionCount MethodId = 3
)

func (i MethodId) String() string {
	switch i {
	case ETH_TXPoolStatus:
		return "txpool_status"
	case ETH_TransactionCount:
		return "eth_getTransactionCount"
	case ETH_RawTransaction:
		return "eth_sendRawTransaction"
	default:
		return fmt.Sprintf("unknown MethodId: %d", i)
	}
}
func (i MethodId) Id() int {
	return int(i)
}

// JSONRPCRequest JSON-RPC request structure
type JSONRPCRequest struct {
	Version string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSONRPCResponse JSON-RPC response structure
type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type PoolStatus struct {
	Pending hexutil.Uint `json:"pending"`
	Queued  hexutil.Uint `json:"queued"`
}
