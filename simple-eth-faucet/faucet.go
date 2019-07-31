package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type faucet struct {
	client  *ethclient.Client
	account accounts.Account
	wallet  accounts.Wallet // wallet to sign with
	network *big.Int        // chain ID
	payout  *big.Int        // amount of ETH to pay out
	port    int
}

func newFaucet(provider string, account accounts.Account, wallet accounts.Wallet, network *big.Int, payout *big.Int, port int) (*faucet, error) {
	client, err := ethclient.Dial(provider)
	if err != nil {
		return nil, err
	}
	return &faucet{
		client,
		account,
		wallet,
		network,
		payout,
		port,
	}, nil
}

func (f *faucet) createTx(to common.Address) (*types.Transaction, error) {
	nonce, err := f.client.PendingNonceAt(context.Background(), f.account.Address)
	if err != nil {
		return nil, err
	}
	gasLimit := uint64(21000)
	gasPrice, err := f.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	return types.NewTransaction(nonce, to, f.payout, gasLimit, gasPrice, []byte{}), nil
}

func (f *faucet) signTx(tx *types.Transaction) (*types.Transaction, error) {
	signedTx, err := f.wallet.SignTx(f.account, tx, f.network)
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}

func (f *faucet) sendTx(tx *types.Transaction) (string, error) {
	if err := f.client.SendTransaction(context.Background(), tx); err != nil {
		return "", err
	}

	receipt, err := bind.WaitMined(context.Background(), f.client, tx)
	if err != nil {
		return "", fmt.Errorf("error waiting for tx %x: %v", tx.Hash().Hex(), err)
	}

	if receipt.Status == uint64(0) {
		return "", fmt.Errorf("tx %x failed", tx.Hash().Hex())
	}

	log.Printf("confirmed tx %x with gasPrice = %v", tx.Hash().Hex(), tx.GasPrice().Int64())
	return tx.Hash().Hex(), nil
}

func (f *faucet) listenAndServe() {
	http.HandleFunc("/", f.faucetHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", f.port), nil)
}

type request struct {
	Address string `json:"address"`
}

type response struct {
	TxHash string `json:"txhash"`
}

func (f *faucet) faucetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		msg := fmt.Sprintf(
			"Hello! this is a simple ETH faucet.\n"+
				"Request ETH by making a POST request to the root URL with your ethereum address\n"+
				"Current payout is %v wei\n",
			f.payout)
		fmt.Fprintf(w, msg)
	case "POST":
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		tx, err := f.createTx(common.HexToAddress(req.Address))
		if err != nil || tx == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error occured: %v \n", err)
			return
		}
		if tx, err = f.signTx(tx); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var res response
		res.TxHash, err = f.sendTx(tx)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error sending transaction: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)

	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
