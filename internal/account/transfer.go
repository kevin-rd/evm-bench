package account

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TransferAccounts(url, privateKeyHex string, keys []string) error {
	client, err := ethclient.Dial(url)
	if err != nil {
		log.Fatal(err)
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(1000000000000000000) // in wei
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, key := range keys {
		to, err := toAddress(key)
		if err != nil {
			return fmt.Errorf("valid private key")
		}

		tx := types.NewTransaction(nonce, to, value, 210000, gasPrice, nil)

		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			log.Fatal(err)
		}

		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
		if err != nil {
			log.Fatal(err)
		}

		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("tx sent: %s", signedTx.Hash().Hex())

	}
	return nil
}

func toAddress(hexKey string) (common.Address, error) {
	privateKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return common.Address{}, fmt.Errorf("valid private key")
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("not public")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA), nil
}
