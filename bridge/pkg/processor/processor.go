package processor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/qtum"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/terra"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type (
	// vaaState represents the local view of a given VAA
	vaaState struct {
		// First time this digest was seen (possibly even before we saw its lockup).
		firstObserved time.Time
		// Copy of the VAA we constructed when we saw the lockup.
		ourVAA *vaa.VAA
		// Map of signatures seen by guardian. During guardian set updates, this may contain signatures belonging
		// to either the old or new guardian set.
		signatures map[ethcommon.Address][]byte
		// Flag set after reaching quorum and submitting the VAA.
		submitted bool
		// Flag set by the cleanup service after the settlement timeout has expired and misses were counted.
		settled bool
		// Human-readable description of the VAA's source, used for metrics.
		source string
		// Number of times the cleanup service has attempted to retransmit this VAA.
		retryCount uint
		// Copy of the bytes we submitted (ourVAA, but signed and serialized). Used for retransmissions.
		ourMsg []byte
		// Copy of the guardian set valid at lockup/injection time.
		gs *common.GuardianSet
	}

	vaaMap map[string]*vaaState

	// aggregationState represents the node's aggregation of guardian signatures.
	aggregationState struct {
		vaaSignatures vaaMap
	}
)

type Processor struct {
	// lockC is a channel of observed chain lockups
	lockC chan *common.ChainLock
	// setC is a channel of guardian set updates
	setC chan *common.GuardianSet

	// sendC is a channel of outbound messages to broadcast on p2p
	sendC chan []byte
	// obsvC is a channel of inbound decoded observations from p2p
	obsvC chan *gossipv1.SignedObservation

	// vaaC is a channel of VAAs to submit to store on Solana (either as target, or for data availability)
	vaaC chan *vaa.VAA

	// injectC is a channel of VAAs injected locally.
	injectC chan *vaa.VAA

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	// devnetMode specified whether to submit transactions to the hardcoded Ethereum devnet
	devnetMode         bool
	devnetNumGuardians uint
	devnetEthRPC       string

	terraEnabled  bool
	terraLCD      string
	terraChainID  string
	terraContract string
	terraFeePayer string

	qtumEnabled  bool
	qtumRPC      string
	qtumChainID  string
	qtumContract string
	qtumFeePayer string

	logger *zap.Logger

	// Runtime state

	// gs is the currently valid guardian set
	gs *common.GuardianSet
	// state is the current runtime VAA view
	state *aggregationState
	// gk pk as eth address
	ourAddr ethcommon.Address
	// cleanup triggers periodic state cleanup
	cleanup *time.Ticker
}

func NewProcessor(
	ctx context.Context,
	lockC chan *common.ChainLock,
	setC chan *common.GuardianSet,
	sendC chan []byte,
	obsvC chan *gossipv1.SignedObservation,
	vaaC chan *vaa.VAA,
	injectC chan *vaa.VAA,
	gk *ecdsa.PrivateKey,
	devnetMode bool,
	devnetNumGuardians uint,
	devnetEthRPC string,
	terraEnabled bool,
	terraLCD string,
	terraChainID string,
	terraContract string,
	terraFeePayer string,
	qtumEnabled bool,
	qtumRPC string,
	qtumChainID string,
	qtumContract string,
	qtumFeePayer string,
) *Processor {

	return &Processor{
		lockC:              lockC,
		setC:               setC,
		sendC:              sendC,
		obsvC:              obsvC,
		vaaC:               vaaC,
		injectC:            injectC,
		gk:                 gk,
		devnetMode:         devnetMode,
		devnetNumGuardians: devnetNumGuardians,
		devnetEthRPC:       devnetEthRPC,

		terraEnabled:  terraEnabled,
		terraLCD:      terraLCD,
		terraChainID:  terraChainID,
		terraContract: terraContract,
		terraFeePayer: terraFeePayer,

		qtumEnabled:  qtumEnabled,
		qtumRPC:      qtumRPC,
		qtumChainID:  qtumChainID,
		qtumContract: qtumContract,
		qtumFeePayer: qtumFeePayer,

		logger:  supervisor.Logger(ctx),
		state:   &aggregationState{vaaMap{}},
		ourAddr: crypto.PubkeyToAddress(gk.PublicKey),
	}
}

func (p *Processor) Run(ctx context.Context) error {
	p.cleanup = time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.gs = <-p.setC:
			p.logger.Info("guardian set updated",
				zap.Strings("set", p.gs.KeysAsHexStrings()),
				zap.Uint32("index", p.gs.Index))

			// Dev mode guardian set update check (no-op in production)
			err := p.checkDevModeGuardianSetUpdate(ctx)
			if err != nil {
				return err
			}
		case k := <-p.lockC:
			p.handleLockup(ctx, k)
		case v := <-p.injectC:
			p.handleInjection(ctx, v)
		case m := <-p.obsvC:
			p.handleObservation(ctx, m)
		case <-p.cleanup.C:
			p.handleCleanup(ctx)
		}
	}
}

func (p *Processor) checkDevModeGuardianSetUpdate(ctx context.Context) error {
	if p.devnetMode {
		if uint(len(p.gs.Keys)) != p.devnetNumGuardians {
			v := devnet.DevnetGuardianSetVSS(p.devnetNumGuardians)

			p.logger.Info(fmt.Sprintf("guardian set has %d members, expecting %d - submitting VAA",
				len(p.gs.Keys), p.devnetNumGuardians),
				zap.Any("v", v))

			timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			trx, err := devnet.SubmitVAA(timeout, p.devnetEthRPC, v)
			if err != nil {
				// Either Ethereum is not yet up, or another node has already submitted - bail
				// and let another node handle it. We only check the guardian set on Ethereum,
				// so we use that to sequence devnet creation for Terra and Solana as well.
				return fmt.Errorf("failed to submit Eth devnet guardian set change: %v", err)
			}

			p.logger.Info("devnet guardian set change submitted to Ethereum", zap.Any("trx", trx), zap.Any("vaa", v))

			if p.terraEnabled {
				// Submit to Terra
				go func() {
					for {
						timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
						trxResponse, err := terra.SubmitVAA(timeout, p.terraLCD, p.terraChainID, p.terraContract, p.terraFeePayer, v)
						if err != nil {
							cancel()
							p.logger.Error("failed to submit Terra devnet guardian set change, retrying", zap.Error(err))
							time.Sleep(1 * time.Second)
							continue
						}
						cancel()
						p.logger.Info("devnet guardian set change submitted to Terra", zap.Any("trxResponse", trxResponse), zap.Any("vaa", v))
						break
					}
				}()
			}

			if p.qtumEnabled {
				// Submit to qtum
				go func() {
					for {
						timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
						trxResponse, err := qtum.SubmitVAA(timeout, p.qtumRPC, p.qtumChainID, p.qtumContract, p.qtumFeePayer, v)
						if err != nil {
							cancel()
							p.logger.Error("failed to submit Qtum devnet guardian set change, retrying", zap.Error(err))
							time.Sleep(1 * time.Second)
							continue
						}
						cancel()
						p.logger.Info("devnet guardian set change submitted to Qtum", zap.Any("trxResponse", trxResponse), zap.Any("vaa", v))
						break
					}
				}()
			}

			// Submit VAA to Solana as well. This is asynchronous and can fail, leading to inconsistent devnet state.
			p.vaaC <- v
		}
	}

	return nil
}
