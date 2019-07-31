package tx

import (
	"context"
	"log"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

// GasPricer determines the gas price to send a tx with
type GasPricer interface {
	GasPrice() *big.Int
}

// EthClient describes the methods needed for the sender to interact with the ETH blockchain
type EthClient interface {
	ethereum.TransactionSender
	ethereum.TransactionReader
	ethereum.PendingStateReader
	ethereum.ChainStateReader
	ethereum.GasPricer
}

// Sender uses a wallet to automatically send a tx at regular intervals in order to include txs
// with different gas prices in blocks
type Sender struct {
	account accounts.Account
	wallet  accounts.Wallet
	chainID *big.Int
	client  EthClient
	txp     GasPricer

	running      bool
	sendInterval time.Duration

	quit chan struct{}
}

// NewSender returns a Sender instance
func NewSender(account accounts.Account, wallet accounts.Wallet, chainID *big.Int, client EthClient, txp GasPricer, interval time.Duration) *Sender {
	return &Sender{
		account:      account,
		wallet:       wallet,
		chainID:      chainID,
		client:       client,
		txp:          txp,
		sendInterval: interval,
		quit:         make(chan struct{}),
	}
}

// Start initiates the tx submission loop
func (s *Sender) Start() {
	if s.running {
		return
	}

	go s.startSubmitLoop()

	s.running = true
}

// Stop signals the tx submission loop to exit gracefully
func (s *Sender) Stop() {
	if !s.running {
		return
	}

	close(s.quit)
}

func (s *Sender) createTx() (*types.Transaction, error) {
	gasLimit := uint64(21000)
	gasPrice := s.txp.GasPrice()
	nonce, err := s.client.PendingNonceAt(context.Background(), s.account.Address)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(nonce, s.account.Address, big.NewInt(0), gasLimit, gasPrice, []byte{})
	signedTx, err := s.wallet.SignTx(s.account, tx, s.chainID)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}

func (s *Sender) sendTx() {
	signedTx, err := s.createTx()
	if err != nil {
		log.Printf("error creating tx: %v", err)
		return
	}

	if err := s.client.SendTransaction(context.Background(), signedTx); err != nil {
		log.Printf("error sending tx: %v", err)
		return
	}

	receipt, err := bind.WaitMined(context.Background(), s.client, signedTx)
	if err != nil {
		log.Printf("error waiting for tx %x: %v", signedTx.Hash(), err)
		return
	}

	if receipt.Status == uint64(0) {
		log.Printf("tx %x failed", signedTx.Hash())
		return
	}

	log.Printf("confirmed tx %x with gasPrice = %v", signedTx.Hash(), signedTx.GasPrice())

	gp, err := s.client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Printf("error querying gas price oracle: %v", err)
		return
	}

	log.Printf("suggested gas price = %v", gp)
}

func (s *Sender) startSubmitLoop() {
	ticker := time.NewTicker(s.sendInterval)

	for {
		select {
		case <-ticker.C:
			go s.sendTx()
		case <-s.quit:
			return
		}
	}
}
