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

func main() {
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
	configPath := os.Args[1]
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		log.Error(err)
		return
	}
	sdk := goSdk.NewOntologySdk()
	restClient := client.NewRpcClient()
	restClient.SetAddress(cfg.Rpc)
	sdk.SetDefaultClient(restClient)
	var wallet, _ = sdk.OpenWallet(cfg.Wallet)
	account, err := wallet.GetDefaultAccount([]byte(cfg.Password))
	if err != nil {
		log.Errorf("get account err: %s", err)
		return
	}
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
		go func(nonce uint32, fileIndex uint) {
			startTime := time.Now().Unix()
			sentNum := int64(0)
			var fileObj *os.File
			if cfg.SaveTx {
				fileObj, err = os.OpenFile(fmt.Sprintf("invoke_%d.txt", fileIndex),
					os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
				if err != nil {
					fmt.Println("Failed to open the file", err.Error())
					os.Exit(2)
				}
			}
			for j := uint(0); j < txNumPerRoutine; j++ {
				mutTx, err := sdk.NeoVM.NewNeoVMInvokeTransaction(cfg.GasLimit, cfg.GasLimit, contractAddress, params)
				if err != nil {
					fmt.Println("contract tx err", err)
					os.Exit(1)
				}
				mutTx.Nonce = nonce
				err = sdk.SignToTransaction(mutTx, account)
				if err != nil {
					log.Errorf("sign tx failed, err: %s", err)
					continue
				}
				if cfg.SendTx {
					hash, err := sdk.SendTransaction(mutTx)
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
		}(uint32(txNumPerRoutine*i), i)
	}
	for i := uint(0); i < cfg.RoutineNum; i++ {
		<-exitChan
	}
}
