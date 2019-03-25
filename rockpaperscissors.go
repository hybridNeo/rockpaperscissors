package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type RPS struct{}

type game struct {
	ObjectType string `json:"docType"`
	Name       string `json:"name"`
	End        int    `json:"end"`
	M1         string `json:"m1"`
	M2         string `json:"m2"`
	U1         string `json:"u1"`
	U2         string `json:"u2"`
	Time       int    `json:"time"`
	Result     string `json:"result"`
}

// Init
func (t *RPS) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func createCell(stub shim.ChaincodeStubInterface, key string, userID string, namespace string) {
	chainCodeArgs := ToChaincodeArgs("createCell", key, userID, namespace)
	response := stub.InvokeChaincode("honeybadgerscc", chainCodeArgs, "mychannel")

	if response.Status != shim.OK {
		fmt.Println(response.Message)
	}
}

func getActiveGames(stub shim.ChaincodeStubInterface) peer.Response {
	keysIter, err := stub.GetStateByRange("", "")
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var items []game
	for keysIter.HasNext() {
		key, _ := keysIter.Next()
		item, _ := stub.GetState(key.Key)
		var gameInstance game
		err = json.Unmarshal([]byte(item), &gameInstance)
		if gameInstance.Result == "None" {
			items = append(items, gameInstance)
		}
	}
	JSONasBytes, err := json.Marshal(items)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(JSONasBytes))

}

func getCompletedGames(stub shim.ChaincodeStubInterface) peer.Response {
	keysIter, err := stub.GetStateByRange("", "")
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var items []game
	for keysIter.HasNext() {
		key, _ := keysIter.Next()
		item, _ := stub.GetState(key.Key)
		var gameInstance game
		err = json.Unmarshal([]byte(item), &gameInstance)
		if gameInstance.Result != "None" {
			items = append(items, gameInstance)
		}
	}
	JSONasBytes, err := json.Marshal(items)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(JSONasBytes))

}

// Invoke
func (t *RPS) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	// Extract the function and args from the transaction proposal
	fn, args := stub.GetFunctionAndParameters()
	if fn == "createGame" && len(args) >= 3 {
		fmt.Println("In createGame endpoint ")
		// Create a game with the parameters provided
		// args[0] - name of the game
		// args[1] - time to end the game in seconds
		// args[2] - name of the user
		// returns memory cell id for the user move
		timeLimit, err := strconv.Atoi(args[1])
		if err != nil {
			return shim.Error(err.Error())
		}

		objectType := "game"

		memcell := "rps" + args[0] + args[2] + "cell" // in real use case this should
		// be a hash or something non-deterministic
		createCell(stub, memcell, args[2], "rps") // create a memory cell for the user to enter data
		game := &game{objectType, args[0], timeLimit, memcell, "None", args[2], "None", int(0), "None"}
		gameJSONasBytes, err := json.Marshal(game)
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.PutState(args[0], gameJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(memcell))
	} else if fn == "getActiveGames" {

		return getActiveGames(stub)

	} else if fn == "getCompletedGames" {

		return getCompletedGames(stub)

	} else if fn == "joinGame" && len(args) >= 2 {
		fmt.Println("In joinGame endpoint")
		// Join a game
		// args[0] - name of the game
		// args[1] - name of the user
		// returns memory cell id for the user move
		memcell := "rps" + args[0] + args[1] + "cell"
		var gameInstance game
		gameAsBytes, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}
		err = json.Unmarshal([]byte(gameAsBytes), &gameInstance)
		if err != nil {
			return shim.Error(err.Error())
		}
		createCell(stub, memcell, args[1], "rps")
		gameInstance.M2 = memcell
		gameInstance.U2 = args[1]
		gameJSONasBytes, err := json.Marshal(gameInstance)
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.PutState(args[0], []byte(gameJSONasBytes))
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(memcell))
	} else if fn == "openMoves" && len(args) >= 1 {
		var gameInstance game
		fmt.Println("In openMoves")
		gameAsBytes, _ := stub.GetState(args[0])
		json.Unmarshal([]byte(gameAsBytes), &gameInstance)
		var cell1, cell2 secretcell
		cell1 = getCellMetaData(stub, gameInstance.M1)
		cell2 = getCellMetaData(stub, gameInstance.M2)
		if cell1.IsWritten && cell2.IsWritten {
			fmt.Println("Started reconstruct")
			reconstructSecret(stub, gameInstance.M1, "rps")
			reconstructSecret(stub, gameInstance.M2, "rps")
			return shim.Success([]byte("Started"))
		}
		return shim.Success([]byte("None"))
	} else if fn == "endGame" && len(args) >= 2 {
		// end the game
		// args[0] - name of the game
		var gameInstance game
		fmt.Println("In endGame ")
		gameAsBytes, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		err = json.Unmarshal([]byte(gameAsBytes), &gameInstance)
		if err != nil {
			return shim.Error(err.Error())
		}
		var cell1, cell2 secretcell
		// check if both cells are ready to be opened
		if gameInstance.M1 == "None" || gameInstance.M2 == "None" {
			return shim.Success([]byte("None"))
		}
		for {
			if checkResult(stub, gameInstance.M1, "rps") != "None" {
				break
			}
			time.Sleep(2 * time.Second)
		}
		for {
			if checkResult(stub, gameInstance.M2, "rps") != "None" {
				break

			}
			time.Sleep(2 * time.Second)
		}
		cell1 = getCellMetaData(stub, gameInstance.M1)
		cell2 = getCellMetaData(stub, gameInstance.M2)
		if cell1.IsOpen && cell2.IsOpen {
			if gameInstance.Result != "None" {
				return shim.Success([]byte(gameInstance.Result))
			}
			// startReconstruct
			cellVal := checkResult(stub, gameInstance.M1, "rps") // checks if there is a reconstruction result.
			fmt.Println("cellVal: " + cellVal)
			gameInstance.M1 = strings.Trim(cellVal, " ")

			cellVal = checkResult(stub, gameInstance.M2, "rps")
			gameInstance.M2 = strings.Trim(cellVal, " ")
			fmt.Println("M1:" + gameInstance.M1 + ".")
			fmt.Println("M2:" + gameInstance.M2 + ".")
			// 10 = rock
			// 11 = paper
			// 12 = scissors
			if gameInstance.M1 == "10" && gameInstance.M2 == "11" {
				gameInstance.Result = gameInstance.U2
			} else if gameInstance.M1 == "10" && gameInstance.M2 == "12" {
				gameInstance.Result = gameInstance.U1
			} else if gameInstance.M1 == "11" && gameInstance.M2 == "12" {
				gameInstance.Result = gameInstance.U2
			} else if gameInstance.M1 == "11" && gameInstance.M2 == "10" {
				gameInstance.Result = gameInstance.U1
			} else if gameInstance.M1 == "12" && gameInstance.M2 == "10" {
				gameInstance.Result = gameInstance.U2
			} else if gameInstance.M1 == "12" && gameInstance.M2 == "11" {
				gameInstance.Result = gameInstance.U1
			} else {
				// stalemate
				gameInstance.Result = "Stalemate"
			}
			fmt.Println("Result is  " + gameInstance.Result)
			gameJSONasBytes, err := json.Marshal(gameInstance)
			if err != nil {
				return shim.Error(err.Error())
			}
			err = stub.PutState(args[0], []byte(gameJSONasBytes))
			if err != nil {
				return shim.Error(err.Error())
			}
			return shim.Success([]byte(gameInstance.Result))
		} else {
			fmt.Println("cells not opened yet")
			return shim.Success([]byte(string("None")))
		}
		return shim.Success([]byte("None"))
	}
	return shim.Success([]byte("Invalid endpoint"))
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(RPS)); err != nil {
		fmt.Printf("Error starting RPS chaincode: %s", err)
	}
}
