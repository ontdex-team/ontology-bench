# ontology-bench
test ontology tps, only support oep-4 token transfer bench at current.

## requet
1. deploy a oep-4 contract
2. account at wallet.dat has enough ong and oep-4 token that you deployed

## config
```
{
  "Wallet": "wallet.dat", // wallet path
  "Password": "passwordtest", // account password
  "Contract": "4ef9e947c975eacb60c75b542c8d2fea36e09c65", // contract address
  "To": "AdTgdGPjahJjubZU19AwBu9F3oE4hncx4u", // to account
  "Amount": 1, // amount per transfer
  "GasPrice": 500,
  "GasLimit": 20000,
  "Rpc": "http://polaris4.ont.io:20336", // ontology network rpc address
  "TxNum": 100, // tx num per factor
  "TxFactory": 1, // tx factor
  "RoutineNum": 2, // use go routine to handle bench
  "TPS": 10, // bench tps
  "SaveTx": true, // should save bench tx to file
  "SendTx": true // should send tx
}
```

If SendTx is true, tx will be sent to Rpc address, if SaveTx is true, tx will be save to file. If you want to only save tx rather than sent it, should set SaveTx true and SendTx false.

The program will use RoutineNum go routines to handle TxNum * TxFactor transaction.

## Usage

build:
``` go build -o main main.go ```

run:
``` ./main config.json```
