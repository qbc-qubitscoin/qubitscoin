package vm

import (
	"context"
	"errors"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ErrOutOfGas is returned when a contract exceeds its gas allowance.
var ErrOutOfGas = errors.New("out of gas")

// ExecCtxKey is the context key for ExecutionContext.
type ExecCtxKey struct{}

// ExecutionContext holds the runtime environment for a single contract call.
type ExecutionContext struct {
	ContractAddr [crypto.AddressSize]byte
	CallerAddr   [crypto.AddressSize]byte
	Value        uint64 // QBC (in qubits) sent with the call
	BlockHeight  uint64
	BlockTime    int64
	GasLimit     uint64
	GasUsed      uint64
	ReadOnly     bool
}

// UseGas subtracts amount from the remaining gas.
// Returns ErrOutOfGas if the budget is exceeded.
func (ec *ExecutionContext) UseGas(amount uint64) error {
	if ec.GasUsed+amount > ec.GasLimit {
		return ErrOutOfGas
	}
	ec.GasUsed += amount
	return nil
}

// WithExecCtx returns a new context carrying the given ExecutionContext.
func WithExecCtx(ctx context.Context, ec *ExecutionContext) context.Context {
	return context.WithValue(ctx, ExecCtxKey{}, ec)
}

// GetExecCtx retrieves the ExecutionContext from a context or nil.
func GetExecCtx(ctx context.Context) *ExecutionContext {
	ec, _ := ctx.Value(ExecCtxKey{}).(*ExecutionContext)
	return ec
}
