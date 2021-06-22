package abi

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/qtumproject/janus/pkg/qtum"
	"github.com/qtumproject/janus/pkg/utils"
)

type Log struct {
	Entries   []LogEntry `json:"entries"`
	Count     uint64     `json:"count"`
	Nextblock uint64     `json:"nextblock"`
}

type LogEntry struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      uint64 `json:"blockNumber"`
	TransactionHash  string `json:"transactionHash"`
	TransactionIndex uint64 `json:"transactionIndex"`
	From             string `json:"from"`
	// NOTE: will be null for a contract creation transaction
	To                string `json:"to"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed"`
	GasUsed           uint64 `json:"gasUsed"`
	ContractAddress   string `json:"contractAddress"`

	Excepted        string `json:"excepted"`
	ExceptedMessage string `json:"exceptedMessage"`
	Bloom           string `json:"bloom"`
	StateRoot       string `json:"stateRoot"`
	UtxoRoot        string `json:"utxoRoot"`

	Topics []string `json:"topics"`
	Data   string   `json:"data"`
}

type Filter struct {
	Addresses []string `json:"addresses,omitempty"`
	Topics    []string `json:"topics,omitempty"`
}

type Params struct {
	FromBlock string `json:"fromBlock"`
	ToBlock   string `json:"toBlock"`
	Filter    Filter `json:"filter"`
}

//Qtum implementation of ContractFilterer interface
type qtumContractFilterer struct {
	rpcURL    string
	isMainNet bool
	//client        *qtum.Client
	confirmations uint64
}

func NewFilterer(rpcURL, chainID string, confirmations uint64) (*qtumContractFilterer, error) {

	if rpcURL == "" {
		return nil, fmt.Errorf("empty rpcURL")
	}

	if chainID == "" {
		return nil, fmt.Errorf("empty chainID")
	}

	if confirmations == 0 {
		return nil, fmt.Errorf("wrong confirmations num")
	}

	return &qtumContractFilterer{
		rpcURL:        rpcURL,
		isMainNet:     chainID == qtum.ChainMain,
		confirmations: confirmations,
	}, nil
}

func (f qtumContractFilterer) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {

	//Remove 0x prefix
	addresses := make([]string, len(query.Addresses))
	for i, address := range query.Addresses {
		addresses[i] = utils.RemoveHexPrefix(address.String())
	}

	//Make at least 1 topic for each
	topics := make([]string, 0, len(query.Topics))

	//Remove 0x prefix
	for i, topicCompact := range query.Topics {
		for j := range query.Topics[i] {
			topics = append(topics, utils.RemoveHexPrefix(topicCompact[j].String()))
		}
	}

	return event.NewSubscription(func(quit <-chan struct{}) (err error) {
		lCh := make(chan types.Log)
		errCh := make(chan error)

		client, err := qtum.NewClient(f.isMainNet, f.rpcURL)
		if err != nil {
			return fmt.Errorf("dialing qtum client failed: %w", err)
		}

		go func() {

			var log Log
			var err error

			//waits for logs in the future matching the specified conditions
			params := []interface{}{nil, nil, Filter{
				Addresses: addresses,
				Topics:    topics,
			},
				//6,
				f.confirmations,
			}

			for {

				if err = client.Request(qtum.MethodWaitForLogs, params, &log); err != nil {
					//logger.Error("Qtum wait for logs error", zap.Error(err))

					errCh <- err
					return
				}

				//Context closed
				if ctx.Err() != nil {
					return
				}

				var logTopics []common.Hash

				for i := range log.Entries {

					for j := range log.Entries[i].Topics {
						logTopics = append(logTopics, common.HexToHash(log.Entries[i].Topics[j]))
					}

					data, err := hex.DecodeString(log.Entries[i].Data)
					if err != nil {
						continue
					}

					lCh <- types.Log{
						BlockNumber: log.Entries[i].BlockNumber,
						TxHash:      common.HexToHash(log.Entries[i].TransactionHash),
						Topics:      logTopics,
						Data:        data,
					}
				}

			}
		}()

		var log types.Log

		for {
			select {
			case log = <-lCh:
				ch <- log
			case err = <-errCh:
				return err
			case <-ctx.Done():
				return ctx.Err()
			case <-quit:
				return nil
			}
		}

	}), nil
}

func (f qtumContractFilterer) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return nil, fmt.Errorf("not implemented")
}
