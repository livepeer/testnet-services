
## Running the faucet 

Requires a RPC connection to a running ethereum node

```
go run main.go faucet.go --network 54321 --provider "http://localhost:8545" --keystore keystore/ --address d0c38befc1cf6c126b6dac7184edad7f64ebf7e7 --password password.txt --httpport 3099
```

## Example GET 
```
$ curl http://localhost:3099
Hello! this is a simple ETH faucet.
Request ETH by making a POST request to the root URL with your ethereum address
Current payout is 1000000000000000000 wei
```

## Example POST
```
$ curl -X POST http://localhost:3099 -H "Content-Type:application/json" -d '{"address":"0xd0c38befc1cf6c126b6dac7184edad7f64ebf7e7"}'
{"txhash":"0xc45c0c7a83c009adcaaf7f006f6f77e4eb265d164b796c2fd09d71709eddb130"}
```