package qtum

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/qtum/abi"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"go.uber.org/zap"
	"io/ioutil"
)

// SubmitVAA prepares transaction with signed VAA and sends it to the Qtum blockchain
func SubmitVAA(ctx context.Context, urlRPC string, chainID string, contractAddress string, feePayerKey string, signed *vaa.VAA) (string, error) {

	// Serialize VAA
	vaaBytes, err := signed.Marshal()
	if err != nil {
		return "", err
	}

	supervisor.Logger(ctx).Info("submitted VAA to Qtum", zap.Binary("binary", vaaBytes))

	qtumABI, err := abi.NewAbiQtum(urlRPC, contractAddress, chainID, nil)
	if err != nil {
		return "", fmt.Errorf("new abi qtum failed: %w", err)
	}

	return qtumABI.SubmitVAA(feePayerKey, vaaBytes)
}

// ReadKey reads file and returns its content as a string
func ReadKey(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// WriteDevnetKey writes default devnet key to the file
func WriteDevnetKey(path string) {
	err := ioutil.WriteFile(path, []byte(devnet.QtumFeePayerKey), 0600)
	if err != nil {
		panic("Cannot write Terra key file")
	}
}
