package vm

import (
	"context"
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ── ContractStorage ───────────────────────────────────────────────────────────

func TestContractStorage_GetDefault(t *testing.T) {
	s := NewContractStorage()
	var addr [crypto.AddressSize]byte
	if got := s.Get(addr, 0); got != 0 {
		t.Errorf("default Get: want 0, got %d", got)
	}
}

func TestContractStorage_SetGet(t *testing.T) {
	s := NewContractStorage()
	var addr [crypto.AddressSize]byte
	addr[0] = 1

	s.Set(addr, 7, 999)
	if got := s.Get(addr, 7); got != 999 {
		t.Errorf("Get after Set: want 999, got %d", got)
	}
}

func TestContractStorage_MultipleAddresses(t *testing.T) {
	s := NewContractStorage()
	var a1, a2 [crypto.AddressSize]byte
	a1[0] = 1
	a2[0] = 2

	s.Set(a1, 0, 100)
	s.Set(a2, 0, 200)

	if got := s.Get(a1, 0); got != 100 {
		t.Errorf("a1: want 100, got %d", got)
	}
	if got := s.Get(a2, 0); got != 200 {
		t.Errorf("a2: want 200, got %d", got)
	}
}

func TestContractStorage_OverwriteSlot(t *testing.T) {
	s := NewContractStorage()
	var addr [crypto.AddressSize]byte
	s.Set(addr, 1, 42)
	s.Set(addr, 1, 99)
	if got := s.Get(addr, 1); got != 99 {
		t.Errorf("overwrite: want 99, got %d", got)
	}
}

// ── ExecutionContext ──────────────────────────────────────────────────────────

func TestExecutionContext_UseGas_OK(t *testing.T) {
	ec := &ExecutionContext{GasLimit: 1000}
	if err := ec.UseGas(500); err != nil {
		t.Errorf("UseGas(500/1000): unexpected error: %v", err)
	}
	if ec.GasUsed != 500 {
		t.Errorf("GasUsed: want 500, got %d", ec.GasUsed)
	}
}

func TestExecutionContext_UseGas_OutOfGas(t *testing.T) {
	ec := &ExecutionContext{GasLimit: 100}
	if err := ec.UseGas(101); err != ErrOutOfGas {
		t.Errorf("UseGas over limit: want ErrOutOfGas, got %v", err)
	}
}

func TestExecutionContext_UseGas_Exact(t *testing.T) {
	ec := &ExecutionContext{GasLimit: 100}
	if err := ec.UseGas(100); err != nil {
		t.Errorf("UseGas at exact limit: unexpected error: %v", err)
	}
}

// ── WithExecCtx / GetExecCtx ──────────────────────────────────────────────────

func TestWithAndGetExecCtx(t *testing.T) {
	ec := &ExecutionContext{BlockHeight: 42, GasLimit: 1000}
	ctx := WithExecCtx(context.Background(), ec)
	got := GetExecCtx(ctx)
	if got == nil {
		t.Fatal("GetExecCtx returned nil")
	}
	if got.BlockHeight != 42 {
		t.Errorf("BlockHeight: want 42, got %d", got.BlockHeight)
	}
}

func TestGetExecCtx_NilWhenMissing(t *testing.T) {
	got := GetExecCtx(context.Background())
	if got != nil {
		t.Errorf("GetExecCtx with no value: want nil, got %v", got)
	}
}

// ── VM lifecycle ──────────────────────────────────────────────────────────────

func TestNewVM_CreatesSuccessfully(t *testing.T) {
	ctx := context.Background()
	v, err := NewVM(ctx)
	if err != nil {
		t.Fatalf("NewVM: %v", err)
	}
	defer v.Close(ctx)

	if v.Storage() == nil {
		t.Error("Storage() should not be nil after NewVM")
	}
}

func TestVM_Call_UndeployedContract(t *testing.T) {
	ctx := context.Background()
	v, err := NewVM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close(ctx)

	var fakeHash [crypto.HashSize]byte
	ec := &ExecutionContext{GasLimit: 100_000}
	_, err = v.Call(WithExecCtx(ctx, ec), ec, fakeHash, "foo")
	if err == nil {
		t.Error("Call on undeployed contract should fail")
	}
}

func TestVM_Deploy_InvalidWASM(t *testing.T) {
	ctx := context.Background()
	v, err := NewVM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close(ctx)

	var addr [crypto.AddressSize]byte
	err = v.Deploy(ctx, addr, []byte("not-wasm"))
	if err == nil {
		t.Error("Deploy with invalid WASM should fail")
	}
}

func TestVM_Deploy_AlreadyDeployed(t *testing.T) {
	ctx := context.Background()
	v, err := NewVM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close(ctx)

	// Use minimal valid WASM: empty module "(module)"
	// Magic bytes + version 1 + empty sections
	minimalWASM := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic: \0asm
		0x01, 0x00, 0x00, 0x00, // version 1
	}
	var addr [crypto.AddressSize]byte

	if err := v.Deploy(ctx, addr, minimalWASM); err != nil {
		t.Skipf("minimal wasm deploy failed (may need exported funcs): %v", err)
	}
	// Second deploy of same bytecode should be a no-op (no error)
	if err := v.Deploy(ctx, addr, minimalWASM); err != nil {
		t.Errorf("re-deploy same contract should return nil, got: %v", err)
	}
}
