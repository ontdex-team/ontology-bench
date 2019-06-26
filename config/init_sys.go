package config

import (
	"encoding/hex"
	"fmt"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology-crypto/keypair"
	goSdk "github.com/ontio/ontology-go-sdk"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	"io/ioutil"
	"strings"
	"time"
)

const (
	DEFAULT_GAS_PRICE       = 0
	DEFAULT_GAS_LIMIT       = 20000
	DEFAULT_DEPLOY_GASLIMIT = 200000000
)

func SetGasPrice(sdk *goSdk.OntologySdk, consensusAccounts []*goSdk.Account, gasPrice uint64) {
	params := map[string]string{
		"gasPrice": fmt.Sprint(gasPrice),
	}
	tx, err := sdk.Native.GlobalParams.NewSetGlobalParamsTransaction(DEFAULT_GAS_PRICE, DEFAULT_GAS_LIMIT, params)
	if err != nil {
		log.Errorf("SetGasPrice: build tx failed, err: %s", err)
		return
	}
	err = MultiSign(tx, sdk, consensusAccounts)
	if err != nil {
		log.Errorf("SetGasPrice: multi sign failed, err: %s", err)
		return
	}
	hash, err := sdk.SendTransaction(tx)
	if err != nil {
		log.Errorf("SetGasPrice: send tx failed, err: %s", err)
		return
	}
	log.Infof("SetGasPrice: success, tx hash is %s", hash.ToHexString())
}

func WithdrawAsset(sdk *goSdk.OntologySdk, consensusAccounts []*goSdk.Account, destAcc *goSdk.Account) {
	pubKeys := make([]keypair.PublicKey, len(consensusAccounts))
	for index, account := range consensusAccounts {
		pubKeys[index] = account.PublicKey
	}
	m := (5*len(pubKeys) + 6) / 7
	multiSignAddr, err := types.AddressFromMultiPubKeys(pubKeys, m)
	if err != nil {
		log.Errorf("WithdrawAsset: build multi sign addr failed, err: %s", err)
		return
	}
	ontBalance, err := sdk.Native.Ont.BalanceOf(multiSignAddr)
	if err != nil {
		log.Errorf("WithdrawAsset: get multi sign addr %s ont balance failed, err: %s", multiSignAddr.ToBase58(),
			err)
		return
	}
	log.Infof("WithdrawAsset: multi sign addr %s ont balance is %d", multiSignAddr.ToBase58(), ontBalance)
	log.Infof("WithdrawAsset: start withdraw ont...")
	withdrawOntTx, err := sdk.Native.Ont.NewTransferTransaction(DEFAULT_GAS_PRICE, DEFAULT_GAS_LIMIT, multiSignAddr,
		destAcc.Address, ontBalance)
	if err != nil {
		log.Errorf("WithdrawAsset: build withdraw ont tx failed, err: %s", err)
		return
	}
	err = MultiSign(withdrawOntTx, sdk, consensusAccounts)
	if err != nil {
		log.Errorf("WithdrawAsset: multi sign withdraw ont tx failed, err: %s", err)
		return
	}
	withdrawOntHash, err := sdk.SendTransaction(withdrawOntTx)
	if err != nil {
		log.Errorf("WithdrawAsset: send withdraw ont tx failed, err: %s", err)
		return
	}
	log.Infof("WithdrawAsset: withdraw ont success, tx hash is %s, wait one block to confirm", withdrawOntHash.ToHexString())
	wait, err := sdk.WaitForGenerateBlock(30*time.Second, 1)
	if !wait || err != nil {
		log.Errorf("WithdrawAsset: wait withdraw ont failed, err: %s", err)
		return
	}
	log.Infof("WithdrawAsset: completed withdraw ont")
	log.Infof("WithdrawAsset: start withdraw ong...")
	uboundOng, err := sdk.Native.Ong.UnboundONG(multiSignAddr)
	if err != nil {
		log.Errorf("WithdrawAsset: get unbound ong num failed, err: %s", err)
		return
	}
	log.Infof("WithdrawAsset: multi sign addr %s unbound ong is %d", multiSignAddr.ToBase58(), uboundOng)
	withdrawOngTx, err := sdk.Native.Ong.NewTransferFromTransaction(DEFAULT_GAS_PRICE, DEFAULT_GAS_LIMIT, multiSignAddr,
		goSdk.ONT_CONTRACT_ADDRESS, destAcc.Address, uboundOng)
	if err != nil {
		log.Errorf("WithdrawAsset: build withdraw ong tx failed, err: %s", err)
		return
	}
	err = MultiSign(withdrawOngTx, sdk, consensusAccounts)
	if err != nil {
		log.Errorf("WithdrawAsset: multi sign withdraw ong tx failed, err: %s", err)
		return
	}
	withdrawOngHash, err := sdk.SendTransaction(withdrawOngTx)
	if err != nil {
		log.Errorf("WithdrawAsset: send withdraw ong tx failed, err: %s", err)
		return
	}
	log.Infof("WithdrawAsset: withdraw ong success, tx hash is %s, wait one block to confirm", withdrawOngHash.ToHexString())
	wait, err = sdk.WaitForGenerateBlock(30*time.Second, 1)
	if !wait || err != nil {
		log.Errorf("WithdrawAsset: wait withdraw ong failed, err: %s", err)
		return
	}
	log.Infof("WithdrawAsset: completed withdraw ong")
}

func InitOep4(sdk *goSdk.OntologySdk, acc *goSdk.Account, avmPath string) {
	fileContent, err := ioutil.ReadFile(avmPath)
	if err != nil {
		log.Errorf("InitOep4: read source code failed, err: %s", err)
		return
	}
	contractStr := strings.TrimSpace(string(fileContent))
	deployHash, err := sdk.NeoVM.DeployNeoVMSmartContract(DEFAULT_GAS_PRICE, DEFAULT_DEPLOY_GASLIMIT, acc, true,
		contractStr, "MYT", "1.0", "my", "1@1.com", "test")
	if err != nil {
		log.Errorf("InitOep4: deploy failed, err: %s", err)
		return
	}
	log.Infof("InitOep4: deploy success, tx hash is %s", deployHash.ToHexString())
	avmCode, err := hex.DecodeString(contractStr)
	if err != nil {
		log.Errorf("InitOep4: decode avm code failed, err: %s", err)
	}
	contractAddr := common.AddressFromVmCode(avmCode)
	initHash, err := sdk.NeoVM.InvokeNeoVMContract(DEFAULT_GAS_PRICE, DEFAULT_GAS_LIMIT, acc, contractAddr,
		[]interface{}{"init", []interface{}{}})
	if err != nil {
		log.Errorf("InitOep4: init contract failed, err: %s", err)
		return
	}
	log.Infof("InitOep4: init contract %s success, tx hash is %s", contractAddr.ToHexString(), initHash.ToHexString())
}

func MultiSign(tx *types.MutableTransaction, sdk *goSdk.OntologySdk, consensusAccounts []*goSdk.Account) error {
	pubKeys := make([]keypair.PublicKey, len(consensusAccounts))
	for index, account := range consensusAccounts {
		pubKeys[index] = account.PublicKey
	}
	m := uint16((5*len(pubKeys) + 6) / 7)
	for index, account := range consensusAccounts {
		err := sdk.MultiSignToTransaction(tx, m, pubKeys, account)
		if err != nil {
			return fmt.Errorf("MultiSign: index %d, account %s failed, err: %s", index, account.Address.ToBase58(), err)
		}
	}
	return nil
}
