package state

import (
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// makeTx builds a minimal unsigned transaction for state-transition tests.
// ApplyTransaction does NOT verify signatures — only balance/nonce/gas.
func makeTx(txType core.TxType, from, to [crypto.AddressSize]byte, nonce, amount, gasLimit, gasPrice uint64, data []byte) *core.Transaction {
	return &core.Transaction{
		Version:   1,
		Type:      txType,
		Nonce:     nonce,
		From:      from,
		To:        to,
		Amount:    amount,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		Timestamp: time.Now().UnixNano(),
		Data:      data,
	}
}

func stateWithBalance(addr [crypto.AddressSize]byte, balance uint64) *DB {
	st := NewStateDB()
	st.SetAccount(addr, &Account{Balance: balance})
	return st
}

var (
	addrA = [crypto.AddressSize]byte{0xAA}
	addrB = [crypto.AddressSize]byte{0xBB}
)

// ── Transfer ──────────────────────────────────────────────────────────────────

func TestApplyTransaction_Transfer_Success(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 100*core.OneQBC, core.GasTransfer, core.MinGasPrice, nil)

	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}
	if st.GetBalance(addrB) != 100*core.OneQBC {
		t.Errorf("recipient balance: want %d, got %d", 100*core.OneQBC, st.GetBalance(addrB))
	}
	if res.GasUsed != core.GasTransfer {
		t.Errorf("GasUsed: want %d, got %d", core.GasTransfer, res.GasUsed)
	}
}

func TestApplyTransaction_Transfer_FeeCollected(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	gasPrice := core.MinGasPrice
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 0, core.GasTransfer, gasPrice, nil)

	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFee := core.GasTransfer * gasPrice
	if res.FeeCollected != wantFee {
		t.Errorf("FeeCollected: want %d, got %d", wantFee, res.FeeCollected)
	}
}

func TestApplyTransaction_Transfer_GasRefund(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	// Set gas limit higher than intrinsic cost — sender should be refunded.
	gasLimit := core.GasTransfer * 2
	gasPrice := core.MinGasPrice
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 0, gasLimit, gasPrice, nil)

	balBefore := st.GetBalance(addrA)
	res, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	balAfter := st.GetBalance(addrA)
	expectedDeduction := res.GasUsed * gasPrice
	actualDeduction := balBefore - balAfter
	if actualDeduction != expectedDeduction {
		t.Errorf("sender deduction: want %d, got %d", expectedDeduction, actualDeduction)
	}
}

func TestApplyTransaction_Transfer_InsufficientBalance(t *testing.T) {
	st := stateWithBalance(addrA, 1*core.OneQBC)
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 100*core.OneQBC, core.GasTransfer, core.MinGasPrice, nil)

	_, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err == nil {
		t.Fatal("expected error for insufficient balance")
	}
}

func TestApplyTransaction_Transfer_NonceMismatch(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	// Account nonce is 0, but tx nonce is 5.
	tx := makeTx(core.TxTransfer, addrA, addrB, 5, 1, core.GasTransfer, core.MinGasPrice, nil)

	_, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err == nil {
		t.Fatal("expected nonce mismatch error")
	}
}

func TestApplyTransaction_Transfer_NonceIncrement(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 1, core.GasTransfer, core.MinGasPrice, nil)

	_, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.GetNonce(addrA) != 1 {
		t.Errorf("sender nonce: want 1, got %d", st.GetNonce(addrA))
	}
}

func TestApplyTransaction_Transfer_GasLimitBelowIntrinsic(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 1, core.GasTransfer-1, core.MinGasPrice, nil)

	_, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
	if err == nil {
		t.Fatal("expected error when gas limit < intrinsic cost")
	}
}

func TestApplyTransaction_Transfer_ExceedsBlockGas(t *testing.T) {
	st := stateWithBalance(addrA, 1_000_000*core.OneQBC)
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 1, core.GasTransfer, core.MinGasPrice, nil)

	_, err := ApplyTransaction(st, tx, core.GasTransfer-1, nil, core.MinBaseFee)
	if err == nil {
		t.Fatal("expected error when tx gas exceeds remaining block gas")
	}
}

func TestApplyTransaction_Transfer_SequentialNonces(t *testing.T) {
	st := stateWithBalance(addrA, 10_000_000*core.OneQBC)
	for i := uint64(0); i < 5; i++ {
		tx := makeTx(core.TxTransfer, addrA, addrB, i, 1, core.GasTransfer, core.MinGasPrice, nil)
		_, err := ApplyTransaction(st, tx, core.BlockGasLimit, nil, core.MinBaseFee)
		if err != nil {
			t.Fatalf("tx %d failed: %v", i, err)
		}
	}
	if st.GetNonce(addrA) != 5 {
		t.Errorf("nonce after 5 txs: want 5, got %d", st.GetNonce(addrA))
	}
}

// ── IntrinsicGas ─────────────────────────────────────────────────────────────

func TestIntrinsicGas_Transfer(t *testing.T) {
	tx := &core.Transaction{Type: core.TxTransfer}
	if g := IntrinsicGas(tx); g != core.GasTransfer {
		t.Errorf("want %d, got %d", core.GasTransfer, g)
	}
}

func TestIntrinsicGas_Deploy_WithData(t *testing.T) {
	data := make([]byte, 100)
	tx := &core.Transaction{Type: core.TxDeploy, Data: data}
	want := core.GasDeploy + uint64(len(data))*68
	if g := IntrinsicGas(tx); g != want {
		t.Errorf("want %d, got %d", want, g)
	}
}

func TestIntrinsicGas_Call_WithData(t *testing.T) {
	data := make([]byte, 50)
	tx := &core.Transaction{Type: core.TxCall, Data: data}
	want := core.GasCall + uint64(len(data))*16
	if g := IntrinsicGas(tx); g != want {
		t.Errorf("want %d, got %d", want, g)
	}
}
