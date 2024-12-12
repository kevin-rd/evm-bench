package main

import (
	"github.io/kevin-rd/evm-bench/eth"
	"github.io/kevin-rd/evm-bench/internal/statistics"
	"log"
	"sync"
	"time"
)

const (
	wsURL   = "ws://127.0.0.1:8546"
	rpcAddr = "http://127.0.0.1:26657"

	maxPending    = 3000
	PressDuration = time.Second * 120

	recipientAddr = "0x2344991936359AAcaAC175198F556c08cd74dF55"
)

var accounts = []string{
	"dbf2999f925145213f7262580a7a3a0562426509746d1e10cd1e610198e679a0",
	"23c7159b2b8b02b1f45edc6069c1771784a2630358c9d0cdb82c41033b79f635",
	"b93f760c5524e6883d0019f03cb82797603b5b90870669e501e5296f79e156a6",
	"f4411f3e1323f7b6238f109510781d62d08a00b3041ff16c7cae0a9c4d111cae",
	"5be9f77d4c91b4acb422ece974547eea0721a72108f1e4027ac01eef02ba9439",
}

func main() {
	var wg sync.WaitGroup
	var wgReceiver sync.WaitGroup
	ch := make(chan *statistics.TestResult, 1)

	// 建立连接
	works := make([]*eth.Client, len(accounts))
	for i := 0; i < len(accounts); i++ {
		client, err := eth.NewClient(i, wsURL, rpcAddr, accounts[i], recipientAddr)
		if err != nil {
			log.Fatal("Failed to connect to WebSocket:", err)
		}
		works[i] = client
	}

	// statistics
	wgReceiver.Add(1)
	go func() {
		defer wgReceiver.Done()
		log.Printf("statistics start...")
		statistics.HandleStatistics(uint64(len(works)), ch)
	}()

	for i := 0; i < len(works); i++ {
		// Slow start
		if i > 0 && i%10 == 0 {
			time.Sleep(PressDuration / 1000)
		}

		wg.Add(1)
		log.Printf("worker %d start...", i)
		go func(index int, ch chan<- *statistics.TestResult) {
			defer wg.Done()

			err := works[i].BatchSendTxs(PressDuration, maxPending, ch)
			if err != nil {
				log.Printf("worker %d failed: %v", index, err)
				return
			}
		}(i, ch)
	}
	wg.Wait()
	close(ch)
	wgReceiver.Wait()
}
