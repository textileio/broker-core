package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/textileio/broker-core/cmd/ethd/contractclient"
)

func main() {
	client, err := ethclient.Dial("http://127.0.0.1:8545")
	// client, err := ethclient.Dial("https://mainnet.infura.io/v3/92f6902cf1214401ae5b08a1e117eb91")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("we have a connection")

	registryContractAddr := common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0")
	providerContractAddr := common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9")

	registryContract, err := contractclient.NewBridgeRegistry(registryContractAddr, client)
	if err != nil {
		log.Fatal(err)
	}

	providerContract, err := contractclient.NewBridgeProvider(providerContractAddr, client)
	if err != nil {
		log.Fatal(err)
	}

	hasDeposit, err := providerContract.HasDeposit(nil, common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(hasDeposit)

	// c := make(chan *contractclient.BridgeProviderAddDeposit)

	// _, err = providerContract.WatchAddDeposit(nil, c, nil, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Get count of deposits

	despositAddrs, err := providerContract.ListDepositees(nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(despositAddrs)

	var deposits []struct {
		Timestamp *big.Int
		Sender    common.Address
		Value     *big.Int
	}
	for _, depositAddr := range despositAddrs {
		deposit, err := providerContract.Deposits(nil, depositAddr)
		if err != nil {
			log.Fatal(err)
		}
		deposits = append(deposits, deposit)
	}
	fmt.Println(deposits)

	// Deposits total

	providers, err := registryContract.ListProviders(nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(providers)

	bal, err := client.BalanceAt(context.Background(), registryContractAddr, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bal)

	// blockNum, err := client.BlockNumber(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(blockNum)
}
