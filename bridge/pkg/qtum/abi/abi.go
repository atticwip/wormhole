package abi

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/qtum/base"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/qtumproject/janus/pkg/qtum"
	"github.com/qtumproject/qtumsuite"
	"math/big"
	"strings"
)

// AbiABI is the input ABI used to generate the binding from.
const AbiABI = "[{\"inputs\":[{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"initial_guardian_set\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"wrapped_asset_master\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"_guardian_set_expirity\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"oldGuardianIndex\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"newGuardianIndex\",\"type\":\"uint32\"}],\"name\":\"LogGuardianSetChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"token_chain\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"token_decimals\",\"type\":\"uint8\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"token\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"sender\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"}],\"name\":\"LogTokensLocked\",\"type\":\"event\"},{\"stateMutability\":\"payable\",\"type\":\"fallback\",\"payable\":true},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"consumedVAAs\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"guardian_set_expirity\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"guardian_set_index\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"name\":\"guardian_sets\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"isWrappedAsset\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[],\"name\":\"wrappedAssetMaster\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"wrappedAssets\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"stateMutability\":\"payable\",\"type\":\"receive\",\"payable\":true},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"}],\"name\":\"getGuardianSet\",\"outputs\":[{\"components\":[{\"internalType\":\"address[]\",\"name\":\"keys\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"expiration_time\",\"type\":\"uint32\"}],\"internalType\":\"structWormhole.GuardianSet\",\"name\":\"gs\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"constant\":true},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"vaa\",\"type\":\"bytes\"}],\"name\":\"submitVAA\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"},{\"internalType\":\"bool\",\"name\":\"refund_dust\",\"type\":\"bool\"}],\"name\":\"lockAssets\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"recipient\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"target_chain\",\"type\":\"uint8\"},{\"internalType\":\"uint32\",\"name\":\"nonce\",\"type\":\"uint32\"}],\"name\":\"lockETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\",\"payable\":true}]"
const defaultGasLimit = int64(2500000)

type AbiQtum struct {
	abi             abi.ABI
	contractAddress string
	filter          bind.ContractFilterer
	base            *base.QtumBase
}

// NewAbiQtum creates a new instance of Abi, bound to a specific deployed contract.
func NewAbiQtum(rpcURL, contractAddress, chainID string, filter bind.ContractFilterer /*, c *qtum.Client*/) (*AbiQtum, error) {

	parsed, err := abi.JSON(strings.NewReader(AbiABI))
	if err != nil {
		return nil, err
	}
	if !common.IsHexAddress(contractAddress) {
		return nil, fmt.Errorf("Not hex contract address: %s", contractAddress)
	}

	cl, err := qtum.NewClient(chainID == qtum.ChainMain, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dialing devnet qtum rpc failed: %v", err)
	}

	qtumBase, err := base.NewQtumBase(cl, chainID)
	if err != nil {
		return nil, err
	}

	filter = qtumContractFilterer{rpcURL: rpcURL}

	return &AbiQtum{
		abi:             parsed,
		contractAddress: contractAddress,
		filter:          filter,

		base: qtumBase,
	}, nil
}

func (a AbiQtum) SubmitVAAParam(vaa []byte) ([]byte, error) {
	return a.abi.Pack("submitVAA", vaa)
}

func (a AbiQtum) SubmitVAA(signerWIF string, vaa []byte) (txID string, err error) {
	args, err := a.abi.Pack("submitVAA", vaa)
	if err != nil {
		return "", err
	}

	wif, err := qtumsuite.DecodeWIF(signerWIF)
	if err != nil {
		return "", err
	}

	senderAddress, err := base.GetAddressFromWIF(wif, a.base.GetChainID())
	if err != nil {
		return "", err
	}

	raw, scripts, err := a.base.SentContractCallFromAddress(senderAddress, a.contractAddress, args)
	if err != nil {
		return "", err
	}

	signedTx, err := a.base.SignTx(wif, raw, scripts)
	if err != nil {
		return "", err
	}

	return a.base.SendRawTx(signedTx)
}

func (a AbiQtum) LockAssets(signerWIF string, asset common.Address, amount *big.Int, recipient [32]byte, target_chain uint8, nonce uint32, refund_dust bool) (txID string, err error) {
	args, err := a.abi.Pack("lockAssets", asset, amount, recipient, target_chain, nonce, refund_dust)
	if err != nil {
		return "", err
	}

	wif, err := qtumsuite.DecodeWIF(signerWIF)
	if err != nil {
		return "", err
	}

	senderAddress, err := base.GetAddressFromWIF(wif, a.base.GetChainID())
	if err != nil {
		return "", err
	}

	raw, scripts, err := a.base.SentContractCallFromAddress(senderAddress, a.contractAddress, args)
	if err != nil {
		return "", err
	}

	signedTx, err := a.base.SignTx(wif, raw, scripts)
	if err != nil {
		return "", err
	}

	return a.base.SendRawTx(signedTx)
}

// Solidity: function guardian_set_index() view returns(uint32)
func (a AbiQtum) GuardianSetIndex() (uint32, error) {

	out, err := a.contractCall("guardian_set_index")
	if err != nil {
		return 0, err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err
}

// WormholeGuardianSet is an auto generated low-level Go binding around an user-defined struct.
type WormholeGuardianSet struct {
	Keys           []common.Address
	ExpirationTime uint32
}

func (a AbiQtum) GetGuardianSet(idx uint32) (guardianSet *WormholeGuardianSet, err error) {

	out, err := a.contractCall("getGuardianSet", idx)
	if err != nil {
		return guardianSet, err
	}

	out0 := abi.ConvertType(out[0], new(WormholeGuardianSet)).(*WormholeGuardianSet)

	return out0, err
}

// AbiLogGuardianSetChanged represents a LogGuardianSetChanged event raised by the Abi contract.
type AbiLogGuardianSetChanged struct {
	OldGuardianIndex uint32
	NewGuardianIndex uint32
	Raw              types.Log // Blockchain specific contextual infos
}

// WatchLogGuardianSetChanged is a free log subscription operation binding the contract event 0xdfb80683934199683861bf00b64ecdf0984bbaf661bf27983dba382e99297a62.
//
// Solidity: event LogGuardianSetChanged(uint32 oldGuardianIndex, uint32 newGuardianIndex)
func (a AbiQtum) WatchLogGuardianSetChanged(ctx context.Context, sink chan<- *AbiLogGuardianSetChanged) (ethereum.Subscription, error) {
	if a.filter == nil {
		return nil, fmt.Errorf("ContractFilterer missed")
	}

	boundContr := bind.NewBoundContract(common.HexToAddress(a.contractAddress), a.abi, nil, nil, a.filter)

	logs, sub, err := boundContr.WatchLogs(nil, "LogGuardianSetChanged")
	if err != nil {
		return nil, err
	}

	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiLogGuardianSetChanged)
				if err := boundContr.UnpackLog(event, "LogGuardianSetChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
	return nil, nil
}

// AbiLogTokensLocked represents a LogTokensLocked event raised by the Abi contract.
type AbiLogTokensLocked struct {
	TargetChain   uint8
	TokenChain    uint8
	TokenDecimals uint8
	Token         [32]byte
	Sender        [32]byte
	Recipient     [32]byte
	Amount        *big.Int
	Nonce         uint32
	Raw           types.Log // Blockchain specific contextual infos
}

// WatchLogTokensLocked is a free log subscription operation binding the contract event 0x6bbd554ad75919f71fd91bf917ca6e4f41c10f03ab25751596a22253bb39aab8.
//
// Solidity: event LogTokensLocked(uint8 target_chain, uint8 token_chain, uint8 token_decimals, bytes32 indexed token, bytes32 indexed sender, bytes32 recipient, uint256 amount, uint32 nonce)
func (a AbiQtum) WatchLogTokensLocked(ctx context.Context, sink chan<- *AbiLogTokensLocked) (ethereum.Subscription, error) {
	if a.filter == nil {
		return nil, fmt.Errorf("ContractFilterer missed")
	}

	boundContr := bind.NewBoundContract(common.HexToAddress(a.contractAddress), a.abi, nil, nil, a.filter)
	logs, sub, err := boundContr.WatchLogs(nil, "LogTokensLocked")
	if err != nil {
		return nil, err
	}

	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AbiLogTokensLocked)
				if err := boundContr.UnpackLog(event, "LogTokensLocked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

func (a AbiQtum) contractCall(methodName string, args ...interface{}) ([]interface{}, error) {
	contractData, err := a.abi.Pack(methodName, args...)
	if err != nil {
		return nil, err
	}

	resp, err := a.base.CallContract(&qtum.CallContractRequest{
		From:     "",
		To:       a.contractAddress,
		Data:     hex.EncodeToString(contractData),
		GasLimit: big.NewInt(defaultGasLimit),
	})
	if err != nil {
		return nil, err
	}

	if resp.ExecutionResult.Excepted != "None" {
		return nil, fmt.Errorf("ExecutionResult excepted: %s", resp.ExecutionResult.Excepted)
	}

	respData, err := hex.DecodeString(resp.ExecutionResult.Output)
	if err != nil {
		return nil, err
	}

	unPackedResp, err := a.abi.Unpack(methodName, respData)
	if err != nil {
		return nil, err
	}

	return unPackedResp, err
}
