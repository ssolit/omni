package ethclient

import (
	"context"
	"math/big"
	"net/http"
	"time"

	"github.com/omni-network/omni/lib/errors"
	"github.com/omni-network/omni/lib/log"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultRPCHTTPTimeout = time.Second * 30

	newPayloadV2 = "engine_newPayloadV2"
	newPayloadV3 = "engine_newPayloadV3"

	forkchoiceUpdatedV2 = "engine_forkchoiceUpdatedV2"
	forkchoiceUpdatedV3 = "engine_forkchoiceUpdatedV3"

	getPayloadV2 = "engine_getPayloadV2"
	getPayloadV3 = "engine_getPayloadV3"
)

// EngineClient defines the Engine API authenticated JSON-RPC endpoints.
// It extends the normal Client interface with the Engine API.
type EngineClient interface {
	Client

	// NewPayloadV3 creates an Eth1 block, inserts it in the chain, and returns the status of the chain.
	NewPayloadV3(ctx context.Context, params engine.ExecutableData, versionedHashes []common.Hash,
		beaconRoot *common.Hash) (engine.PayloadStatusV1, error)

	// ForkchoiceUpdatedV3 is equivalent to V2 with the addition of parent beacon block root in the payload attributes.
	ForkchoiceUpdatedV3(ctx context.Context, update engine.ForkchoiceStateV1,
		payloadAttributes *engine.PayloadAttributes) (engine.ForkChoiceResponse, error)

	// GetPayloadV3 returns a cached payload by id.
	GetPayloadV3(ctx context.Context, payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
}

// engineClient implements EngineClient using JSON-RPC.
type engineClient struct {
	Wrapper
}

// NewAuthClient returns a new authenticated JSON-RPc engineClient.
func NewAuthClient(ctx context.Context, urlAddr string, jwtSecret []byte) (EngineClient, error) {
	transport := http.DefaultTransport
	if len(jwtSecret) > 0 {
		transport = newJWTRoundTripper(http.DefaultTransport, jwtSecret)
	}

	client := &http.Client{Timeout: defaultRPCHTTPTimeout, Transport: transport}

	rpcClient, err := rpc.DialOptions(ctx, urlAddr, rpc.WithHTTPClient(client))
	if err != nil {
		return engineClient{}, errors.Wrap(err, "rpc dial")
	}

	return engineClient{
		Wrapper: NewClient(rpcClient, "engine", urlAddr),
	}, nil
}

//go:generate go run github.com/fjl/gencodec -type RethPayloadV3 -field-override rethPayloadV3Marshaling -out gen_reth_payload_v3.go

type RethPayloadV3 struct {
	ParentHash    common.Hash         `json:"parentHash"    gencodec:"required"`
	FeeRecipient  common.Address      `json:"feeRecipient"  gencodec:"required"`
	StateRoot     common.Hash         `json:"stateRoot"     gencodec:"required"`
	ReceiptsRoot  common.Hash         `json:"receiptsRoot"  gencodec:"required"`
	LogsBloom     []byte              `json:"logsBloom"     gencodec:"required"`
	Random        common.Hash         `json:"prevRandao"    gencodec:"required"`
	Number        uint64              `json:"blockNumber"   gencodec:"required"`
	GasLimit      uint64              `json:"gasLimit"      gencodec:"required"`
	GasUsed       uint64              `json:"gasUsed"       gencodec:"required"`
	Timestamp     uint64              `json:"timestamp"     gencodec:"required"`
	ExtraData     []byte              `json:"extraData"     gencodec:"required"`
	BaseFeePerGas *big.Int            `json:"baseFeePerGas" gencodec:"required"`
	BlockHash     common.Hash         `json:"blockHash"     gencodec:"required"`
	Transactions  [][]byte            `json:"transactions"  gencodec:"required"`
	Withdrawals   []*types.Withdrawal `json:"withdrawals"`
	BlobGasUsed   *uint64             `json:"blobGasUsed"`
	ExcessBlobGas *uint64             `json:"excessBlobGas"`
	// Deposits         types.Deposits          `json:"depositRequests"`
	ExecutionWitness *types.ExecutionWitness `json:"executionWitness,omitempty"`
}

type rethPayloadV3Marshaling struct {
	Number        hexutil.Uint64
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Timestamp     hexutil.Uint64
	BaseFeePerGas *hexutil.Big
	ExtraData     hexutil.Bytes
	LogsBloom     hexutil.Bytes
	Transactions  []hexutil.Bytes
	BlobGasUsed   *hexutil.Uint64
	ExcessBlobGas *hexutil.Uint64
}

// ConvertExecutableDataToRethPayloadV3 converts ExecutableData to RethPayloadV3.
func ConvertExecutableDataToRethPayloadV3(ed engine.ExecutableData) RethPayloadV3 {
	var baseFee *big.Int
	if ed.BaseFeePerGas != nil {
		baseFee = new(big.Int).Set(ed.BaseFeePerGas)
	}

	var blobGasUsed *uint64
	if ed.BlobGasUsed != nil {
		blobGasUsed = new(uint64)
		*blobGasUsed = *ed.BlobGasUsed
	}

	var excessBlobGas *uint64
	if ed.ExcessBlobGas != nil {
		excessBlobGas = new(uint64)
		*excessBlobGas = *ed.ExcessBlobGas
	}

	return RethPayloadV3{
		ParentHash:       ed.ParentHash,
		FeeRecipient:     ed.FeeRecipient,
		StateRoot:        ed.StateRoot,
		ReceiptsRoot:     ed.ReceiptsRoot,
		LogsBloom:        ed.LogsBloom,
		Random:           ed.Random,
		Number:           ed.Number,
		GasLimit:         ed.GasLimit,
		GasUsed:          ed.GasUsed,
		Timestamp:        ed.Timestamp,
		ExtraData:        ed.ExtraData,
		BaseFeePerGas:    baseFee,
		BlockHash:        ed.BlockHash,
		Transactions:     copyTransactions(ed.Transactions),
		Withdrawals:      copyWithdrawals(ed.Withdrawals),
		BlobGasUsed:      blobGasUsed,
		ExcessBlobGas:    excessBlobGas,
		ExecutionWitness: ed.ExecutionWitness, // Assuming it's safe to assign directly
	}
}

// copyTransactions creates a deep copy of the transactions slice.
func copyTransactions(transactions [][]byte) [][]byte {
	if transactions == nil {
		return [][]byte{}
	}
	copied := make([][]byte, len(transactions))
	for i, tx := range transactions {
		if tx != nil {
			copied[i] = append([]byte(nil), tx...)
		}
	}
	return copied
}

// copyWithdrawals creates a deep copy of the withdrawals slice.
func copyWithdrawals(withdrawals []*types.Withdrawal) []*types.Withdrawal {
	if withdrawals == nil {
		return []*types.Withdrawal{}
	}
	copied := make([]*types.Withdrawal, len(withdrawals))
	for i, withdrawal := range withdrawals {
		if withdrawal != nil {
			copied[i] = withdrawal // Assuming Withdrawal is immutable or doesn't require deep copy
		}
	}
	return copied
}

func (c engineClient) NewPayloadV3(ctx context.Context, params engine.ExecutableData, versionedHashes []common.Hash,
	beaconRoot *common.Hash,
) (engine.PayloadStatusV1, error) {
	log.Debug(ctx, "Entering NewPayloadV3. Converting standard paylod to Seismic Reth payload", nil)
	rethPayload := ConvertExecutableDataToRethPayloadV3(params)
	const endpoint = "new_payload_v3"
	defer latency(c.chain, endpoint)()

	// isStatusOk returns true if the response status is valid.
	isStatusOk := func(status engine.PayloadStatusV1) bool {
		return map[string]bool{
			engine.VALID:    true,
			engine.INVALID:  true,
			engine.SYNCING:  true,
			engine.ACCEPTED: true,
		}[status.Status]
	}

	var resp engine.PayloadStatusV1
	err := c.cl.Client().CallContext(ctx, &resp, newPayloadV3, rethPayload, versionedHashes, beaconRoot)
	if isStatusOk(resp) {
		// Swallow errors when geth returns errors along with proper responses (but at least log it).
		if err != nil {
			log.Warn(ctx, "Ignoring new_payload_v3 error with proper response", err, "status", resp.Status)
		}

		return resp, nil
	} else if err != nil {
		incError(c.chain, endpoint)
		return engine.PayloadStatusV1{}, errors.Wrap(err, "rpc new payload")
	} /* else err==nil && status!=ok */

	incError(c.chain, endpoint)

	return engine.PayloadStatusV1{}, errors.New("nil error and unknown status", "status", resp.Status)
}

func (c engineClient) ForkchoiceUpdatedV3(ctx context.Context, update engine.ForkchoiceStateV1,
	payloadAttributes *engine.PayloadAttributes,
) (engine.ForkChoiceResponse, error) {
	const endpoint = "forkchoice_updated_v3"
	defer latency(c.chain, endpoint)()

	// isStatusOk returns true if the response status is valid.
	isStatusOk := func(resp engine.ForkChoiceResponse) bool {
		return map[string]bool{
			engine.VALID:    true,
			engine.INVALID:  true,
			engine.SYNCING:  true,
			engine.ACCEPTED: false, // Unexpected in ForkchoiceUpdated
		}[resp.PayloadStatus.Status]
	}

	var resp engine.ForkChoiceResponse
	err := c.cl.Client().CallContext(ctx, &resp, forkchoiceUpdatedV3, update, payloadAttributes)
	if isStatusOk(resp) {
		// Swallow errors when geth returns errors along with proper responses (but at least log it).
		if err != nil {
			log.Warn(ctx, "Ignoring forkchoice_updated_v3 error with proper response", err, "status", resp.PayloadStatus.Status)
		}

		return resp, nil
	} else if err != nil {
		incError(c.chain, endpoint)
		return engine.ForkChoiceResponse{}, errors.Wrap(err, "rpc forkchoice updated v3")
	} /* else err==nil && status!=ok */

	incError(c.chain, endpoint)

	return engine.ForkChoiceResponse{}, errors.New("nil error and unknown status", "status", resp.PayloadStatus.Status)
}

func (c engineClient) GetPayloadV3(ctx context.Context, payloadID engine.PayloadID) (
	*engine.ExecutionPayloadEnvelope, error,
) {
	const endpoint = "get_payload_v3"
	defer latency(c.chain, endpoint)()

	var resp engine.ExecutionPayloadEnvelope
	err := c.cl.Client().CallContext(ctx, &resp, getPayloadV3, payloadID)
	if err != nil {
		incError(c.chain, endpoint)
		return nil, errors.Wrap(err, "rpc get payload v3")
	}

	return &resp, nil
}
