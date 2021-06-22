package base

import (
	"fmt"
	"github.com/qtumproject/janus/pkg/qtum"
	"github.com/shopspring/decimal"
)

const (
	minQtumSafetyConfirmationNum = 10
	maxConfirmationNum           = 99999999
	qtumPrecision                = 1e8
)

func (b QtumBase) GatherUTXOs(address string) (*qtum.ListUnspentResponse, int64, error) {

	//Get UTXOs from network
	//Use the UTXOs to figure out the previousTxId as well as the pubKeyScript
	req := &qtum.ListUnspentRequest{
		MinConf:      minQtumSafetyConfirmationNum,
		MaxConf:      maxConfirmationNum,
		Addresses:    []string{address},
		QueryOptions: qtum.ListUnspentQueryOptions{},
	}

	listUnspentResp, err := b.m.ListUnspent(req)
	if err != nil {
		return nil, 0, err
	}

	balance := decimal.NewFromFloat(0)
	for _, utxo := range *listUnspentResp {
		balance = balance.Add(utxo.Amount)
	}

	balance = balance.Mul(decimal.NewFromFloat(1e8))
	floatBalance, exact := balance.Float64()
	if exact != true {
		return nil, 0, fmt.Errorf("Float exact error")
	}

	return listUnspentResp, int64(floatBalance), nil
}

func getRequiredUtxos(unspentList *qtum.ListUnspentResponse, neededAmount decimal.Decimal) (inputs []qtum.RawTxInputs, pkScripts []string, balance decimal.Decimal, err error) {

	// need to get utxos with txid and vouts. In order to do this we get a list of unspent transactions and begin summing them up
	balance = decimal.New(0, 0)

	for _, utxo := range *unspentList {
		balance = balance.Add(utxo.Amount.Mul(decimal.NewFromFloat(qtumPrecision)))
		inputs = append(inputs, qtum.RawTxInputs{TxID: utxo.Txid, Vout: utxo.Vout})
		pkScripts = append(pkScripts, utxo.ScriptPubKey)

		if balance.GreaterThanOrEqual(neededAmount) {
			return inputs, pkScripts, balance, nil
		}
	}

	return nil, nil, decimal.Decimal{}, fmt.Errorf("Insufficient UTXO value attempted to be sent")
}
