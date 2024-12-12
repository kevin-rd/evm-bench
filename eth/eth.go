package eth

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
	"github.io/kevin-rd/evm-bench/internal/statistics"
	"log"
	"math/big"
	"strings"
	"time"
)

const (
	chainID  int64  = 5151
	gasLimit uint64 = 42000
	gasPrice        = 100
)

type Client struct {
	Id int

	evmAddr string
	rpcAddr string
	ws      *websocket.Conn

	privateKey  *ecdsa.PrivateKey
	fromAddress common.Address
	toAddress   common.Address
}

func NewClient(id int, url string, rpcAddr, privateKey, recipient string) (*Client, error) {
	c := Client{Id: id, evmAddr: url, rpcAddr: rpcAddr}
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	_ = ws.SetReadDeadline(time.Now().Add(time.Second * 240))
	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 240))
	c.ws = ws

	c.privateKey, err = crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}
	c.fromAddress = crypto.PubkeyToAddress(c.privateKey.PublicKey)
	c.toAddress = common.HexToAddress(recipient)
	return &c, nil
}

func (c *Client) BatchSendTxs(PressDuration time.Duration, maxPending int, ch chan<- *statistics.TestResult) error {
	index := 0
	var success int
	var nonce uint64
	res := map[int]*statistics.TestResult{}
	var startNonce uint64
	var lastNonce uint64
	var startTime time.Time
	var lastTime time.Time
	var lastTps float64

	// send initial request
	if err := c.WriteJSON(ETH_TransactionCount, []interface{}{c.fromAddress.Hex(), "pending"}); err != nil {
		log.Fatalf("Failed to send initial request: %v", err)
	}

	// for until total num txs
	startTime = time.Now()
	for {
		resp, err := c.ReadResponse()
		if err != nil {
			log.Printf("Error reading response: %v", err)
			_ = c.ReConn()
			continue
		}

		switch MethodId(resp.ID) {
		case ETH_TXPoolStatus: // txpool_status
			pending := getPending(c.rpcAddr)
			var poolStatus PoolStatus
			if err := json.Unmarshal(resp.Result, &poolStatus); err != nil {
				log.Printf("Failed to parse poolStatus: %v", err)
				continue
			}
			log.Printf("tolal num_unconfirmed_txs in mempool: %d", pending)

			if maxPending-pending >= 200 {
				for i := 0; i < 400; i++ {
					tx := types.NewTx(&types.LegacyTx{
						Nonce:    nonce,
						To:       &c.toAddress,
						Value:    big.NewInt(1000000000),
						Gas:      gasLimit,
						GasPrice: big.NewInt(gasPrice),
					})
					signedTx, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(chainID)), c.privateKey)
					if err != nil {
						log.Printf("Failed to sign transaction: %v", err)
						break
					}
					rawTx, err := signedTx.MarshalBinary()
					if err != nil {
						log.Printf("Failed to marshal transaction: %v", err)
						break
					}
					res[index] = &statistics.TestResult{
						ChanId:  c.Id,
						Nonce:   nonce,
						ReqTime: time.Now(),
					}
					// send raw tx
					if err := c.WriteJSON(ETH_RawTransaction, []interface{}{fmt.Sprintf("0x%x", rawTx)}); err != nil {
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
						if err := c.WriteJSON(ETH_TransactionCount, []interface{}{c.fromAddress.Hex(), "pending"}); err != nil {
							log.Printf("Failed to transaction_count request: %v", err)
							_ = c.ReConn()
						}
					}
				}
			}

			// send self
			if err := c.WriteJSON(ETH_TXPoolStatus, []interface{}{}); err != nil {
				log.Fatalf("Failed to send txpool_status request: %v", err)
			}
			time.Sleep(time.Second * 1)
		case ETH_RawTransaction: // eth_sendRawTransaction
			if resp.Error != nil {
				log.Printf("eth_sendRawTransaction Error: %v", resp.Error.Message)
				continue
			}

			var txHex string
			if err := json.Unmarshal(resp.Result, &txHex); err != nil {
				log.Printf("Error unmarshaling JSON: %v", err)
				continue
			}
			if r, ok := res[success]; ok {
				r.TxHash = txHex
				success++
				ch <- r
			} else {
				log.Fatalf("Error, no found testResult, wordId:%d, success:%d", c.Id, success)
			}
			//log.Printf("Successfully send tx: %s, nonce:%d", txHex, startNonce+uint64(success))
		case ETH_TransactionCount: // eth_getTransactionCount
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
				log.Printf("Begin to test, startNonce: %d", startNonce)
				// request tx pool
				if err := c.WriteJSON(ETH_TXPoolStatus, []interface{}{}); err != nil {
					log.Printf("Failed to txpool_status request: %v", err)
					_ = c.ReConn()
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
			log.Printf("Connected to Chain MethodId: %d", chainId)
		default:
			log.Printf("Unknown MethodId: %d", resp.ID)
		}
	}
	return nil
}

// QueryTxTime statistical tx confirmation time
func (c *Client) QueryTxTime(chTx chan *statistics.TestResult, chStatistics chan<- *statistics.TestResult) {
	var blocks map[uint64]Block = make(map[uint64]Block)

	for res := range chTx {
		if res.BlockNum == 0 {
			// query tx in block
			if err := c.WriteJSONRaw(1, "eth_getTransactionByHash", []interface{}{res.TxHash}); err != nil {
				log.Printf("Failed to send eth_getTransactionByHash: %v", err)
				chTx <- res
				continue
			}

			var tx Transaction
			if err := c.ReadJson(1, &tx); err != nil {
				log.Printf("Failed to get tx: %v", err)
				chTx <- res
				continue
			} else if tx.Hash == "" || tx.BlockHash == nil {
				log.Printf("Failed to get tx: %v", tx)
				chTx <- res
				continue
			}
			res.Success = true
			res.BlockNum = uint64(*tx.BlockNumber)
		}

		// find clock in local blocks
		block, ok := blocks[res.BlockNum]
		if !ok {
			// query block from chain
			if err := c.WriteJSONRaw(1, "eth_getBlockByNumber", []interface{}{res.BlockNum, false}); err != nil {
				log.Printf("Failed to get block by number: %v", err)
				chTx <- res
			}
			if err := c.ReadJson(1, &block); err != nil {
				log.Printf("Failed to get block: %v", err)
				chTx <- res
			}
			blocks[res.BlockNum] = block
		}
		res.Cost = time.Unix(int64(block.Timestamp), 0).Sub(res.ReqTime)
		if res.Cost <= 0 {
			log.Printf("Error, tx time is negative, tx:%s, block:%d, time:%s, cost: %.2f", res.TxHash, res.BlockNum, res.ReqTime.Format("04:05.000"), res.Cost.Seconds())
		}
		chStatistics <- res
	}
}

func (c *Client) ReConn() error {
	if err := c.ws.Close(); err != nil {
		log.Fatalf("Error Close ws: %v", err)
	}
	ws, _, err := websocket.DefaultDialer.Dial(c.evmAddr, nil)
	if err != nil {
		log.Fatalf("Error ReConn to ws: %v", err)
	}
	_ = ws.SetReadDeadline(time.Now().Add(time.Second * 120))
	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 120))
	c.ws = ws
	return nil
}

func (c *Client) WriteJSON(id MethodId, params []interface{}) error {
	return c.ws.WriteJSON(&JSONRPCRequest{
		Version: DefaultVersion,
		Method:  id.String(),
		Params:  params,
		ID:      int(id),
	})
}

func (c *Client) WriteJSONRaw(id int, method string, params []interface{}) error {
	return c.ws.WriteJSON(&JSONRPCRequest{
		Version: DefaultVersion,
		Method:  method,
		Params:  params,
		ID:      id,
	})
}

func (c *Client) ReadJson(id int, v any) error {
	resp, err := c.ReadResponse()
	if err != nil {
		return err
	}
	if resp.ID != id {
		return fmt.Errorf("invalid id: %d", resp.ID)
	}
	return json.Unmarshal(resp.Result, v)
}

func (c *Client) ReadResponse() (resp JSONRPCResponse, err error) {
	_, message, err := c.ReadMessage()
	if err != nil {
		return
	}
	err = json.Unmarshal(message, &resp)
	return
}

func (c *Client) ReadMessage() (messageType int, p []byte, err error) {
	return c.ws.ReadMessage()
}

func (c *Client) Close() error {
	return c.ws.Close()
}
