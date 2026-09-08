package vm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// Gas costs for host functions.
const (
	gasCostGet         uint64 = 200
	gasCostSet         uint64 = 500
	gasCostLog         uint64 = 100
	gasCostCaller      uint64 = 50
	gasCostBlockHeight uint64 = 10
	gasCostValue       uint64 = 10
)

// NOTE: wazero GoModuleFunc signature is:
//   func(ctx context.Context, mod api.Module, stack []uint64)
// Params are in stack[0...n-1]; return values are written back into stack[0...m-1].

// hostGet implements env.qbc_get(slot i32) -> i64
func hostGet(storage *ContractStorage) api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		ec := GetExecCtx(ctx)
		if ec == nil {
			stack[0] = 0
			return
		}
		if err := ec.UseGas(gasCostGet); err != nil {
			panic(ErrOutOfGas)
		}
		slot := uint32(stack[0])
		stack[0] = storage.Get(ec.ContractAddr, slot)
	}
}

// hostSet implements env.qbc_set(slot i32, value i64)
func hostSet(storage *ContractStorage) api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		ec := GetExecCtx(ctx)
		if ec == nil {
			return
		}
		if ec.ReadOnly {
			panic("write to storage in a read-only context")
		}
		if err := ec.UseGas(gasCostSet); err != nil {
			panic(ErrOutOfGas)
		}
		slot := uint32(stack[0])
		value := stack[1]
		storage.Set(ec.ContractAddr, slot, value)
	}
}

// hostLog implements env.qbc_log(ptr i32, len i32)
func hostLog() api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		ec := GetExecCtx(ctx)
		if ec != nil {
			if err := ec.UseGas(gasCostLog); err != nil {
				panic(ErrOutOfGas)
			}
		}
		ptr, length := uint32(stack[0]), uint32(stack[1])
		mem := mod.Memory()
		if mem == nil {
			return
		}
		b, ok := mem.Read(ptr, length)
		if !ok {
			return
		}
		fmt.Printf("[contract log] %s\n", b)
	}
}

// hostCaller implements env.qbc_caller() -> i64
func hostCaller() api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		ec := GetExecCtx(ctx)
		if ec == nil {
			stack[0] = 0
			return
		}
		if err := ec.UseGas(gasCostCaller); err != nil {
			panic(ErrOutOfGas)
		}
		var v uint64
		for i := 0; i < 8; i++ {
			v = (v << 8) | uint64(ec.CallerAddr[i])
		}
		stack[0] = v
	}
}

// hostBlockHeight implements env.qbc_block_height() -> i64
func hostBlockHeight() api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		ec := GetExecCtx(ctx)
		if ec == nil {
			stack[0] = 0
			return
		}
		if err := ec.UseGas(gasCostBlockHeight); err != nil {
			panic(ErrOutOfGas)
		}
		stack[0] = ec.BlockHeight
	}
}

// hostValue implements env.qbc_value() -> i64
func hostValue() api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		ec := GetExecCtx(ctx)
		if ec == nil {
			stack[0] = 0
			return
		}
		if err := ec.UseGas(gasCostValue); err != nil {
			panic(ErrOutOfGas)
		}
		stack[0] = ec.Value
	}
}
