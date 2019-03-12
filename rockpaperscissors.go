package main

import (
	"encoding/json"
	"fmt"
	"strconv"
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

func ToChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

func callHBSCC(stub shim.ChaincodeStubInterface, key string, namespace string) {
	chainCodeArgs := ToChaincodeArgs("initRecon", key, namespace)
	response := stub.InvokeChaincode("honeybadgerscc", chainCodeArgs, "mychannel")

	if response.Status != shim.OK {
		fmt.Println(response.Message)
	}

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

		memcell := "rps" + args[2] + "cell" // in real use case this should
		// be a hash or something non-deterministic
		game := &game{objectType, args[0], timeLimit, memcell, "None", args[2], "None", int(time.Now().Unix()), "None"}
		gameJSONasBytes, err := json.Marshal(game)
		if err != nil {
			return shim.Error(err.Error())
		}
		err = stub.PutState(args[0], gameJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(memcell))
	} else if fn == "joinGame" && len(args) >= 2 {
		fmt.Println("In joinGame endpoint")
		// Join a game
		// args[0] - name of the game
		// args[1] - name of the user
		// returns memory cell id for the user move
		memcell := "rps" + args[1] + "cell"
		var gameInstance game
		gameAsBytes, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}
		err = json.Unmarshal([]byte(gameAsBytes), &gameInstance)
		if err != nil {
			return shim.Error(err.Error())
		}
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
		// check if time as elapsed
		timeCur := int(time.Now().Unix())
		timeElapsed := gameInstance.Time + gameInstance.End
		fmt.Println("timeCur: " + string(timeCur))
		fmt.Println("timeElapsed: " + string(timeElapsed))
		if timeCur >= timeElapsed {
			// startReconstruct
			fmt.Println("Started reconstruct")
			callHBSCC(stub, gameInstance.M1, "rps")
			callHBSCC(stub, gameInstance.M2, "rps")
		} else {
			fmt.Println("Time not elapsed")
			return shim.Success([]byte(string(timeCur - timeElapsed)))
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
