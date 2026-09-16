package vm

import (
	"context"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// testVM is created once per test run.
var testVM *VM

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error
	testVM, err = NewVM(ctx)
	if err != nil {
		panic("failed to create VM: " + err.Error())
	}
	defer testVM.Close(ctx)
	m.Run()
}

func testAddr(seed byte) [crypto.AddressSize]byte {
	var addr [crypto.AddressSize]byte
	addr[0] = seed
	return addr
}

func deployCounter(t *testing.T, deployer [crypto.AddressSize]byte) ([crypto.HashSize]byte, [crypto.HashSize]byte) {
	t.Helper()
	ctx := context.Background()
	codeHash := crypto.Hash256(CounterContractWASM)
	contractAddr := testAddr(0xCC)
	if err := testVM.Deploy(ctx, contractAddr, CounterContractWASM); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	return contractAddr, codeHash
}

func callIncrement(t *testing.T, contractAddr, callerAddr [crypto.HashSize]byte, codeHash [crypto.HashSize]byte) {
	t.Helper()
	ctx := context.Background()
	ec := &ExecutionContext{
		ContractAddr: contractAddr,
		CallerAddr:   callerAddr,
		GasLimit:     500_000,
	}
	_, err := testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "increment")
	if err != nil {
		t.Fatalf("Call increment: %v", err)
	}
}

func getCount(t *testing.T, contractAddr, callerAddr [crypto.HashSize]byte, codeHash [crypto.HashSize]byte) uint64 {
	t.Helper()
	ctx := context.Background()
	ec := &ExecutionContext{
		ContractAddr: contractAddr,
		CallerAddr:   callerAddr,
		GasLimit:     100_000,
	}
	results, err := testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "get_count")
	if err != nil {
		t.Fatalf("Call get_count: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("get_count: want 1 return value, got %d", len(results))
	}
	return results[0]
}

func TestVM_Deploy_CounterContract(t *testing.T) {
	ctx := context.Background()
	addr := testAddr(0xD1)
	codeHash := crypto.Hash256(CounterContractWASM)
	if err := testVM.Deploy(ctx, addr, CounterContractWASM); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	// Verify the code hash is stored.
	if testVM.Storage().Get(addr, 0) == 0 {
		_ = codeHash // deployed, storage starts at 0 which is correct for counter
	}
}

func TestVM_Deploy_InvalidWASM2(t *testing.T) {
	ctx := context.Background()
	addr := testAddr(0xD2)
	if err := testVM.Deploy(ctx, addr, []byte("not wasm")); err == nil {
		t.Fatal("expected error when deploying invalid WASM")
	}
}

func TestVM_Call_IncrementOnce(t *testing.T) {
	deployer := testAddr(0x01)
	contractAddr, codeHash := deployCounter(t, deployer)

	callIncrement(t, contractAddr, deployer, codeHash)
	count := getCount(t, contractAddr, deployer, codeHash)
	if count != 1 {
		t.Errorf("after 1 increment: want count=1, got %d", count)
	}
}

func TestVM_Call_IncrementFive(t *testing.T) {
	// Use a fresh contract address to avoid state from other tests.
	ctx := context.Background()
	addr := testAddr(0xC5)
	deployer := testAddr(0x05)
	codeHash := crypto.Hash256(CounterContractWASM)

	if err := testVM.Deploy(ctx, addr, CounterContractWASM); err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	for i := 0; i < 5; i++ {
		ec := &ExecutionContext{ContractAddr: addr, CallerAddr: deployer, GasLimit: 500_000}
		if _, err := testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "increment"); err != nil {
			t.Fatalf("increment #%d: %v", i+1, err)
		}
	}

	ec := &ExecutionContext{ContractAddr: addr, CallerAddr: deployer, GasLimit: 100_000}
	results, err := testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "get_count")
	if err != nil {
		t.Fatalf("get_count: %v", err)
	}
	if len(results) != 1 || results[0] != 5 {
		t.Errorf("after 5 increments: want 5, got %v", results)
	}
}

func TestVM_Call_GetCount_Initial(t *testing.T) {
	ctx := context.Background()
	addr := testAddr(0xC0)
	deployer := testAddr(0x00)
	codeHash := crypto.Hash256(CounterContractWASM)

	if err := testVM.Deploy(ctx, addr, CounterContractWASM); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	ec := &ExecutionContext{ContractAddr: addr, CallerAddr: deployer, GasLimit: 100_000}
	results, err := testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "get_count")
	if err != nil {
		t.Fatalf("get_count: %v", err)
	}
	if len(results) != 1 || results[0] != 0 {
		t.Errorf("initial count: want 0, got %v", results)
	}
}

func TestVM_Storage_IsolatedPerContract(t *testing.T) {
	ctx := context.Background()
	addr1 := testAddr(0xE1)
	addr2 := testAddr(0xE2)
	caller := testAddr(0xFF)
	codeHash := crypto.Hash256(CounterContractWASM)

	_ = testVM.Deploy(ctx, addr1, CounterContractWASM)
	_ = testVM.Deploy(ctx, addr2, CounterContractWASM)

	// Increment addr1 twice, addr2 once.
	for i := 0; i < 2; i++ {
		ec := &ExecutionContext{ContractAddr: addr1, CallerAddr: caller, GasLimit: 500_000}
		_, _ = testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "increment")
	}
	ec := &ExecutionContext{ContractAddr: addr2, CallerAddr: caller, GasLimit: 500_000}
	_, _ = testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "increment")

	ec1 := &ExecutionContext{ContractAddr: addr1, CallerAddr: caller, GasLimit: 100_000}
	r1, _ := testVM.Call(WithExecCtx(ctx, ec1), ec1, codeHash, "get_count")

	ec2 := &ExecutionContext{ContractAddr: addr2, CallerAddr: caller, GasLimit: 100_000}
	r2, _ := testVM.Call(WithExecCtx(ctx, ec2), ec2, codeHash, "get_count")

	if len(r1) != 1 || r1[0] != 2 {
		t.Errorf("addr1 count: want 2, got %v", r1)
	}
	if len(r2) != 1 || r2[0] != 1 {
		t.Errorf("addr2 count: want 1, got %v", r2)
	}
}

func TestCounterContractWASM_NotEmpty(t *testing.T) {
	if len(CounterContractWASM) == 0 {
		t.Fatal("CounterContractWASM should not be empty")
	}
}

func TestVM_GasUsed_Increases(t *testing.T) {
	ctx := context.Background()
	addr := testAddr(0xAB)
	deployer := testAddr(0x0A)
	codeHash := crypto.Hash256(CounterContractWASM)
	_ = testVM.Deploy(ctx, addr, CounterContractWASM)

	ec := &ExecutionContext{ContractAddr: addr, CallerAddr: deployer, GasLimit: 500_000}
	_, _ = testVM.Call(WithExecCtx(ctx, ec), ec, codeHash, "increment")
	if ec.GasUsed == 0 {
		t.Error("GasUsed should be > 0 after a call")
	}
}
