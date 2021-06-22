package base

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/decred/dcrd/dcrec/secp256k1/v3"
	"github.com/decred/dcrd/dcrec/secp256k1/v3/ecdsa"
	"github.com/qtumproject/janus/pkg/qtum"
	"github.com/qtumproject/janus/pkg/utils"
	"github.com/qtumproject/qtumsuite"
	"github.com/qtumproject/qtumsuite/chaincfg"
	"github.com/qtumproject/qtumsuite/txscript"
	"github.com/qtumproject/qtumsuite/wire"
	"github.com/shopspring/decimal"
	"math/big"
)

const (
	defaultGasLimit = int64(2500000)
	defaultGasPrice = int64(40)
)

type QtumBase struct {
	m *qtum.Method
	//wif     *qtumsuite.WIF
	chainID string
}

func NewQtumBase(c *qtum.Client, chainID string) (*QtumBase, error) {
	return &QtumBase{
		m:       &qtum.Method{Client: c},
		chainID: chainID,
	}, nil
}

func (b QtumBase) GetChainID() (address string) {
	return b.chainID
}

func (b QtumBase) SendRawTx(signedTx string) (string, error) {
	req := qtum.SendRawTransactionRequest([1]string{signedTx})

	resp, err := b.m.SendRawTransaction(&req)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

func (b QtumBase) SentContractCallFromAddress(senderAddress, contractAddress string, contractArgs []byte) (rawTx string, pkScripts []string, err error) {
	unspent, _, err := b.GatherUTXOs(senderAddress)
	if err != nil {
		return "", nil, err
	}

	//Zero amount to call SC
	amount := decimal.New(0, 0)

	//Default values
	gasPrice := decimal.NewFromInt(defaultGasPrice)

	gasLimit, err := b.EstimateGas(senderAddress, contractAddress, contractArgs)
	if err != nil {
		return "", nil, err
	}

	neededBalance := calculateNeededAmount(amount, decimal.NewFromInt(gasLimit), gasPrice)

	usedUTXO, pkScripts, usedBalance, err := getRequiredUtxos(unspent, neededBalance)
	if err != nil {
		return "", nil, err
	}

	change, err := calculateChange(usedBalance, neededBalance)
	if err != nil {
		return "", nil, err
	}

	floatChange, _ := change.Mul(decimal.NewFromFloat(1e-8)).Float64()
	//TODO check
	//if !exact {
	//	return nil, fmt.Errorf("change not exact: %s", change)
	//}

	rawTx, err = b.ContractCallRawTxBuild(usedUTXO, &qtum.SendToContractRawRequest{
		ContractAddress: utils.RemoveHexPrefix(contractAddress),
		Datahex:         hex.EncodeToString(contractArgs),
		Amount:          amount,
		GasLimit:        big.NewInt(gasLimit),
		GasPrice:        gasPrice.Mul(decimal.NewFromFloat(1e-8)).String(),
	}, senderAddress, floatChange)
	if err != nil {
		return "", nil, err
	}

	return rawTx, pkScripts, nil
}

func (b QtumBase) SignTx(wif *qtumsuite.WIF, rawTx string, sourcePkScript []string) (string, error) {

	bt, err := hex.DecodeString(rawTx)
	if err != nil {
		return "", err
	}

	redeemTx := &wire.MsgTx{}

	rd := bytes.NewReader(bt)

	err = redeemTx.Deserialize(rd)
	if err != nil {
		return "", err
	}

	for i := range redeemTx.TxIn {

		//Generate signature script
		script, err := hex.DecodeString(sourcePkScript[i])
		if err != nil {
			return "", err
		}

		signatureHash, err := txscript.CalcSignatureHash(script, txscript.SigHashAll, redeemTx, i)
		if err != nil {
			return "", err
		}

		privKey := secp256k1.PrivKeyFromBytes(wif.PrivKey.Serialize())
		signature := ecdsa.Sign(privKey, signatureHash)

		signatureScript, err := txscript.NewScriptBuilder().AddData(append(signature.Serialize(), byte(txscript.SigHashAll))).AddData(wif.SerializePubKey()).Script()
		if err != nil {
			return "", err
		}

		redeemTx.TxIn[i].SignatureScript = signatureScript
	}

	buf := bytes.NewBuffer(make([]byte, 0, redeemTx.SerializeSize()))
	redeemTx.Serialize(buf)

	hexSignedTx := hex.EncodeToString(buf.Bytes())

	return hexSignedTx, nil
}

func (b QtumBase) EstimateGas(senderAddress, contractAddress string, contractCallData []byte) (gasLimit int64, err error) {

	//Default value
	gasLimit = defaultGasLimit
	//return
	resp, err := b.m.CallContract(&qtum.CallContractRequest{
		From:     senderAddress,
		To:       contractAddress,
		Data:     hex.EncodeToString(contractCallData),
		GasLimit: big.NewInt(gasLimit),
	})
	if err != nil {
		return 0, err
	}

	if resp.ExecutionResult.Excepted == "Revert" {
		return 0, fmt.Errorf("Execution error: %s", resp.ExecutionResult.ExceptedMessage)
	}

	//TODO count gas limit more intelligently
	//Add 5k for safe
	gasLimit = int64(resp.ExecutionResult.GasUsed) + 5000

	return gasLimit, nil
}

func (b QtumBase) CallContract(req *qtum.CallContractRequest) (resp *qtum.CallContractResponse, err error) {
	return b.m.CallContract(req)
}

func (b QtumBase) ContractCallRawTxBuild(usedUTXO []qtum.RawTxInputs, contractInteractTx *qtum.SendToContractRawRequest, feePayerAddress string, floatChange float64) (rawTx string, err error) {
	rawtxreq := []interface{}{usedUTXO, []interface{}{map[string]*qtum.SendToContractRawRequest{"contract": contractInteractTx}, map[string]float64{feePayerAddress: floatChange}}}

	//Node tx building used to escape hold node constants in code
	if err = b.m.Request(qtum.MethodCreateRawTx, rawtxreq, &rawTx); err != nil {
		return "", err
	}

	return rawTx, nil
}

func GetAddressFromWIF(wif *qtumsuite.WIF, chainID string) (string, error) {
	// Generate Address from Public Key
	addressPubKey, err := qtumsuite.NewAddressPubKey(wif.SerializePubKey(), getNetParamsByChainID(chainID))
	if err != nil {
		return "", err
	}

	return addressPubKey.EncodeAddress(), nil
}

func calculateNeededAmount(value, gasLimit, gasPrice decimal.Decimal) decimal.Decimal {
	return value.Add(gasLimit.Mul(gasPrice))
}

func calculateChange(balance, neededAmount decimal.Decimal) (decimal.Decimal, error) {
	if balance.LessThan(neededAmount) {
		return decimal.Decimal{}, fmt.Errorf("insufficient funds to create fee to chain")
	}
	return balance.Sub(neededAmount), nil
}

var qtumTestNetParams = chaincfg.TestNet3Params

func getNetParamsByChainID(chainID string) *chaincfg.Params {
	switch chainID {
	case qtum.ChainMain:
		return &chaincfg.MainNetParams
	case qtum.ChainTest:
		return &chaincfg.TestNet3Params
	case qtum.ChainRegTest:
		return &chaincfg.RegressionNetParams
	}

	return &chaincfg.MainNetParams
}
