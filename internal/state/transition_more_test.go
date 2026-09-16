package state

import (
	"context"
	"testing"
	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
	"github.com/qbc-qubitscoin/qubitscoin/internal/vm"
)

// ── Deploy ────────────────────────────────────────────────────────────────────

func TestApplyTransaction_Deploy_NoData(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(core.TxDeploy, addrA, addrB, 0, 0, core.GasDeploy, core.MinGasPrice, nil)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected structural error: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure for deploy tx without bytecode")
	}
	if res.GasUsed != core.GasDeploy {
		t.Errorf("GasUsed: want %d, got %d", core.GasDeploy, res.GasUsed)
	}
}

func TestApplyTransaction_Deploy_Success_NoVM(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	data := []byte{0x01, 0x02}
	tx := makeTx(core.TxDeploy, addrA, addrB, 0, 100, core.GasDeploy + 68*2, core.MinGasPrice, data)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}
	// Check contract address balance
	if len(res.ReturnData) != crypto.AddressSize {
		t.Fatalf("expected %d bytes return data (address), got %d", crypto.AddressSize, len(res.ReturnData))
	}
	var contractAddr [crypto.AddressSize]byte
	copy(contractAddr[:], res.ReturnData)
	if st.GetBalance(contractAddr) != 100 {
		t.Errorf("contract balance: want 100, got %d", st.GetBalance(contractAddr))
	}
}

func TestApplyTransaction_Deploy_WithVM(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	
	ctx := context.Background()
	execVM, err := vm.NewVM(ctx)
	if err != nil {
		t.Fatalf("failed to create VM: %v", err)
	}
	defer execVM.Close(ctx)
	
	// Create a dummy WASM module that exports _init
	wasmCode := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // Magic + version
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00, // Type section
		0x03, 0x02, 0x01, 0x00, // Function section
		0x07, 0x09, 0x01, 0x05, 0x5f, 0x69, 0x6e, 0x69, 0x74, 0x00, 0x00, // Export section "_init"
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b, // Code section
	}
	
	tx := makeTx(core.TxDeploy, addrA, addrB, 0, 100, core.GasDeploy + 68*uint64(len(wasmCode)) + 10000, core.MinGasPrice, wasmCode)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, execVM, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}
}

// ── Call ──────────────────────────────────────────────────────────────────────

func TestApplyTransaction_Call_NotContract(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(core.TxCall, addrA, addrB, 0, 0, core.GasCall, core.MinGasPrice, nil)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure for calling non-contract")
	}
	if res.GasUsed != core.GasCall {
		t.Errorf("GasUsed: want %d, got %d", core.GasCall, res.GasUsed)
	}
}

func TestApplyTransaction_Call_Success_NoVM(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	contractAddr := addrB
	contractAcct := &Account{CodeHash: [32]byte{1}}
	st.SetAccount(contractAddr, contractAcct)
	
	tx := makeTx(core.TxCall, addrA, contractAddr, 0, 100, core.GasCall, core.MinGasPrice, nil)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.GetBalance(contractAddr) != 100 {
		t.Errorf("contract balance: want 100, got %d", st.GetBalance(contractAddr))
	}
	if res.GasUsed != core.GasCall {
		t.Errorf("GasUsed: want %d, got %d", core.GasCall, res.GasUsed)
	}
}

func TestApplyTransaction_Call_WithVM(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	
	ctx := context.Background()
	execVM, err := vm.NewVM(ctx)
	if err != nil {
		t.Fatalf("failed to create VM: %v", err)
	}
	defer execVM.Close(ctx)
	wasmCode := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 
		0x01, 0x06, 0x01, 0x60, 0x01, 0x7e, 0x01, 0x7e, // Type: func(i64) -> i64
		0x03, 0x02, 0x01, 0x00, 
		0x07, 0x08, 0x01, 0x04, 0x63, 0x61, 0x6c, 0x6c, 0x00, 0x00, // Export "call"
		0x0a, 0x06, 0x01, 0x04, 0x00, 0x42, 0x2a, 0x0b, 
	}
	codeHash := crypto.Hash256(wasmCode)
	
	contractAddr := addrB
	contractAcct := &Account{CodeHash: codeHash}
	st.SetAccount(contractAddr, contractAcct)
	
	err = execVM.Deploy(ctx, contractAddr, wasmCode)
	if err != nil {
		t.Fatalf("failed to deploy to VM: %v", err)
	}
	
	// Create tx data calling "call" with 1 parameter
	data := []byte{4, 'c', 'a', 'l', 'l', 0, 0, 0, 0, 0, 0, 0, 42} // len=4, "call", param=42
	
	tx := makeTx(core.TxCall, addrA, contractAddr, 0, 100, core.GasCall + 16*13 + 10000, core.MinGasPrice, data)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, execVM, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}
	
	// Test fallback funcName with length < 1+nameLen
	txFallback := makeTx(core.TxCall, addrA, contractAddr, 1, 0, core.GasCall + 16*1 + 10000, core.MinGasPrice, []byte{100})
	ApplyTransaction(st, txFallback, core.BlockGasLimit, execVM, core.MinBaseFee)
}

func TestApplyTransaction_Call_Revert(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	
	ctx := context.Background()
	execVM, err := vm.NewVM(ctx)
	if err != nil {
		t.Fatalf("failed to create VM: %v", err)
	}
	defer execVM.Close(ctx)
	
	// Create a contract that will fail (e.g. traps)
	wasmCode := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00, 
		0x03, 0x02, 0x01, 0x00, 
		0x07, 0x08, 0x01, 0x04, 0x63, 0x61, 0x6c, 0x6c, 0x00, 0x00, 
		0x0a, 0x05, 0x01, 0x03, 0x00, 0x00, 0x0b, // unreachable instruction (0x00)
	}
	codeHash := crypto.Hash256(wasmCode)
	
	contractAddr := addrB
	contractAcct := &Account{CodeHash: codeHash}
	st.SetAccount(contractAddr, contractAcct)
	
	err = execVM.Deploy(ctx, contractAddr, wasmCode)
	if err != nil {
		t.Fatalf("failed to deploy to VM: %v", err)
	}
	
	data := []byte{4, 'c', 'a', 'l', 'l'}
	
	tx := makeTx(core.TxCall, addrA, contractAddr, 0, 100, core.GasCall + 16*5 + 10000, core.MinGasPrice, data)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, execVM, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected failure due to revert")
	}
}

func TestApplyTransaction_UnknownType(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(255, addrA, addrB, 0, 0, core.GasTransfer, core.MinGasPrice, nil)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure for unknown tx type")
	}
	if res.Error == nil {
		t.Fatal("expected result error")
	}
}

func TestApplyCall_InsufficientBalance(t *testing.T) {
	st := stateWithBalance(addrA, 50)
	tx := makeTx(core.TxCall, addrA, addrB, 0, 100, core.GasCall, core.MinGasPrice, nil)
	st.SetAccount(addrB, &Account{CodeHash: [32]byte{1}})
	_, _, err := applyCall(st, tx, nil, 1000)
	if err == nil || err.Error() != "insufficient balance for call value" {
		t.Fatalf("expected insufficient balance for call value, got %v", err)
	}
}

func TestApplyBlock_EmptyTxs(t *testing.T) {
	st := NewStateDB()
	block := &core.Block{
		Header: core.BlockHeader{
			Height: 1,
		},
		Txs: nil,
	}
	res, err := ApplyBlock(st, block, addrA, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedReward := core.BlockReward(1)
	if res.BlockReward != expectedReward {
		t.Errorf("expected block reward %d, got %d", expectedReward, res.BlockReward)
	}
	if res.TotalValidatorIncome() != expectedReward {
		t.Errorf("expected validator income %d, got %d", expectedReward, res.TotalValidatorIncome())
	}
}
