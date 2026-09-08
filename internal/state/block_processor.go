package state

import (
	"fmt"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/vm"
)

// BlockResult summarises the outcome of applying a full block to state.
type BlockResult struct {
	GasUsed      uint64
	FeeCollected uint64 // total fees paid by all senders
	BurnedFees   uint64 // base-fee portion burned (removed from supply)
	ValidatorTip uint64 // priority-tip portion credited to validator
	BlockReward  uint64 // coinbase subsidy
	TxResults    []*TxResult
}

// TotalValidatorIncome returns the sum credited to the validator for this block.
// = ValidatorTip + BlockReward (BurnedFees are NOT included — they are destroyed)
func (r *BlockResult) TotalValidatorIncome() uint64 {
	return r.ValidatorTip + r.BlockReward
}

// ApplyBlock applies every transaction in blk to st, splits each fee into
// burned (base-fee) and validator tip, then credits tip + block subsidy
// to validatorAddr.
//
// BurnedFees are implicitly removed from supply: they are deducted from
// senders but never credited to anyone — effective deflation.
func ApplyBlock(
	st *DB,
	blk *core.Block,
	validatorAddr [crypto.AddressSize]byte,
	execVM *vm.VM,
) (*BlockResult, error) {

	baseFee := blk.Header.BaseFee
	remainingGas := core.BlockGasLimit
	result := &BlockResult{}

	for _, tx := range blk.Txs {
		txRes, err := ApplyTransaction(st, tx, remainingGas, execVM, baseFee)
		if err != nil {
			return nil, fmt.Errorf("tx %s failed: %w", crypto.ToHex(tx.Hash), err)
		}
		remainingGas -= txRes.GasUsed
		result.GasUsed += txRes.GasUsed
		result.FeeCollected += txRes.FeeCollected
		result.BurnedFees += txRes.BurnedFee
		result.ValidatorTip += txRes.ValidatorTip
		result.TxResults = append(result.TxResults, txRes)
	}

	// ── Credit validator: tip + block subsidy ────────────────────────────────
	// BurnedFees are NOT credited — they simply vanish, reducing supply.
	reward := core.BlockReward(blk.Header.Height)
	result.BlockReward = reward

	income := result.ValidatorTip + reward
	if income > 0 {
		validator := st.GetAccount(validatorAddr)
		validator.Balance += income
		st.SetAccount(validatorAddr, validator)
	}

	return result, nil
}
