package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
)

// JSON-RPC request structure
type JSONRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSON-RPC response structure
type JSONRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	// Configuration
	wsURL := "ws://154.223.178.3:8546"
	recipient := "0x32a91324730D77FC25cfFF5a21038f306b6a8a30"
	privateKeyHex := "0xf78a036930ce63791ea6ea20072986d8c3f16a6811f6a2583b0787c45086f769"
	maxGap := 5 * 60                   // 5 minutes in seconds
	maxPending := 5000                 // Maximum pending transactions
	gasPrice := big.NewInt(1000000000) // 1 Gwei
	gasLimit := uint64(21000)          // Standard gas limit

	// Initialize account
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Initialize WebSocket connection
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer ws.Close()
	log.Println("Connected to WebSocket")

	// Initialize synchronization
	// var wg sync.WaitGroup

	// Initialize variables
	var startNonce int64 = -1
	var sendNonce uint64 = 0
	var replyCount int = 0
	var totalSend int = 0
	var chainID int64 = -1
	var errorCount int = 0
	startTime := time.Now().Unix()

	// Create JSON-RPC requests
	txPoolReq := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "txpool_status",
		Params:  []interface{}{},
		ID:      0,
	}

	sendRawTxReq := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  []interface{}{},
		ID:      1,
	}

	getNonceReq := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getTransactionCount",
		Params:  []interface{}{fromAddress.Hex(), "latest"},
		ID:      3,
	}

	getChainIDReq := JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_chainId",
		Params:  []interface{}{},
		ID:      4,
	}

	// Helper function to send JSON-RPC requests
	sendJSONRPC := func(req JSONRPCRequest) error {
		message, err := json.Marshal(req)
		if err != nil {
			return err
		}
		return ws.WriteMessage(websocket.TextMessage, message)
	}

	// Send initial chain ID request
	if err := sendJSONRPC(getChainIDReq); err != nil {
		log.Fatalf("Failed to send chain ID request: %v", err)
	}

	// Handle incoming messages
	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Handle errors
		if resp.Error != nil {
			log.Printf("JSON-RPC Error: %v", resp.Error.Message)
			errorCount++
			if errorCount%10 == 0 {
				log.Println("Too many errors, closing WebSocket")
				ws.Close()
				break
			}
			continue
		}

		switch resp.ID {
		case 0: // txpool_status response
			var txpoolStatus struct {
				Pending string `json:"pending"`
				Queued  string `json:"queued"`
			}
			if err := json.Unmarshal(resp.Result, &txpoolStatus); err != nil {
				log.Printf("Failed to parse txpool_status: %v", err)
				continue
			}

			pending := 0
			send := 0

			// Fill the transaction pool
			for (maxPending - pending) > 0 {
				// Create and sign transaction
				tx := types.NewTransaction(sendNonce, common.HexToAddress(recipient), big.NewInt(10), gasLimit, gasPrice, nil)

				signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
				if err != nil {
					log.Printf("Failed to sign transaction: %v", err)
					continue
				}

				rawTx, err := signedTx.MarshalBinary()
				if err != nil {
					log.Printf("Failed to marshal transaction: %v", err)
					continue
				}

				// Send raw transaction
				sendRawTxReq.Params = []interface{}{fmt.Sprintf("0x%x", rawTx)}
				if err := sendJSONRPC(sendRawTxReq); err != nil {
					log.Printf("Failed to send raw transaction: %v", err)
					continue
				}

				pending++
				sendNonce++
				send++
				totalSend++

				// Periodically request nonce
				if send%5000 == 0 {
					if err := sendJSONRPC(getNonceReq); err != nil {
						log.Printf("Failed to send nonce request: %v", err)
					}
				}

				// Optional: Break if you want to limit sends in one loop
				// if send >= someLimit {
				//     break
				// }
			}

			// Request txpool status again
			if err := sendJSONRPC(txPoolReq); err != nil {
				log.Printf("Failed to send txpool_status request: %v", err)
			}

		case 3: // eth_getTransactionCount response
			var nonceValue hexBigInt
			if err := json.Unmarshal(resp.Result, &nonceValue); err != nil {
				log.Printf("Failed to parse nonce: %v", err)
				continue
			}

			if startNonce < 0 {
				startNonce = nonceValue.Big.Int64()
				sendNonce = uint64(startNonce)
				if err := sendJSONRPC(txPoolReq); err != nil {
					log.Printf("Failed to send txpool_status request: %v", err)
				}
			} else {
				endTime := time.Now().Unix()
				gapTime := endTime - startTime
				count := nonceValue.Big.Int64() - startNonce
				tps := float64(totalSend) / float64(gapTime)
				replyTps := float64(replyCount) / float64(gapTime)
				log.Printf("Total Send: %d, Cached Txs: %d, Send Reply: %d, Tx Reply: %d, Spend: %d s, Reply TPS: %.2f, TPS: %.2f",
					totalSend, totalSend-replyCount, replyCount, count, gapTime, replyTps, tps)

				// Check if maxGap time has passed
				if gapTime > int64(maxGap) {
					log.Println("Max testing time reached, closing WebSocket")
					ws.Close()
					return
				}
			}

		case 4: // eth_chainId response
			var chainIDHex string
			if err := json.Unmarshal(resp.Result, &chainIDHex); err != nil {
				log.Printf("Failed to parse chain ID: %v", err)
				continue
			}
			chainIDBig, ok := new(big.Int).SetString(strings.TrimPrefix(chainIDHex, "0x"), 16)
			if !ok {
				log.Printf("Invalid chain ID format: %s", chainIDHex)
				continue
			}
			chainID = chainIDBig.Int64()
			log.Printf("Connected to Chain ID: %d", chainID)

			// Request nonce
			if err := sendJSONRPC(getNonceReq); err != nil {
				log.Printf("Failed to send nonce request: %v", err)
			}

		default:
			// Handle other responses (e.g., transaction receipts)
			replyCount++
		}
	}
}

// hexBigInt is a helper type to unmarshal hex string to big.Int
type hexBigInt struct {
	Big *big.Int
}

func (h *hexBigInt) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := strings.Trim(string(data), "\"")
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	i := new(big.Int)
	_, ok := i.SetString(s, 16)
	if !ok {
		return fmt.Errorf("invalid hex string: %s", s)
	}
	h.Big = i
	return nil
}
