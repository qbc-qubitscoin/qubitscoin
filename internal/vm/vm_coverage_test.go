package vm

import (
	"context"
	"fmt"
	"testing"

	"github.com/tetratelabs/wazero"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestAppendULEB128_MultiByte(t *testing.T) {
	buf := appendULEB128(nil, 300)
	if len(buf) < 2 {
		t.Fatalf("expected at least 2 bytes for 300, got %d", len(buf))
	}
	buf2 := appendULEB128(nil, 65536)
	if len(buf2) < 3 {
		t.Fatalf("expected at least 3 bytes for 65536, got %d", len(buf2))
	}
}

func TestNewVM_HostModuleError(t *testing.T) {
	ctx := context.Background()
	orig := newRuntime
	defer func() { newRuntime = orig }()

	newRuntime = func(ctx context.Context) wazero.Runtime {
		rt := wazero.NewRuntime(ctx)
		_, _ = rt.NewHostModuleBuilder("env").Instantiate(ctx)
		return rt
	}

	_, err := NewVM(ctx)
	if err == nil {
		t.Fatal("expected error from NewVM when env module already exists")
	}
}

func TestVM_Call_InstantiateCollision(t *testing.T) {
	ctx := context.Background()
	v, err := NewVM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close(ctx)

	wasm := buildCounterWASM()
	var addr [crypto.AddressSize]byte
	addr[0] = 77
	if err := v.Deploy(ctx, addr, wasm); err != nil {
		t.Fatal(err)
	}

	codeHash := crypto.Hash256(wasm)
	key := crypto.ToHex(codeHash)
	nextInstance := fmt.Sprintf("contract_%s_%d", key[:8], v.seq.Load()+1)
	_, _ = v.rt.NewHostModuleBuilder(nextInstance).Instantiate(ctx)

	ec := &ExecutionContext{
		CallerAddr:  addr,
		BlockHeight: 10,
		GasLimit:    100_000,
	}

	_, err = v.Call(WithExecCtx(ctx, ec), ec, codeHash, "get_count")
	if err == nil {
		t.Fatal("expected error on colliding module name in Call")
	}

	// Function not exported
	_, err = v.Call(WithExecCtx(ctx, ec), ec, codeHash, "non_existent_function")
	if err == nil {
		t.Fatal("expected error calling non-existent function")
	}

	_, err = v.Call(WithExecCtx(ctx, ec), ec, codeHash, "increment", 1, 2, 3)
	if err == nil {
		t.Fatal("expected error calling increment with invalid params")
	}
}

func TestVM_Close_MultipleTimes(t *testing.T) {
	ctx := context.Background()
	v, err := NewVM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	v.Close(ctx)
	v.Close(ctx)
}
