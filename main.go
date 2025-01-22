package main

import (
	"github.io/kevin-rd/evm-bench/internal/account"
	"github.io/kevin-rd/evm-bench/internal/evm"
	"github.io/kevin-rd/evm-bench/internal/statistics"
	"log"
	"sync"
	"time"
)

const (
	wsURL   = "ws://127.0.0.1:8546"
	evmURL  = "http://devint-rpc.mechain.tech:80"
	rpcAddr = "http://127.0.0.1:26657"

	maxPending    = 5000
	PressDuration = time.Second * 120

	RandomAccountNum = 100
	recipientAddr    = "0x2344991936359AAcaAC175198F556c08cd74dF55"
)

func main() {
	var wg sync.WaitGroup
	var wgReceiver sync.WaitGroup

	keys, err := account.GenerateAccounts("accounts.txt", RandomAccountNum)
	if err != nil {
		log.Fatal(err)
	}

	chTemp := make(chan *statistics.TestResult, RandomAccountNum*10)
	chStatistics := make(chan *statistics.TestResult)

	// 建立连接
	works := make([]*evm.Client, 1)
	for i := 0; i < len(works); i++ {
		client, err := evm.NewClient(i, wsURL, rpcAddr, recipientAddr, keys)
		if err != nil {
			log.Fatal("Failed to connect to WebSocket:", err)
		}
		client.QueryAccountNonce()
		works[i] = client
	}

	// query time
	go func() {
		client, _ := evm.NewClient(0, wsURL, rpcAddr, recipientAddr, keys)
		client.QueryTxTime(chTemp, chStatistics)
		log.Printf("query time done")
	}()

	// statistics
	wgReceiver.Add(1)
	go func() {
		defer wgReceiver.Done()
		log.Printf("statistics start...")
		statistics.HandleStatistics(uint64(len(works)), chStatistics)
	}()

	for i := 0; i < 1; i++ {
		// slow start
		if i%10 == 0 {
			time.Sleep(PressDuration / 1000)
		}

		wg.Add(1)
		log.Printf("worker %d start...", i)
		go func(index int, ch chan *statistics.TestResult) {
			defer wg.Done()

			err := works[i].BatchSendTxs(PressDuration, maxPending, ch)
			if err != nil {
				log.Printf("worker %d failed: %v", index, err)
				return
			}
		}(i, chTemp)
	}
	wg.Wait()
	close(chTemp)
	close(chStatistics)
	wgReceiver.Wait()
}

func main3() {
	keys, err := account.GenerateAccounts("accounts.txt", 100)
	if err != nil {
		log.Fatal(err)
	}

	privateKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	err = account.TransferAccounts(rpcAddr, evmURL, privateKey, keys)
	if err != nil {
		log.Printf("transfer failed: %v", err)
	}
	log.Printf("transfer done")
}
