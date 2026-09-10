package vm

import (
	"context"
	"testing"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func TestHostFunctions(t *testing.T) {
	storage := NewContractStorage()
	stack := make([]uint64, 2)
	ctxBg := context.Background()
	ecValid := &ExecutionContext{GasLimit: 1000000}
	ctxValid := WithExecCtx(ctxBg, ecValid)

	// hostGet
	fGet := hostGet(storage)
	fGet(ctxBg, nil, stack) // ec == nil

	func() {
		defer func() { recover() }()
		fGet(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 0}), nil, stack) // OutOfGas
	}()

	fGet(ctxValid, nil, stack) // Success

	// hostSet
	fSet := hostSet(storage)
	fSet(ctxBg, nil, stack) // ec == nil

	func() {
		defer func() { recover() }()
		fSet(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 1000, ReadOnly: true}), nil, stack) // ReadOnly
	}()

	func() {
		defer func() { recover() }()
		fSet(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 0}), nil, stack) // OutOfGas
	}()

	fSet(ctxValid, nil, stack) // Success

	// hostCaller
	fCaller := hostCaller()
	fCaller(ctxBg, nil, stack) // ec == nil
	func() {
		defer func() { recover() }()
		fCaller(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 0}), nil, stack) // OutOfGas
	}()
	fCaller(ctxValid, nil, stack) // Success

	// hostBlockHeight
	fBlockHeight := hostBlockHeight()
	fBlockHeight(ctxBg, nil, stack) // ec == nil
	func() {
		defer func() { recover() }()
		fBlockHeight(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 0}), nil, stack) // OutOfGas
	}()
	fBlockHeight(ctxValid, nil, stack) // Success

	// hostValue
	fValue := hostValue()
	fValue(ctxBg, nil, stack) // ec == nil
	func() {
		defer func() { recover() }()
		fValue(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 0}), nil, stack) // OutOfGas
	}()
	fValue(ctxValid, nil, stack) // Success

	// hostLog needs a module for memory
	r := wazero.NewRuntime(ctxBg)
	defer r.Close(ctxBg)
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x05, 0x03, 0x01, 0x00, 0x01}
	mod, err := r.Instantiate(ctxBg, wasm)
	if err != nil {
		t.Fatalf("failed to instantiate WASM: %v", err)
	}

	fLog := hostLog()
	// ec == nil, mem != nil (panics if mem access invalid but stack[0] is 0, stack[1] is 0)
	stack[0] = 0
	stack[1] = 0
	fLog(ctxBg, mod, stack) 

	// ec != nil, OutOfGas
	func() {
		defer func() { recover() }()
		fLog(WithExecCtx(ctxBg, &ExecutionContext{GasLimit: 0}), mod, stack)
	}()
	
	// valid mem access
	stack[0] = 0
	stack[1] = 4 // memory has some bytes
	fLog(ctxValid, mod, stack)

	// invalid mem access
	stack[0] = 0xFFFFFFFF
	stack[1] = 4
	fLog(ctxValid, mod, stack)
}

type mockModule struct {
	api.Module
}

func (m mockModule) Memory() api.Memory {
	return nil
}

func TestHostLog_NoMemory(t *testing.T) {
	ctxBg := context.Background()
	fLog := hostLog()
	stack := make([]uint64, 2)
	
	mod := mockModule{}
	fLog(ctxBg, mod, stack)
}
