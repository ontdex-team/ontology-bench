package main

import (
	"bytes"
	"fmt"
	goSdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology-go-sdk/client"
	"github.com/ontio/ontology-go-sdk/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontology-bench/config"
	"math"
	"os"
	"time"
)

const (
	INIT_PRIVATE_NETWORK = "init"
	TEST_BY_CONFIG       = "test"
	BALANCE_OF           = "balanceOf"
)

func main() {
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
	if len(os.Args) < 2 {
		log.Errorf("not enough args")
	}
	cmd := os.Args[1]
	configPath := "config.json"
	if len(os.Args) >= 3 {
		configPath = os.Args[2]
	}
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		log.Error(err)
		return
	}
	sdk := goSdk.NewOntologySdk()
	wallet, err := sdk.OpenWallet(cfg.Wallet)
	if err != nil {
		log.Errorf("parse wallet err: %s", err)
		return
	}
	account, err := wallet.GetDefaultAccount([]byte(cfg.Password))
	if err != nil {
		log.Errorf("get account err: %s", err)
		return
	}
	if cmd == INIT_PRIVATE_NETWORK {
		consensusAccounts := make([]*goSdk.Account, 0)
		for _, consensusWallet := range cfg.ConsensusPeerPath {
			wallet, err := sdk.OpenWallet(consensusWallet[0])
			if err != nil {
				log.Errorf("parse consensus wallet err: %s", err)
				return
			}
			account, err := wallet.GetDefaultAccount([]byte(consensusWallet[1]))
			if err != nil {
				log.Errorf("get consensus account err: %s", err)
				return
			}
			consensusAccounts = append(consensusAccounts, account)
		}
		rpcClient := client.NewRpcClient()
		rpcClient.SetAddress(cfg.Rpc[0])
		sdk.SetDefaultClient(rpcClient)
		config.SetGasPrice(sdk, consensusAccounts, 500)
		config.WithdrawAsset(sdk, consensusAccounts, account)
		config.DeployOep4(sdk, account, cfg.ContractCodePath)
	} else if cmd == TEST_BY_CONFIG {
		testOep4Transfer(cfg, account)
	} else if cmd == BALANCE_OF {
		rpcClient := client.NewRpcClient()
		rpcClient.SetAddress(cfg.Rpc[0])
		sdk.SetDefaultClient(rpcClient)
		addr := account.Address
		if len(os.Args) > 4 {
			argAddr, err := utils.AddressFromBase58(os.Args[3])
			if err != nil {
				log.Errorf("decode arg %s to address failed, err: %s", os.Args[3], err)
			}
			addr = argAddr
		}
		balanceOf(cfg, sdk, addr)
	} else {
		log.Errorf("un support cmd")
		return
	}

}

func testOep4Transfer(cfg *config.Config, account *goSdk.Account) {
	txNum := cfg.TxNum * cfg.TxFactor
	if txNum > math.MaxUint32 {
		txNum = math.MaxUint32
	}
	contractAddress, err := utils.AddressFromHexString(cfg.Contract)
	if err != nil {
		log.Errorf("parse contract addr failed, err: %s", err)
		return
	}
	toAddr, err := utils.AddressFromBase58(cfg.To)
	if err != nil {
		log.Errorf("parse to addr failed, err: %s", err)
		return
	}
	params := []interface{}{"transfer", []interface{}{account.Address, toAddr, cfg.Amount}}
	exitChan := make(chan int)
	txNumPerRoutine := txNum / cfg.RoutineNum
	tpsPerRoutine := int64(cfg.TPS / cfg.RoutineNum)
	for i := uint(0); i < cfg.RoutineNum; i++ {
		go func(nonce uint32, routineIndex uint) {
			sendTxSdk := goSdk.NewOntologySdk()
			rpcClient := client.NewRpcClient()
			rpcClient.SetAddress(cfg.Rpc[int(routineIndex)%len(cfg.Rpc)])
			sendTxSdk.SetDefaultClient(rpcClient)
			startTime := time.Now().Unix()
			sentNum := int64(0)
			var fileObj *os.File
			if cfg.SaveTx {
				fileObj, err = os.OpenFile(fmt.Sprintf("invoke_%d.txt", routineIndex),
					os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
				if err != nil {
					fmt.Println("Failed to open the file", err.Error())
					os.Exit(2)
				}
			}
			for j := uint(0); j < txNumPerRoutine; j++ {
				mutTx, err := sendTxSdk.NeoVM.NewNeoVMInvokeTransaction(cfg.GasPrice, cfg.GasLimit, contractAddress, params)
				if err != nil {
					fmt.Println("contract tx err", err)
					os.Exit(1)
				}
				mutTx.Nonce = nonce
				err = sendTxSdk.SignToTransaction(mutTx, account)
				if err != nil {
					log.Errorf("sign tx failed, err: %s", err)
					continue
				}
				if cfg.SendTx {
					hash, err := sendTxSdk.SendTransaction(mutTx)
					if err != nil {
						log.Errorf("send tx failed, err: %s", err)
					} else {
						log.Infof("send tx %s", hash.ToHexString())
					}
					sentNum++
					now := time.Now().Unix()
					diff := sentNum - (now-startTime)*tpsPerRoutine
					if now > startTime && diff > 0 {
						sleepTime := time.Duration(diff*1000/tpsPerRoutine) * time.Millisecond
						time.Sleep(sleepTime)
						log.Infof("sleep %d ms", sleepTime)
					}
				}
				nonce++
				if cfg.SaveTx {
					tx, _ := mutTx.IntoImmutable()
					txHash := tx.Hash()
					txbf := new(bytes.Buffer)
					if err := tx.Serialize(txbf); err != nil {
						fmt.Println("Serialize transaction error.", err)
						os.Exit(1)
					}
					txContent := common.ToHexString(txbf.Bytes())
					fileObj.WriteString(txHash.ToHexString() + "," + txContent + "\n")
				}
			}
			exitChan <- 1
		}(uint32(txNumPerRoutine*i)+3, i)
	}
	for i := uint(0); i < cfg.RoutineNum; i++ {
		<-exitChan
	}
}

func balanceOf(cfg *config.Config, sdk *goSdk.OntologySdk, address common.Address) {
	contractAddr, err := utils.AddressFromHexString(cfg.Contract)
	if err != nil {
		log.Errorf("balanceOf: decode contract addr failed, err: %s", err)
		return
	}
	preResult, err := sdk.NeoVM.PreExecInvokeNeoVMContract(contractAddr, []interface{}{"balanceOf", []interface{}{address}})
	if err != nil {
		log.Errorf("balanceOf: pre-execute failed, err: %s", err)
		return
	}
	balance, err := preResult.Result.ToInteger()
	if err != nil {
		log.Errorf("balanceOf: parse result %v failed, err: %s", preResult, err)
		return
	}
	log.Infof("balanceOf: addr %s is %d", address.ToBase58(), balance)
}
