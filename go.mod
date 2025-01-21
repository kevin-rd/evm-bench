module github.io/kevin-rd/evm-bench

go 1.22.5

require (
	github.com/ethereum/go-ethereum v1.14.11
	github.com/gorilla/websocket v1.5.3
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240223125850-b1e8a79f509c // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/ethereum/go-verkle v0.1.1-0.20240829091221-dffa7562dbe9 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/holiman/uint256 v1.3.1 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.18.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rs/cors v1.8.3 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/supranational/blst v0.3.13 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/zkMeLabs/mechain-go-sdk v0.2.0-alpha.1.0.20241212065041-42ab97c3c753 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

replace (
	cosmossdk.io/api => github.com/zkMeLabs/mechain-cosmos-sdk/api v0.0.0-20241017101002-ab985b5a45ec
	cosmossdk.io/math => github.com/zkMeLabs/mechain-cosmos-sdk/math v0.0.0-20241017101002-ab985b5a45ec
	cosmossdk.io/simapp => github.com/zkMeLabs/mechain-cosmos-sdk/simapp v0.0.0-20241017101002-ab985b5a45ec
	github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.22.1
	github.com/cometbft/cometbft => github.com/zkMeLabs/mechain-cometbft v1.3.0-mechain.2
	github.com/cometbft/cometbft-db => github.com/zkMeLabs/mechain-cometbft-db v0.8.1-alpha.1
	github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0
	github.com/consensys/gnark-crypto => github.com/consensys/gnark-crypto v0.7.0
	github.com/cosmos/cosmos-sdk => github.com/zkMeLabs/mechain-cosmos-sdk v0.2.0-alpha.4
	github.com/cosmos/gogoproto => github.com/zkMeLabs/gogoproto v1.4.10-mechain.1
	github.com/cosmos/iavl => github.com/zkMeLabs/mechain-iavl v0.20.1
	github.com/cosmos/ibc-go/v7 => github.com/zkMeLabs/mechain-ibc-go/v7 v7.2.0-mocks-mechain.2
	github.com/ethereum/go-ethereum => github.com/evmos/go-ethereum v1.10.26-evmos-rc2
	github.com/evmos/evmos/v12 => github.com/zkMeLabs/mechain/v12 v12.0.0-20241203073109-477b21592402
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
)
