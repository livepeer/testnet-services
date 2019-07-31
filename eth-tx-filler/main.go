package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/yondonfu/eth-tx-filler/gasprice"
	"github.com/yondonfu/eth-tx-filler/tx"
)

func run() error {
	// sender flags
	senderAddr := flag.String("senderAddr", "", "tx sender address")
	keystoreDir := flag.String("keystoreDir", "", "keystore directory")
	password := flag.String("password", "password.txt", "File containing password")

	chainID := flag.Int("chainID", -1, "chain ID")
	provider := flag.String("provider", "http://localhost:8545", "ETH provider URL")
	sendInterval := flag.Int("sendInterval", 5, "time interval to send tx in seconds")
	// gas price flags
	randomizeInterval := flag.Int("randomizeInterval", 30, "time interval to randomize gas price in seconds")
	maxGasPrice := flag.String("maxGasPrice", "1000", "max for randomizing gas price")
	minGasPrice := flag.String("minGasPrice", "100", "min for randomizing gas price")

	flag.Parse()

	if *senderAddr == "" {
		return errors.New("need -senderAddr")
	}

	if *keystoreDir == "" {
		return errors.New("need -keystoreDir")
	}

	if *chainID == -1 {
		return errors.New("need -chainID")
	}

	if *provider == "" {
		return errors.New("need -provider")
	}

	if *sendInterval <= 0 {
		return errors.New("-senderInterval must be > 0")
	}

	if *randomizeInterval <= 0 {
		return errors.New("-randomizeInterval must be > 0")
	}

	bigMaxGasPrice, ok := new(big.Int).SetString(*maxGasPrice, 10)
	if !ok || bigMaxGasPrice.Cmp(big.NewInt(0)) < 0 {
		return errors.New("-maxGasPrice must be >= 0")
	}

	bigMinGasPrice, ok := new(big.Int).SetString(*minGasPrice, 10)
	if !ok || bigMinGasPrice.Cmp(big.NewInt(0)) < 0 {
		return errors.New("-minGasPrice must be >= 0")
	}

	randomizer := gasprice.NewRandomizer(time.Duration(*randomizeInterval)*time.Second, bigMaxGasPrice, bigMinGasPrice)

	randomizer.Start()
	defer randomizer.Stop()

	// Load up the account key and decrypt its password
	wallet, err := unlockWallet(*senderAddr, *password, *keystoreDir)
	if err != nil {
		log.Fatal(err)
	}
	acct := wallet.Accounts()[0]

	if !strings.HasPrefix(*provider, "http") {
		*provider = "http://" + *provider
	}

	client, err := ethclient.Dial(*provider)
	if err != nil {
		return err
	}

	sender := tx.NewSender(acct, wallet, big.NewInt(int64(*chainID)), client, randomizer, time.Duration(*sendInterval)*time.Second)
	sender.Start()
	defer sender.Stop()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	return nil
}

func findWallet(ks *keystore.KeyStore, acct accounts.Account) (accounts.Wallet, error) {
	wallets := ks.Wallets()
	for _, w := range wallets {
		accts := w.Accounts()
		if len(accts) > 0 && accts[0] == acct {
			return w, nil
		}
	}

	return nil, fmt.Errorf("wallet for %x not found", acct.Address)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// unlockWallet unlocks a wallet file
// @param address Ethereum account to unlock
// @param password Path to txt file containing decryption key
// @param keys keystore path
// @return error
func unlockWallet(address string, password string, keys string) (accounts.Wallet, error) {
	// Load up the account key and decrypt its password
	blob, err := ioutil.ReadFile(password)
	if err != nil {
		return nil, fmt.Errorf("failed to read password from %v", password)
	}
	// Delete trailing newline in password
	pass := strings.TrimSuffix(string(blob), "\n")

	ks := keystore.NewKeyStore(keys, keystore.StandardScryptN, keystore.StandardScryptP)
	addr := common.HexToAddress(address)
	acc, err := ks.Find(accounts.Account{Address: addr})
	if err != nil {
		return nil, fmt.Errorf("unable to find account %v in keystore %v", address, keys)
	}
	if err := ks.Unlock(acc, pass); err != nil {
		return nil, fmt.Errorf("unable to unlock account %v: %v", address, err)
	}
	log.Printf("Account %v succesfully unlocked", address)

	// Fetch Wallet interface for unlocked account to sign transactions
	wallets := ks.Wallets()
	for _, w := range wallets {
		accts := w.Accounts()
		if len(accts) > 0 && accts[0] == acc {
			return w, nil
		}
	}

	return nil, fmt.Errorf("wallet for %x not found", acc.Address)
}
