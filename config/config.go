package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Token uint8

const (
	ONT Token = iota
	ONG
	OEP4
)

type Config struct {
	Wallet            string
	Password          string
	ConsensusPeerPath [][2]string
	ContractCodePath  string
	Contract          string
	To                string
	Amount            uint64
	Rpc               []string
	TxNum             uint // whole tx num is *TxFactor
	TxFactor          uint
	RoutineNum        uint // whole tx save to RoutineNum files, and one go-routine per file
	TPS               uint
	StartNonce        uint32
	GasPrice          uint64
	GasLimit          uint64
	SaveTx            bool
	SendTx            bool
}

func ParseConfig(path string) (*Config, error) {
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ParseConfig: failed, err: %s", err)
	}
	config := &Config{}
	err = json.Unmarshal(fileContent, config)
	if err != nil {
		return nil, fmt.Errorf("ParseConfig: failed, err: %s", err)
	}
	return config, nil
}
