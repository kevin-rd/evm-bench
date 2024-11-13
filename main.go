package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.io/kevin-rd/evm-bench/eth"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const (
	wsURL            = "ws://127.0.0.1:8546"
	rpcAddr          = "http://127.0.0.1:26657"
	chainID    int64 = 5151
	maxPending       = 10000

	privateKeyHex        = "0xf78a036930ce63791ea6ea20072986d8c3f16a6811f6a2583b0787c45086f769"
	recipient            = "0x32a91324730D77FC25cfFF5a21038f306b6a8a30"
	gasLimit      uint64 = 42000
	gasPrice             = 1000000000
)

var (
	privateKey  *ecdsa.PrivateKey
	fromAddress common.Address
	toAddress   common.Address
)

func init() {
	var err error
	privateKey, err = crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	fromAddress = crypto.PubkeyToAddress(privateKey.PublicKey)

}

func main() {
	for i := 0; i < 1; i++ {
		_ = batchSendTxs(5000)
		time.Sleep(2 * time.Second)
	}
}

func batchSendTxs(num int) error {
	index := 0
	var success int
	var nonce uint64
	var startNonce uint64
	toAddress = common.HexToAddress(recipient)

	// 建立连接
	client, err := eth.NewClient(wsURL, rpcAddr)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket:", err)
	}
	log.Println("Connected to WebSocket")
	defer client.Close()

	// send initial request
	if err := client.WriteJSON(eth.ETH_TransactionCount, []interface{}{fromAddress.Hex(), "pending"}); err != nil {
		log.Fatalf("Failed to send initial request: %v", err)
	}

	// for until total num txs
	var startTime = time.Now()
	for {
		resp, err := client.ReadResponse()
		if err != nil {
			log.Printf("Error reading response: %v", err)
			_ = client.ReConn()
			continue
		}

		switch eth.MethodId(resp.ID) {
		case eth.ETH_TXPoolStatus: // txpool_status
			pending := getPending()
			var poolStatus eth.PoolStatus
			if err := json.Unmarshal(resp.Result, &poolStatus); err != nil {
				log.Printf("Failed to parse poolStatus: %v", err)
				continue
			}
			log.Printf("num_unconfirmed_txs total: %d", pending)
			if maxPending-pending < (num - index) {
				time.Sleep(5 * time.Second)
				break
			}

			for num-index > 0 {
				tx := types.NewTx(&types.LegacyTx{
					Nonce:    uint64(nonce),
					To:       &toAddress,
					Value:    big.NewInt(1000000000),
					Gas:      gasLimit,
					GasPrice: big.NewInt(gasPrice),
				})
				signedTx, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(chainID)), privateKey)
				if err != nil {
					log.Printf("Failed to sign transaction: %v", err)
					break
				}
				rawTx, err := signedTx.MarshalBinary()
				if err != nil {
					log.Printf("Failed to marshal transaction: %v", err)
					break
				}
				// send raw tx
				if err := client.WriteJSON(eth.ETH_RawTransaction, []interface{}{fmt.Sprintf("0x%x", rawTx)}); err != nil {
					log.Printf("Failed to send eth_sendRawTransaction: %v", err)
					break
				}
				pending++
				nonce++
				index++
				if index%500 == 0 {
					log.Printf("Sent tx index:%d, nonce:%d", index, nonce)
				}
			}
			// send initial request
			if err := client.WriteJSON(eth.ETH_TransactionCount, []interface{}{fromAddress.Hex(), "pending"}); err != nil {
				log.Fatalf("Failed to send initial request: %v", err)
			}
		case eth.ETH_RawTransaction: // eth_sendRawTransaction
			if resp.Error != nil {
				log.Printf("eth_sendRawTransaction Error: %v", resp.Error.Message)
				continue
			}

			var txHex string
			if err := json.Unmarshal(resp.Result, &txHex); err != nil {
				log.Printf("Failed to parse txHex: %v", err)
				continue
			}
			success++
			// log.Printf("Received Transaction: %s", txHex)
		case eth.ETH_TransactionCount: // eth_getTransactionCount
			if resp.Error != nil {
				log.Printf("eth_getTransactionCount Error: %v", resp.Error.Message)
				continue
			}

			var nonceValue hexutil.Uint64
			if err = json.Unmarshal(resp.Result, &nonceValue); err != nil {
				log.Printf("Failed to parse nonce: %v", err)
				continue
			}
			log.Printf("Transaction Nonce: %d", nonceValue)
			if nonce <= 0 {
				nonce = uint64(nonceValue)
				startNonce = uint64(nonceValue)
				// request tx pool
				if err := client.WriteJSON(eth.ETH_TXPoolStatus, []interface{}{}); err != nil {
					log.Printf("Failed to txpool_status request: %v", err)
					_ = client.ReConn()
				}
			} else {
				// waiting for tx to be confirmed
				cost := time.Now().Sub(startTime)
				tps := (float64)(uint64(nonceValue)-startNonce) / cost.Seconds()
				log.Printf("Total send: %d, confirmed: %d, spend: %fs, tps: %f", index, uint64(nonceValue)-startNonce, cost.Seconds(), tps)
			}

		case 4:
			if resp.Error != nil {
				log.Printf("JSON-RPC Error: %v", resp.Error.Message)
				continue
			}
			var chainIdHex string
			if err = json.Unmarshal(resp.Result, &chainIdHex); err != nil {
				log.Printf("Failed to parse chainId: %v", err)
				continue
			}
			chainId, ok := new(big.Int).SetString(strings.TrimPrefix(chainIdHex, "0x"), 16)
			if !ok {
				log.Printf("Invalid chainId format: %s", chainIdHex)
				continue
			}
			if chainId.Int64() != chainID {
				log.Fatal("Invalid chain MethodId")
			}
			log.Printf("Connected to Chain MethodId: %d", chainId)
		default:
			log.Printf("Unknown MethodId: %d", resp.ID)
		}
	}
	return nil
}

func getPending() int {
	resp, err := http.Get(rpcAddr + "/num_unconfirmed_txs")
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
		return 0
	}
	defer resp.Body.Close()

	var data eth.JSONRPCResponse
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return 0
	}

	var unconfirmedTxs UnconfirmedTxs
	if err = json.Unmarshal(data.Result, &unconfirmedTxs); err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return 0
	}
	return unconfirmedTxs.Total
}

type UnconfirmedTxs struct {
	Txs        int `json:"n_txs,string"`
	Total      int `json:"total,string"`
	TotalBytes int `json:"total_bytes,string"`
}
