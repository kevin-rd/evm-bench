package main

import (
	"github.io/kevin-rd/evm-bench/eth"
	"github.io/kevin-rd/evm-bench/internal/account"
	"github.io/kevin-rd/evm-bench/internal/statistics"
	"log"
	"sync"
	"time"
)

const (
	wsURL   = "ws://127.0.0.1:8546"
	evmURL  = "http://127.0.0.1:8545"
	rpcAddr = "http://127.0.0.1:26657"

	maxPending    = 2000
	PressDuration = time.Second * 120

	recipientAddr = "0x2344991936359AAcaAC175198F556c08cd74dF55"
)

var accounts = []string{
	"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
	"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
	"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a",
	"7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
	"47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
}

func main2() {
	var wg sync.WaitGroup
	var wgReceiver sync.WaitGroup

	chTemp := make(chan *statistics.TestResult, len(accounts)*1000)
	chStatistics := make(chan *statistics.TestResult)

	// 建立连接
	works := make([]*eth.Client, len(accounts))
	for i := 0; i < len(works); i++ {
		client, err := eth.NewClient(i, wsURL, rpcAddr, accounts[i], recipientAddr)
		if err != nil {
			log.Fatal("Failed to connect to WebSocket:", err)
		}
		works[i] = client
	}

	// query time
	go func() {
		client, _ := eth.NewClient(0, wsURL, rpcAddr, accounts[0], recipientAddr)
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

	for i := 0; i < len(works); i++ {
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

func main() {
	keys, err := account.GenerateAccounts("accounts.txt", 100)
	if err != nil {
		log.Fatal(err)
	}

	_ = account.TransferAccounts(evmURL, accounts[0], keys)
	log.Printf("transfer done")
}
