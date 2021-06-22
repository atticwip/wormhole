package qtum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/qtumproject/janus/pkg/qtum"
	"math/big"
	"time"
)

// PadAddress creates 32-byte VAA.Address from 20-byte Ethereum addresses by adding 12 0-bytes at the left
func PadAddress(address common.Address) vaa.Address {
	paddedAddress := common.LeftPadBytes(address[:], 32)

	addr := vaa.Address{}
	copy(addr[:], paddedAddress)

	return addr
}

func BlockByNumber(url, chainID string, blockNum *big.Int) (*qtum.GetBlockResponse, error) {

	client, err := qtum.NewClient(chainID == qtum.ChainMain, url)
	if err != nil {
		return nil, fmt.Errorf("dialing qtum client failed: %w", err)
	}

	m := &qtum.Method{Client: client}

	hash, err := m.GetBlockHash(blockNum)
	if err != nil {
		return nil, err
	}

	block, err := m.GetBlock(string(hash))
	if err != nil {
		return nil, err
	}

	return block, nil
}

const cache = 500

func SubscribeNewHead(url, chainID string, confirmations uint64, ctx context.Context, headSink chan *qtum.GetBlockHeaderResponse) (ethereum.Subscription, error) {

	client, err := qtum.NewClient(chainID == qtum.ChainMain, url)
	if err != nil {
		return nil, fmt.Errorf("dialing qtum client failed: %w", err)
	}

	m := &qtum.Method{Client: client}

	return event.NewSubscription(func(quit <-chan struct{}) error {

		blocksCache := map[int64]string{}

		var prevBlocksLen int64

		for {
			select {
			case <-time.Tick(1 * time.Second):

				blocks, err := m.GetBlockCount()
				if err != nil {
					return fmt.Errorf("SubscribeNewHead error: %s", err)
				}

				//No new blocks
				if prevBlocksLen == blocks.Int64() {
					continue
				}

				currentHash, err := m.GetBlockHash(blocks.Int)
				if err != nil {
					return fmt.Errorf("GetBlockHash error: %s", err)
				}

				currentHeader, err := m.GetBlockHeader(string(currentHash))
				if err != nil {
					return fmt.Errorf("GetBlockHash error: %s", err)
				}

				//If hashes on equals and blocksCache initialized
				if cachedBlockHash, ok := blocksCache[prevBlocksLen]; cachedBlockHash != currentHeader.Previousblockhash && ok {
					//Reorg chain
					head := prevBlocksLen - cache

					for head <= prevBlocksLen {
						hash, err := m.GetBlockHash(big.NewInt(head))
						if err != nil {
							return fmt.Errorf("GetBlockHash lastBeforeFork error: %s", err)
						}

						if blocksCache[head] != string(hash) {
							//Update cache
							blocksCache[head] = string(hash)
							headSink <- &qtum.GetBlockHeaderResponse{
								Hash:   string(hash),
								Height: int(head),
							}
						}

						head++
					}

				}

				//Push last block
				blocksCache[int64(currentHeader.Height)] = currentHeader.Hash
				delete(blocksCache, prevBlocksLen-cache)
				headSink <- currentHeader

			case <-ctx.Done():
				return ctx.Err()
			case <-quit:
				return nil
			}
		}

	}), nil
}
