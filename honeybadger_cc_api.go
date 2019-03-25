package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type secretcell struct {
	ObjectType string `json:"docType"`
	CellName   string `json:"cellName"`
	IsWritten  bool   `json:"isWriten"`
	WriterKey  string `json:"WriterKey"`
	IsOpen     bool   `json:"IsOpen"`
	Value      string `json:"Value"`
}

const channelName = "mychannel"

func getCellMetaData(stub shim.ChaincodeStubInterface, cellname string) secretcell {
	chainCodeArgs := ToChaincodeArgs("getCell", cellname, "rps")
	response := stub.InvokeChaincode("honeybadgerscc", chainCodeArgs, channelName)
	var cellInstance secretcell
	json.Unmarshal([]byte(response.Payload), &cellInstance)
	return cellInstance
}

func ToChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

func checkResult(stub shim.ChaincodeStubInterface, key string, namespace string) string {
	chainCodeArgs := ToChaincodeArgs("getResult", key, namespace)
	response := stub.InvokeChaincode("honeybadgerscc", chainCodeArgs, channelName)
	return string(response.Payload)
}

func reconstructSecret(stub shim.ChaincodeStubInterface, key string, namespace string) string {
	chainCodeArgs := ToChaincodeArgs("pubRecon", key, namespace)
	response := stub.InvokeChaincode("honeybadgerscc", chainCodeArgs, "mychannel")

	if response.Status != shim.OK {
		fmt.Println(response.Message)
	}

	return string(response.Payload)
}
