package evm

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
	gasLimit uint64 = 840000
	gasPrice        = 200
)

type Client struct {
	Id int

	evmAddr string
	rpcAddr string
	ws      *websocket.Conn

	toAddress *common.Address
	accounts  []*Account
}

func NewClient(id int, url, rpcAddr, toAddressHex string, recipients []string) (*Client, error) {
	c := Client{Id: id, evmAddr: url, rpcAddr: rpcAddr}
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	_ = ws.SetReadDeadline(time.Now().Add(time.Second * 240))
	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 240))
	c.ws = ws

	toAddress := common.HexToAddress(toAddressHex)
	c.toAddress = &toAddress
	c.accounts = make([]*Account, len(recipients))
	for i, recipient := range recipients {
		from, err := crypto.HexToECDSA(strings.TrimPrefix(recipient, "0x"))
		if err != nil {
			log.Fatalf("Failed to load private key: %v", err)
		}
		publicKey, ok := from.Public().(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not public")
		}
		//addr := common.HexToAddress(recipient)
		addr := crypto.PubkeyToAddress(*publicKey)
		c.accounts[i] = &Account{
			PrivateKey: from,
			Address:    &addr,
		}
	}
	return &c, nil
}

func (c *Client) BatchSendTxs(PressDuration time.Duration, maxPending int, ch chan<- *statistics.TestResult) error {
	index := 0
	var success int
	res := map[int]*statistics.TestResult{}
	startTime := time.Now()

	// send initial request
	if err := c.WriteJSON(ETH_TXPoolStatus, []interface{}{}); err != nil {
		log.Printf("Failed to txpool_status request: %v", err)
		_ = c.ReConn()
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

			for i := 0; maxPending-pending >= 2400 && i < 2400; i++ {
				fromAccount := c.accounts[index%len(c.accounts)]
				amount := big.NewInt(123000000000)

				tx := types.NewTx(&types.LegacyTx{
					Nonce:    fromAccount.Nonce,
					To:       c.toAddress,
					Value:    amount,
					Gas:      gasLimit,
					GasPrice: big.NewInt(gasPrice),
				})
				signedTx, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(chainID)), fromAccount.PrivateKey)
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
					Nonce:   fromAccount.Nonce,
					ReqTime: time.Now(),
				}
				// send raw tx
				if err := c.WriteJSON(ETH_RawTransaction, []interface{}{fmt.Sprintf("0x%x", rawTx)}); err != nil {
					log.Printf("Failed to send eth_sendRawTransaction: %v", err)
					break
				}

				pending++
				fromAccount.Nonce++
				index++
				if index%500 == 0 {
					log.Printf("Sent tx index:%d, nonce:%d", index, fromAccount.Nonce)
				}
			}

			if time.Now().Sub(startTime) > PressDuration {

			} else {
				// send self
				if err := c.WriteJSON(ETH_TXPoolStatus, []interface{}{}); err != nil {
					log.Fatalf("Failed to send txpool_status request: %v", err)
				}
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
			log.Printf("eth_getTransactionCount")
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

func (c *Client) QueryAccountNonce() {
	for index := 0; index < len(c.accounts); index++ {
		if err := c.WriteJSON(ETH_TransactionCount, []interface{}{c.accounts[index].Address.Hex(), "pending"}); err != nil {
			log.Fatalf("Failed to send initial request: %v", err)
		}

		resp, err := c.ReadResponse()
		if err != nil {
			log.Fatalf("Error reading response: %v", err)
		}

		switch MethodId(resp.ID) {
		case ETH_TransactionCount: // eth_getTransactionCount
			if resp.Error != nil {
				log.Fatalf("eth_getTransactionCount Error: %v", resp.Error.Message)
			}
			var nonceValue hexutil.Uint64
			if err = json.Unmarshal(resp.Result, &nonceValue); err != nil {
				log.Fatalf("Failed to parse nonce: %v", err)
			}
			c.accounts[index].Nonce = uint64(nonceValue)
		}
	}
}

// QueryTxTime statistical tx confirmation time
func (c *Client) QueryTxTime(chTx chan *statistics.TestResult, chStatistics chan<- *statistics.TestResult) {
	var blocks map[uint64]Block = make(map[uint64]Block)

	for res := range chTx {
		if res.BlockNum == 0 {
			// query tx in nextBlock
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
				time.Sleep(time.Second)
				chTx <- res
				continue
			}
			res.Success = true
			res.BlockNum = uint64(*tx.BlockNumber)
		}

		// find clock in local blocks
		nextBlock, ok := blocks[res.BlockNum+1]
		if !ok {
			// query nextBlock from chain
			if err := c.WriteJSONRaw(1, "eth_getBlockByNumber", []interface{}{res.BlockNum + 1, false}); err != nil {
				log.Printf("Failed to get nextBlock by number: %v", err)
				chTx <- res
				continue
			}
			if err := c.ReadJson(1, &nextBlock); err != nil {
				log.Printf("Failed to get nextBlock: %v", err)
				chTx <- res
				continue
			} else if uint64(nextBlock.Number) != res.BlockNum+1 {
				time.Sleep(time.Second)
				chTx <- res
				continue
			}
			blocks[res.BlockNum+1] = nextBlock
		}
		res.Cost = time.Unix(int64(nextBlock.Timestamp), 0).Sub(res.ReqTime)
		if res.Cost <= 0 {
			log.Printf("Error, tx time is negative, tx:%s, nextBlock:%d, time:%s, cost: %.2f", res.TxHash, nextBlock.Timestamp, res.ReqTime.Format("04:05.000"), res.Cost.Seconds())
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
