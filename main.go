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
	"github.io/kevin-rd/evm-bench/internal/statistics"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	wsURL               = "ws://127.0.0.1:8546"
	rpcAddr             = "http://127.0.0.1:26657"
	chainID       int64 = 5151
	maxPending          = 7000
	PressDuration       = time.Second * 120

	privateKeyHex        = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	recipient            = "0xb83782C315090b826C670f8a354a9dc3B4942ebf"
	gasLimit      uint64 = 42000
	gasPrice             = 100
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
	res := map[uint64]*statistics.TestResult{}

	var wgReceiver sync.WaitGroup
	ch := make(chan *statistics.TestResult, 1)

	// statistics
	wgReceiver.Add(1)
	go func() {
		defer wgReceiver.Done()
		log.Printf("statistics start...")
		statistics.HandleStatistics(1, res, ch)
	}()

	_ = batchSendTxs(5000, res, ch)
	wgReceiver.Wait()
}

func batchSendTxs(num int, res map[uint64]*statistics.TestResult, ch chan<- *statistics.TestResult) error {
	index := 0
	var success int
	var nonce uint64
	var startNonce uint64
	var lastNonce uint64
	var startTime time.Time
	var lastTime time.Time
	var lastTps float64

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
	startTime = time.Now()
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
			log.Printf("tolal num_unconfirmed_txs in mempool: %d", pending)

			for maxPending-pending >= 500 {
				tx := types.NewTx(&types.LegacyTx{
					Nonce:    nonce,
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
				res[nonce] = &statistics.TestResult{
					Nonce:   nonce,
					ReqTime: time.Now(),
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
				if index%2000 == 0 {
					// request tx pool
					if err := client.WriteJSON(eth.ETH_TransactionCount, []interface{}{fromAddress.Hex(), "pending"}); err != nil {
						log.Printf("Failed to transaction_count request: %v", err)
						_ = client.ReConn()
					}
				}
			}
			// send self
			if err := client.WriteJSON(eth.ETH_TXPoolStatus, []interface{}{}); err != nil {
				log.Fatalf("Failed to send txpool_status request: %v", err)
			}
			time.Sleep(time.Second * 1)
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
			//log.Printf("Successfully send tx: %s, nonce:%d", txHex, startNonce+uint64(success))
			//err := client.WriteJSONRaw(1, "eth_getTransactionByHash", []interface{}{txHex})

			if r, ok := res[startNonce+uint64(success)]; ok {
				r.Success = true
				r.Cost = time.Now().Sub(r.ReqTime)
			} else {
				res[startNonce+uint64(success)] = &statistics.TestResult{
					Success: true,
				}
			}
			ch <- res[startNonce+uint64(success)]
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
			if nonce <= 0 {
				nonce = uint64(nonceValue)
				startNonce = uint64(nonceValue)
				lastNonce = startNonce
				startTime = time.Now()
				lastTime = startTime
				log.Printf("Begin to test, %d", startNonce)
				// request tx pool
				if err := client.WriteJSON(eth.ETH_TXPoolStatus, []interface{}{}); err != nil {
					log.Printf("Failed to txpool_status request: %v", err)
					_ = client.ReConn()
				}
			} else {
				// waiting for tx to be confirmed
				curTime := time.Now()
				curNonce := uint64(nonceValue)

				cost := curTime.Sub(lastTime)
				lastTps = 0.5*lastTps + 0.5*(float64)(curNonce-lastNonce)/cost.Seconds()
				log.Printf("Total send: %d, confirmed: %d, spend: %fs, cur tps: %f", index, curNonce-startNonce, cost.Seconds(), lastTps)

				lastTime = time.Now()
				lastNonce = uint64(nonceValue)

				if time.Now().Sub(startTime) > PressDuration {
					log.Printf("Exit.")
					return nil
				}
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
