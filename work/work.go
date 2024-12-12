package work

import (
	"github.com/ethereum/go-ethereum/core/types"
	"log"
	"math/rand"
)

type Worker struct {
	id       int
	cli      client.IClient
	sequence int

	account    *types.Account
	objectSize int64
	data       []byte
	bucketName string
}

func NewWorker(id int, objectSize int64, privateKey string) *Worker {
	data := make([]byte, objectSize)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	// import account
	account, err := types.NewAccountFromPrivateKey("file_test", privateKey)
	if err != nil {
		log.Fatalf("New account from private key error, %v", err)
	}

	// create client
	cli, err := client.New(chainId, rpcAddr, evmRpcAddr, privateKey, client.Option{DefaultAccount: account})
	if err != nil {
		log.Fatalf("unable to new zkMe Chain client, %v", err)
	}

	return &Worker{
		id:         id,
		cli:        cli,
		account:    account,
		objectSize: objectSize,
		data:       data,
	}
}
