package eth

import (
	"encoding/json"
	"log"
	"net/http"
)

type UnconfirmedTxs struct {
	Txs        int `json:"n_txs,string"`
	Total      int `json:"total,string"`
	TotalBytes int `json:"total_bytes,string"`
}

func getPending(rpcAddr string) int {
	resp, err := http.Get(rpcAddr + "/num_unconfirmed_txs")
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
		return 0
	}
	defer resp.Body.Close()

	var data JSONRPCResponse
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
