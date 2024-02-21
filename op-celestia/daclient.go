package celestia

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/rollkit/go-da/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DAClient implements DAStorage with celestia backend
type DAClient struct {
	Log        log.Logger
	Client     *proxy.Client
	GetTimeout time.Duration
}

// NewDAClient returns a celestia DA client.
func NewDAClient(rpc string, verify bool) *DAClient {
	client := proxy.NewClient()
	client.Start(rpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return &DAClient{
		Log:        log.New(),
		Client:     client,
		GetTimeout: time.Minute,
	}
}

func (c *DAClient) GetInput(ctx context.Context, key []byte) ([]byte, error) {
	var out []byte
	switch key[0] {
	case DerivationVersionCelestia:
		log.Info("celestia: blob request", "id", hex.EncodeToString(key))
		ctx, cancel := context.WithTimeout(context.Background(), c.GetTimeout)
		blobs, err := c.Client.Get(ctx, [][]byte{key[1:]})
		cancel()
		if err != nil || len(blobs) == 0 {
			return nil, fmt.Errorf("celestia: failed to resolve frame: %w", err)
		}
		if len(blobs) != 1 {
			c.Log.Warn("celestia: unexpected length for blobs", "expected", 1, "got", len(blobs))
		}
		out = blobs[0]
	default:
		out = key
		log.Info("celestia: using eth fallback")
	}
	return out, nil
}

func (c *DAClient) SetInput(ctx context.Context, data []byte) ([]byte, error) {
	ids, _, err := c.Client.Submit(ctx, [][]byte{data}, -1)
	var key []byte
	if err == nil && len(ids) == 1 {
		c.Log.Info("celestia: blob successfully submitted", "id", hex.EncodeToString(ids[0]))
		key = append([]byte{DerivationVersionCelestia}, ids[0]...)
	} else {
		key = data
		c.Log.Info("celestia: blob submission failed; falling back to eth", "err", err)
	}
	return key, nil
}
