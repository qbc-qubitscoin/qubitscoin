package rpc

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/qbc-qubitscoin/qubitscoin/internal/consensus"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/mempool"
	"github.com/qbc-qubitscoin/qubitscoin/internal/state"
)

// API holds references to node subsystems and handles JSON-RPC method calls.
type API struct {
	engine  *consensus.Engine
	st      *state.DB
	pool    *mempool.Mempool
	peersFn func() int // returns current peer count
	version string
}

// NewAPI constructs an API handler.
// peersFn may be nil (peer count will be reported as 0).
func NewAPI(
	engine *consensus.Engine,
	st *state.DB,
	pool *mempool.Mempool,
	peersFn func() int,
	ver string,
) *API {
	if peersFn == nil {
		peersFn = func() int { return 0 }
	}
	return &API{engine: engine, st: st, pool: pool, peersFn: peersFn, version: ver}
}

// Dispatch routes a parsed request to the appropriate handler.
func (a *API) Dispatch(req *Request) *Response {
	switch req.Method {

	// ── Chain ──────────────────────────────────────────────────────────────
	case "qbc_chainInfo":
		return a.chainInfo(req)

	// ── Blocks ─────────────────────────────────────────────────────────────
	case "qbc_blockByHeight":
		return a.blockByHeight(req)
	case "qbc_blockByHash":
		return a.blockByHash(req)
	case "qbc_blockHeight":
		return okResponse(req.ID, a.engine.Height())

	// ── Accounts ───────────────────────────────────────────────────────────
	case "qbc_getBalance":
		return a.getBalance(req)
	case "qbc_getTransactionCount":
		return a.getTransactionCount(req)

	// ── Transactions ───────────────────────────────────────────────────────
	case "qbc_sendRawTransaction":
		return a.sendRawTransaction(req)

	// ── Fees ───────────────────────────────────────────────────────────────
	case "qbc_feeEstimate":
		return a.feeEstimate(req)
	case "qbc_gasPrice":
		return a.gasPrice(req)

	default:
		return errResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("method %q not found", req.Method))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler implementations
// ─────────────────────────────────────────────────────────────────────────────

func (a *API) chainInfo(req *Request) *Response {
	height := a.engine.Height()
	tipHash := ""
	if blk := a.engine.BlockByHeight(height - 1); blk != nil {
		tipHash = crypto.ToHex(blk.Hash)
	}
	return okResponse(req.ID, &ChainInfo{
		ChainID:    core.ChainID,
		NetworkID:  core.ChainID,
		Height:     height,
		TipHash:    tipHash,
		PeerCount:  a.peersFn(),
		MempoolLen: a.pool.Len(),
		Version:    a.version,
	})
}

func (a *API) blockByHeight(req *Request) *Response {
	var params []uint64
	if err := json.Unmarshal(req.Params, &params); err != nil || len(params) == 0 {
		return errResponse(req.ID, CodeInvalidParams, "params: [height uint64]")
	}
	blk := a.engine.BlockByHeight(params[0])
	if blk == nil {
		return errResponse(req.ID, CodeInternalError, "block not found")
	}
	return okResponse(req.ID, blockToInfo(blk))
}

func (a *API) blockByHash(req *Request) *Response {
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil || len(params) == 0 {
		return errResponse(req.ID, CodeInvalidParams, "params: [hash hex-string]")
	}
	hash, err := crypto.HexToHash(params[0])
	if err != nil {
		return errResponse(req.ID, CodeInvalidParams, "invalid hash: "+err.Error())
	}
	// Linear scan from height 0 up to current tip.
	height := a.engine.Height()
	for h := uint64(0); h < height; h++ {
		blk := a.engine.BlockByHeight(h)
		if blk.Hash == hash {
			return okResponse(req.ID, blockToInfo(blk))
		}
	}
	return errResponse(req.ID, CodeInternalError, "block not found")
}

func (a *API) getBalance(req *Request) *Response {
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil || len(params) == 0 {
		return errResponse(req.ID, CodeInvalidParams, "params: [address hex-string]")
	}
	addr, err := crypto.HexToAddress(params[0])
	if err != nil {
		return errResponse(req.ID, CodeInvalidParams, "invalid address: "+err.Error())
	}
	return okResponse(req.ID, map[string]uint64{
		"balance_qubits": a.st.GetBalance(addr),
	})
}

func (a *API) getTransactionCount(req *Request) *Response {
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil || len(params) == 0 {
		return errResponse(req.ID, CodeInvalidParams, "params: [address hex-string]")
	}
	addr, err := crypto.HexToAddress(params[0])
	if err != nil {
		return errResponse(req.ID, CodeInvalidParams, "invalid address: "+err.Error())
	}
	return okResponse(req.ID, map[string]uint64{
		"nonce": a.st.GetNonce(addr),
	})
}

// sendRawTransaction accepts a hex-encoded gob-serialised Transaction.
func (a *API) sendRawTransaction(req *Request) *Response {
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil || len(params) == 0 {
		return errResponse(req.ID, CodeInvalidParams, "params: [raw_tx hex-string]")
	}
	raw, err := hex.DecodeString(params[0])
	if err != nil {
		return errResponse(req.ID, CodeInvalidParams, "invalid hex: "+err.Error())
	}
	tx, err := gobDecodeTx(raw)
	if err != nil {
		return errResponse(req.ID, CodeInvalidParams, "decode tx: "+err.Error())
	}
	if err := a.pool.Add(tx); err != nil {
		return errResponse(req.ID, CodeInternalError, "mempool: "+err.Error())
	}
	return okResponse(req.ID, map[string]string{
		"tx_hash": crypto.ToHex(tx.Hash),
	})
}

func (a *API) feeEstimate(req *Request) *Response {
	height := a.engine.Height()
	blk := a.engine.BlockByHeight(height - 1)
	baseFee := blk.Header.BaseFee

	_, ulTip := core.FeeEstimate(baseFee, core.FeeTierUltraLow)
	_, stdTip := core.FeeEstimate(baseFee, core.FeeTierStandard)
	_, fastTip := core.FeeEstimate(baseFee, core.FeeTierFast)

	return okResponse(req.ID, &FeeEstimateResult{
		BaseFee:          baseFee,
		UltraLowTip:      ulTip,
		StandardTip:      stdTip,
		FastTip:          fastTip,
		TransferUltraLow: core.TransferCostQubits(baseFee, ulTip),
		TransferStandard: core.TransferCostQubits(baseFee, stdTip),
		TransferFast:     core.TransferCostQubits(baseFee, fastTip),
	})
}

func (a *API) gasPrice(req *Request) *Response {
	height := a.engine.Height()
	blk := a.engine.BlockByHeight(height - 1)
	return okResponse(req.ID, map[string]uint64{"base_fee": blk.Header.BaseFee})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func blockToInfo(blk *core.Block) *BlockInfo {
	return &BlockInfo{
		Height:        blk.Header.Height,
		Hash:          crypto.ToHex(blk.Hash),
		PrevHash:      crypto.ToHex(blk.Header.PrevHash),
		StateRoot:     crypto.ToHex(blk.Header.StateRoot),
		MerkleRoot:    crypto.ToHex(blk.Header.MerkleRoot),
		Timestamp:     blk.Header.Timestamp,
		ValidatorAddr: crypto.AddressToHex(blk.Header.ValidatorAddr),
		GasUsed:       blk.Header.GasUsed,
		GasLimit:      blk.Header.GasLimit,
		BaseFee:       blk.Header.BaseFee,
		BurnedFees:    blk.Header.BurnedFees,
		TxCount:       len(blk.Txs),
	}
}

// gobDecodeTx decodes a gob-encoded Transaction from raw bytes.
func gobDecodeTx(raw []byte) (*core.Transaction, error) {
	var tx core.Transaction
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// GobEncodeTx encodes a transaction to raw bytes suitable for sendRawTransaction.
func GobEncodeTx(tx *core.Transaction) ([]byte, error) {
	var buf bytes.Buffer
	_ = gob.NewEncoder(&buf).Encode(tx)
	return buf.Bytes(), nil
}
