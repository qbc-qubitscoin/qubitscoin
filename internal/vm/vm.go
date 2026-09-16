package vm

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// VM manages WASM contract compilation and execution via wazero.
type VM struct {
	rt      wazero.Runtime
	modules map[string]wazero.CompiledModule // codeHash hex -> compiled module
	storage *ContractStorage
	seq     atomic.Uint64 // unique suffix for module instance names
}

var newRuntime = wazero.NewRuntime

// NewVM creates a new VM and registers the QBC host module.
func NewVM(ctx context.Context) (*VM, error) {
	rt := newRuntime(ctx)
	v := &VM{
		rt:      rt,
		modules: make(map[string]wazero.CompiledModule),
		storage: NewContractStorage(),
	}
	if err := v.registerHostModule(ctx); err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	return v, nil
}

// registerHostModule installs all QBC host functions under the "env" module.
func (v *VM) registerHostModule(ctx context.Context) error {
	_, err := v.rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithGoModuleFunction(hostGet(v.storage), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("qbc_get").
		NewFunctionBuilder().
		WithGoModuleFunction(hostSet(v.storage), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{}).
		Export("qbc_set").
		NewFunctionBuilder().
		WithGoModuleFunction(hostLog(), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("qbc_log").
		NewFunctionBuilder().
		WithGoModuleFunction(hostCaller(), []api.ValueType{}, []api.ValueType{api.ValueTypeI64}).
		Export("qbc_caller").
		NewFunctionBuilder().
		WithGoModuleFunction(hostBlockHeight(), []api.ValueType{}, []api.ValueType{api.ValueTypeI64}).
		Export("qbc_block_height").
		NewFunctionBuilder().
		WithGoModuleFunction(hostValue(), []api.ValueType{}, []api.ValueType{api.ValueTypeI64}).
		Export("qbc_value").
		Instantiate(ctx)
	return err
}

// Deploy compiles and caches a contract's WASM bytecode.
// contractAddr is the on-chain address; code is the raw WASM bytes.
func (v *VM) Deploy(ctx context.Context, contractAddr [crypto.AddressSize]byte, code []byte) error {
	codeHash := crypto.Hash256(code)
	key := crypto.ToHex(codeHash)
	if _, exists := v.modules[key]; exists {
		return nil // already compiled
	}
	compiled, err := v.rt.CompileModule(ctx, code)
	if err != nil {
		return fmt.Errorf("compile WASM: %w", err)
	}
	v.modules[key] = compiled
	return nil
}

// Call invokes a named export function of the contract identified by codeHash.
// The ExecutionContext must be set on ctx via WithExecCtx.
// Returns the function's return values as a []uint64 slice.
func (v *VM) Call(
	ctx context.Context,
	ec *ExecutionContext,
	codeHash [crypto.HashSize]byte,
	funcName string,
	params ...uint64,
) ([]uint64, error) {
	key := crypto.ToHex(codeHash)
	compiled, ok := v.modules[key]
	if !ok {
		return nil, fmt.Errorf("contract isn't deployed: %s", key)
	}

	// Instantiate a fresh module instance for each call (isolation).
	instanceName := fmt.Sprintf("contract_%s_%d", key[:8], v.seq.Add(1))
	cfg := wazero.NewModuleConfig().WithName(instanceName)
	mod, err := v.rt.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}
	defer func(mod api.Module, ctx context.Context) {
		_ = mod.Close(ctx)
	}(mod, ctx)

	fn := mod.ExportedFunction(funcName)
	if fn == nil {
		return nil, fmt.Errorf("function %q not exported by contract", funcName)
	}

	result, err := fn.Call(ctx, params...)
	if err != nil {
		return nil, fmt.Errorf("call %q: %w", funcName, err)
	}
	return result, nil
}

// Storage returns the underlying contract storage (for inspection/testing).
func (v *VM) Storage() *ContractStorage { return v.storage }

// Close releases all wazero resources.
func (v *VM) Close(ctx context.Context) {
	_ = v.rt.Close(ctx)
}
