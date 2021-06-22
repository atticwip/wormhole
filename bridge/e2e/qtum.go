package e2e

import (
	"context"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/abi"
	"github.com/certusone/wormhole/bridge/pkg/ethereum/erc20"
	"github.com/certusone/wormhole/bridge/pkg/qtum"
	qtumABI "github.com/certusone/wormhole/bridge/pkg/qtum/abi"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"k8s.io/apimachinery/pkg/util/wait"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

// waitQtumBalance waits for target account before to increase.
func waitQtumRCBalance(t *testing.T, ctx context.Context, token *erc20.Erc20, before *big.Int, target int64) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err := wait.PollUntil(1*time.Second, func() (bool, error) {

		after, err := token.BalanceOf(nil, devnet.QtumClientMinerAddress)
		if err != nil {
			t.Log(err)
			return false, nil
		}

		d := new(big.Int).Sub(after, before)
		t.Logf("QRC20 balance after: %d -> %d, delta %d", before, after, d)

		if after.Cmp(before) != 0 {
			if d.Cmp(new(big.Int).SetInt64(target)) != 0 {
				t.Errorf("expected QRC20 delta of %v, got: %v", target, d)
			}
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Error(err)
	}
}

func waitWrappedToken(t *testing.T, ctx context.Context, contractAddress common.Address, assetID [32]byte, cl *ethclient.Client) (address common.Address) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var isFound bool

	err := wait.PollUntil(1*time.Second, func() (bool, error) {
		address, isFound = getWrappedAsset(contractAddress, assetID, cl)
		if isFound {
			return true, nil
		}

		t.Logf("WrappedAsset %s not found", contractAddress)
		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Error(err)
	}

	return address
}

func testQtumLockup(t *testing.T, ctx context.Context, qRPCURL string, janusQClient, ethClient *ethclient.Client, signerWIF, bridgeContractAddr string, assetID [32]byte, isWrappedToken, isWrappedDestination bool, tokenAddr, destination common.Address, amount int64, precision int) {
	//Default is true
	isFound := true
	var contract common.Address
	var cl *ethclient.Client

	beforeQrc20 := new(big.Int)

	if isWrappedToken {
		contract = common.HexToAddress(bridgeContractAddr)
		cl = janusQClient
		tokenAddr, isFound = getWrappedAsset(contract, assetID, janusQClient)
	}

	if !isWrappedToken || isFound {
		// Source qtum token client
		qrcToken, err := erc20.NewErc20(tokenAddr, janusQClient)
		if err != nil {
			panic(err)
		}

		// Store balance of source QRC20 token
		beforeQrc20, err = qrcToken.BalanceOf(nil, devnet.QtumClientMinerAddress)
		if err != nil {
			t.Log(err) // account may not yet exist, defaults to 0
		}
	}

	t.Logf("QRC20 balance: %v", beforeQrc20)

	beforeErc20 := new(big.Int)

	if isWrappedDestination {
		contract = devnet.GanacheBridgeContractAddress
		cl = ethClient
		destination, isFound = getWrappedAsset(devnet.GanacheBridgeContractAddress, assetID, ethClient)
	}

	if !isWrappedDestination || isFound {
		ercToken, err := erc20.NewErc20(destination, ethClient)
		if err != nil {
			panic(err)
		}

		/// Store balance of wrapped destination token
		beforeErc20, err = ercToken.BalanceOf(nil, devnet.GanacheClientDefaultAccountAddress)
		if err != nil {
			t.Log(err) // account may not yet exist, defaults to 0
		}
	}

	t.Logf("ERC20 balance: %v", beforeErc20)

	tx := sendLockAsset(qRPCURL, bridgeContractAddr, signerWIF, tokenAddr, amount)

	t.Logf("sent lockup tx: %s", tx)

	if !isFound {
		destination = waitWrappedToken(t, ctx, contract, assetID, cl)
	}

	ercToken, err := erc20.NewErc20(destination, ethClient)
	if err != nil {
		panic(err)
	}

	qrcToken, err := erc20.NewErc20(tokenAddr, janusQClient)
	if err != nil {
		panic(err)
	}

	// Destination account increases by full amount.
	waitEthBalance(t, ctx, ercToken, beforeErc20, int64(float64(amount)*math.Pow10(precision)))

	// Source account decreases by the full amount.
	waitQtumRCBalance(t, ctx, qrcToken, beforeQrc20, -int64(amount))
}

func getWrappedAsset(contractAddress common.Address, assetID [32]byte, cl *ethclient.Client) (address common.Address, isFound bool) {
	// Bridge client
	ethBridge, err := abi.NewAbi(contractAddress, cl)
	if err != nil {
		panic(err)
	}

	destination, err := ethBridge.WrappedAssets(nil, assetID)
	if err != nil {
		panic(err)
	}

	if destination.String() == "0x0000000000000000000000000000000000000000" {
		return address, false
	}

	return destination, true
}

func buildAssetID(tokenAddress common.Address, chainID int64) (assetID [32]byte) {
	paddedAddress := qtum.PadAddress(tokenAddress)
	encoded := append([]byte{byte(chainID)}, paddedAddress[:]...)
	hash := crypto.Keccak256(encoded)
	copy(assetID[:], hash)

	return assetID
}

func sendLockAsset(qRPCURL, bridgeContractAddr, signerWIF string, tokenAddr common.Address, amount int64) string {
	// Bridge client
	qtumBridge, err := qtumABI.NewAbiQtum(qRPCURL, bridgeContractAddr, "regtest", nil)
	if err != nil {
		panic(err)
	}

	// Send lockup
	tx, err := qtumBridge.LockAssets(signerWIF,
		// asset address
		tokenAddr,
		// token amount
		new(big.Int).SetInt64(amount),
		// recipient address on target chain
		qtum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
		// target chain
		vaa.ChainIDEthereum,
		// random nonce
		rand.Uint32(),
		// refund dust?
		false,
	)
	if err != nil {
		panic(err)
	}

	return tx
}
