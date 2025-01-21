package account

import (
	"context"
	"cosmossdk.io/math"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/evmos/v12/sdk/types"
	mechainclient "github.com/zkMeLabs/mechain-go-sdk/client"
	mechaintypes "github.com/zkMeLabs/mechain-go-sdk/types"
	"log"
)

func TransferAccounts(rpcUrl, evmUrl, privateKeyHex string, keys []string) error {

	account, err := mechaintypes.NewAccountFromPrivateKey("default-account", privateKeyHex)
	if err != nil {
		return fmt.Errorf("valid private to")
	}

	cli, err := mechainclient.New("mechain_5151-1", rpcUrl, evmUrl, privateKeyHex, mechainclient.Option{DefaultAccount: account})
	if err != nil {
		return fmt.Errorf("valid private to")
	}

	amount := math.NewIntWithDecimal(20, 18)
	for _, key := range keys {

		to, err := toAddress(key)
		if err != nil {
			return fmt.Errorf("valid private to")
		}

		txHash, err := cli.Transfer(context.TODO(), to.Hex(), amount, types.TxOption{})
		if err != nil {
			return fmt.Errorf("transfer %s azkme to address %s failed, err: %v", amount, to, err)
		}

		tx, err := cli.WaitForTx(context.TODO(), txHash)
		if err != nil {
			return err
		}
		if tx.TxResult.Code != 0 {
			return fmt.Errorf("transfer %s azkme to address %s failed, err: %v", amount, to, err)
		}
		log.Printf("transfer %s azkme to address %s success, txHash: %s", amount, to, txHash)
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
