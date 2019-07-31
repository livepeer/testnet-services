package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	network := flag.Int("network", -1, "chain ID")
	provider := flag.String("provider", "http://localhost:8545", "Ethereum JSON-RPC provider")

	keys := flag.String("keystore", "/keystore", "Keystore path")
	address := flag.String("address", "", "account address to fund user requests with")
	password := flag.String("password", "password.txt", "File containing password")

	payout := flag.Int("payout", 1, "Number of Eth to pay out per request")

	port := flag.Int("httpport", 8080, "Listener port for the HTTP API connection")

	flag.Parse()

	if !strings.HasPrefix(*provider, "http") {
		*provider = "http://" + *provider
	}

	// Load up the account key and decrypt its password
	wallet, err := unlockWallet(*address, *password, *keys)
	if err != nil {
		log.Fatal(err)
	}

	// convert payout ETH > wei
	payoutWei := new(big.Int).Mul(big.NewInt(int64(*payout)), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	// Create a new faucet
	faucet, err := newFaucet(*provider, wallet.Accounts()[0], wallet, big.NewInt(int64(*network)), payoutWei, *port)

	faucet.listenAndServe()

	// shutdown hook
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

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
