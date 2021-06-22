package qtum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/readiness"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"go.uber.org/zap"
	"math/big"
	"time"

	"github.com/certusone/wormhole/bridge/pkg/qtum/abi"
	"github.com/qtumproject/janus/pkg/qtum"

	//gossipv1 "github.com/certusone/wormhole/proto/gossip/v1"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/common"
)

var (
	qtumConnectionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_qtum_connection_errors_total",
			Help: "Total number of Qtum connection errors (either during initial connection or while watching)",
		}, []string{"reason"})

	qtumLockupsFound = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_qtum_lockups_found_total",
			Help: "Total number of Qtum lockups found (pre-confirmation)",
		})
	qtumLockupsConfirmed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_qtum_lockups_confirmed_total",
			Help: "Total number of Qtum lockups verified (post-confirmation)",
		})
	guardianSetChangesConfirmed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_qtum_guardian_set_changes_confirmed_total",
			Help: "Total number of guardian set changes verified (we only see confirmed ones to begin with)",
		})
	currentQtumHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_qtum_current_height",
			Help: "Current Qtum block height",
		})
	queryLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_qtum_query_latency",
			Help: "Latency histogram for Qtum calls (note that most interactions are streaming queries, NOT calls, and we cannot measure latency for those",
		}, []string{"operation"})
)

func init() {
	prometheus.MustRegister(qtumConnectionErrors)
	prometheus.MustRegister(qtumLockupsFound)
	prometheus.MustRegister(qtumLockupsConfirmed)
	prometheus.MustRegister(guardianSetChangesConfirmed)
	prometheus.MustRegister(currentQtumHeight)
	prometheus.MustRegister(queryLatency)
}

type (
	QtumBridgeWatcher struct {
		url              string
		bridge           string
		minConfirmations uint64
		chainID          string

		pendingLocks      map[eth_common.Hash]*pendingLock
		pendingLocksGuard sync.Mutex

		lockChan chan *common.ChainLock
		setChan  chan *common.GuardianSet
	}

	pendingLock struct {
		lock   *common.ChainLock
		height uint64
	}
)

func NewQtumBridgeWatcher(url, bridge, chainID string, minConfirmations uint64, lockEvents chan *common.ChainLock, setEvents chan *common.GuardianSet) *QtumBridgeWatcher {
	return &QtumBridgeWatcher{url: url, bridge: bridge, chainID: chainID, minConfirmations: minConfirmations, lockChan: lockEvents, setChan: setEvents, pendingLocks: map[eth_common.Hash]*pendingLock{}}
}

func (e *QtumBridgeWatcher) Run(ctx context.Context) error {
	// Initialize gossip metrics (we want to broadcast the address even if we're not yet syncing)
	p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDQtum, &gossipv1.Heartbeat_Network{
		BridgeAddress: e.bridge,
	})

	filterer, err := abi.NewFilterer(e.url, e.chainID, e.minConfirmations)
	if err != nil {
		return err
	}

	qtumABI, err := abi.NewAbiQtum(e.url, e.bridge, e.chainID, filterer)
	if err != nil {
		return err
	}

	// Timeout for initializing subscriptions
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Subscribe to new token lockups
	tokensLockedC := make(chan *abi.AbiLogTokensLocked, 2)

	tokensLockedSub, err := qtumABI.WatchLogTokensLocked(timeout, tokensLockedC)
	if err != nil {
		qtumConnectionErrors.WithLabelValues("subscribe_error").Inc()
		return fmt.Errorf("failed to subscribe to token lockup events: %w", err)
	}

	// Subscribe to guardian set changes
	guardianSetC := make(chan *abi.AbiLogGuardianSetChanged, 2)

	guardianSetEvent, err := qtumABI.WatchLogGuardianSetChanged(timeout, guardianSetC)
	if err != nil {
		qtumConnectionErrors.WithLabelValues("subscribe_error").Inc()
		return fmt.Errorf("failed to subscribe to guardian set events: %w", err)
	}

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	// Get initial validator set from Qtum. We could also fetch it from Solana,
	// because both sets are synchronized, we simply made an arbitrary decision to use Qtum.
	timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	idx, gs, err := FetchCurrentGuardianSet(timeout, e.url, e.chainID, e.bridge)
	if err != nil {
		qtumConnectionErrors.WithLabelValues("guardian_set_fetch_error").Inc()
		return fmt.Errorf("failed requesting guardian set from Qtum: %w", err)
	}

	logger.Info("qtum initial guardian set fetched", zap.Any("value", gs), zap.Uint32("index", idx))
	e.setChan <- &common.GuardianSet{
		Keys:  gs.Keys,
		Index: idx,
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-tokensLockedSub.Err():
				qtumConnectionErrors.WithLabelValues("subscription_error").Inc()
				errC <- fmt.Errorf("error while processing token lockup subscription: %w", e)
				return
			case e := <-guardianSetEvent.Err():
				qtumConnectionErrors.WithLabelValues("subscription_error").Inc()
				errC <- fmt.Errorf("error while processing guardian set subscription: %w", e)
				return
			case ev := <-tokensLockedC:
				// Request timestamp for block
				msm := time.Now()
				timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
				b, err := BlockByNumber(e.url, e.chainID, big.NewInt(int64(ev.Raw.BlockNumber)))
				cancel()
				queryLatency.WithLabelValues("block_by_number").Observe(time.Since(msm).Seconds())

				if err != nil {
					qtumConnectionErrors.WithLabelValues("block_by_number_error").Inc()
					errC <- fmt.Errorf("failed to request timestamp for block %d: %w", ev.Raw.BlockNumber, err)
					return
				}

				lock := &common.ChainLock{
					TxHash:        ev.Raw.TxHash,
					Timestamp:     time.Unix(int64(b.Time), 0),
					Nonce:         ev.Nonce,
					SourceAddress: ev.Sender,
					TargetAddress: ev.Recipient,
					SourceChain:   vaa.ChainIDQtum,
					TargetChain:   vaa.ChainID(ev.TargetChain),
					TokenChain:    vaa.ChainID(ev.TokenChain),
					TokenAddress:  ev.Token,
					TokenDecimals: ev.TokenDecimals,
					Amount:        ev.Amount,
				}

				logger.Info("found new lockup transaction", zap.Stringer("tx", ev.Raw.TxHash),
					zap.Uint64("block", ev.Raw.BlockNumber))

				qtumLockupsFound.Inc()

				e.pendingLocksGuard.Lock()
				e.pendingLocks[ev.Raw.TxHash] = &pendingLock{
					lock:   lock,
					height: ev.Raw.BlockNumber,
				}
				e.pendingLocksGuard.Unlock()
			case ev := <-guardianSetC:
				logger.Info("guardian set has changed, fetching new value",
					zap.Uint32("new_index", ev.NewGuardianIndex))

				guardianSetChangesConfirmed.Inc()

				msm := time.Now()
				timeout, cancel = context.WithTimeout(ctx, 15*time.Second)
				gs, err := qtumABI.GetGuardianSet(ev.NewGuardianIndex)
				cancel()
				queryLatency.WithLabelValues("get_guardian_set").Observe(time.Since(msm).Seconds())
				if err != nil {
					// We failed to process the guardian set update and are now out of sync with the chain.
					// Recover by crashing the runnable, which causes the guardian set to be re-fetched.
					errC <- fmt.Errorf("error requesting new guardian set value for %d: %w", ev.NewGuardianIndex, err)
					return
				}

				logger.Info("new guardian set fetched", zap.Any("value", gs), zap.Uint32("index", ev.NewGuardianIndex))
				e.setChan <- &common.GuardianSet{
					Keys:  gs.Keys,
					Index: ev.NewGuardianIndex,
				}
			}
		}
	}()

	// Watch headers
	headSink := make(chan *qtum.GetBlockHeaderResponse, 2)

	headerSubscription, err := SubscribeNewHead(e.url, e.chainID, e.minConfirmations, ctx, headSink)
	if err != nil {
		return fmt.Errorf("failed to subscribe to header events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-headerSubscription.Err():
				errC <- fmt.Errorf("error while processing header subscription: %w", e)
				return
			case ev := <-headSink:
				start := time.Now()
				logger.Info("processing new header", zap.Int("block", ev.Height))
				currentQtumHeight.Set(float64(ev.Height))
				readiness.SetReady(common.ReadinessQtumSyncing)
				p2p.DefaultRegistry.SetNetworkStats(vaa.ChainIDQtum, &gossipv1.Heartbeat_Network{
					Height:        int64(ev.Height),
					BridgeAddress: e.bridge,
				})

				e.pendingLocksGuard.Lock()

				blockNumberU := uint64(ev.Height)
				for hash, pLock := range e.pendingLocks {

					// Transaction was dropped and never picked up again
					if pLock.height+4*e.minConfirmations <= blockNumberU {
						logger.Debug("lockup timed out", zap.Stringer("tx", pLock.lock.TxHash),
							zap.Int("block", ev.Height))
						delete(e.pendingLocks, hash)
						continue
					}

					// Transaction is now ready
					if pLock.height+e.minConfirmations <= uint64(ev.Height) {
						logger.Debug("lockup confirmed", zap.Stringer("tx", pLock.lock.TxHash),
							zap.Int("block", ev.Height))
						delete(e.pendingLocks, hash)
						e.lockChan <- pLock.lock
						qtumLockupsConfirmed.Inc()
					}
				}

				e.pendingLocksGuard.Unlock()
				logger.Info("processed new header", zap.Int("block", ev.Height),
					zap.Duration("took", time.Since(start)))
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// Fetch the current guardian set ID and guardian set from the chain.
func FetchCurrentGuardianSet(ctx context.Context, rpcURL, chainID, bridgeContract string) (uint32, *abi.WormholeGuardianSet, error) {

	abiQtum, err := abi.NewAbiQtum(rpcURL, bridgeContract, chainID, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("dialing qtum client failed: %w", err)
	}

	currentIndex, err := abiQtum.GuardianSetIndex()
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}

	gs, err := abiQtum.GetGuardianSet(currentIndex)
	if err != nil {
		return 0, nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}

	return currentIndex, gs, nil
}
