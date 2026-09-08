package state

import (
	"testing"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
)

// addrA and addrB are defined in transition_test.go (same package).

func TestApplyBlock_EmptyBlock_CreditsReward(t *testing.T) {
	st := NewStateDB()
	blk := &core.Block{Header: core.BlockHeader{Height: 1}}

	result, err := ApplyBlock(st, blk, addrA, nil)
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}

	expectedReward := core.BlockReward(1)
	if result.BlockReward != expectedReward {
		t.Errorf("BlockReward: want %d, got %d", expectedReward, result.BlockReward)
	}
	if result.FeeCollected != 0 {
		t.Errorf("FeeCollected: want 0, got %d", result.FeeCollected)
	}
	if st.GetBalance(addrA) != expectedReward {
		t.Errorf("validator balance: want %d, got %d", expectedReward, st.GetBalance(addrA))
	}
}

func TestApplyBlock_GenesisNoReward(t *testing.T) {
	st := NewStateDB()
	blk := &core.Block{Header: core.BlockHeader{Height: 0}}

	result, err := ApplyBlock(st, blk, addrA, nil)
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}
	if result.BlockReward != 0 {
		t.Errorf("genesis block should have zero reward, got %d", result.BlockReward)
	}
}

func TestApplyBlock_TotalValidatorIncome(t *testing.T) {
	st := NewStateDB()
	blk := &core.Block{Header: core.BlockHeader{Height: 1}}

	result, err := ApplyBlock(st, blk, addrA, nil)
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}
	// TotalValidatorIncome = ValidatorTip + BlockReward (BurnedFees are destroyed).
	want := result.ValidatorTip + result.BlockReward
	if result.TotalValidatorIncome() != want {
		t.Errorf("TotalValidatorIncome: want %d, got %d", want, result.TotalValidatorIncome())
	}
}

func TestApplyBlock_WithTransfer_FeeGoesToValidator(t *testing.T) {
	st := NewStateDB()
	st.SetAccount(addrA, &Account{Balance: 10_000_000 * core.OneQBC})

	gasPrice := core.MinGasPrice
	tx := makeTx(core.TxTransfer, addrA, addrB, 0, 1*core.OneQBC, core.GasTransfer, gasPrice, nil)
	blk := &core.Block{
		Header: core.BlockHeader{Height: 1},
		Txs:    []*core.Transaction{tx},
	}

	result, err := ApplyBlock(st, blk, addrA, nil)
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}

	expectedFee := core.GasTransfer * gasPrice
	expectedReward := core.BlockReward(1)
	// BaseFee in this test block is 0 (default), so all fees go to the validator as tip.
	// ValidatorTip = FeeCollected − BurnedFee = expectedFee − 0 = expectedFee.
	expectedIncome := expectedFee + expectedReward

	if result.FeeCollected != expectedFee {
		t.Errorf("FeeCollected: want %d, got %d", expectedFee, result.FeeCollected)
	}
	if result.ValidatorTip != expectedFee {
		t.Errorf("ValidatorTip: want %d (all fees, baseFee=0), got %d", expectedFee, result.ValidatorTip)
	}
	if result.TotalValidatorIncome() != expectedIncome {
		t.Errorf("TotalValidatorIncome: want %d, got %d", expectedIncome, result.TotalValidatorIncome())
	}
}

func TestApplyBlock_RewardHalves(t *testing.T) {
	st1 := NewStateDB()
	blk1 := &core.Block{Header: core.BlockHeader{Height: 1}}
	r1, _ := ApplyBlock(st1, blk1, addrA, nil)

	st2 := NewStateDB()
	blk2 := &core.Block{Header: core.BlockHeader{Height: core.HalvingInterval + 1}}
	r2, _ := ApplyBlock(st2, blk2, addrA, nil)

	if r1.BlockReward != r2.BlockReward*2 {
		t.Errorf("era 0 reward %d should be twice era 1 reward %d", r1.BlockReward, r2.BlockReward)
	}
}

func TestApplyBlock_TxResultsCount(t *testing.T) {
	st := NewStateDB()
	st.SetAccount(addrA, &Account{Balance: 100_000_000 * core.OneQBC})

	txs := make([]*core.Transaction, 3)
	for i := range txs {
		txs[i] = makeTx(core.TxTransfer, addrA, addrB, uint64(i), 1, core.GasTransfer, core.MinGasPrice, nil)
	}
	blk := &core.Block{Header: core.BlockHeader{Height: 1}, Txs: txs}

	result, err := ApplyBlock(st, blk, addrA, nil)
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}
	if len(result.TxResults) != 3 {
		t.Errorf("TxResults count: want 3, got %d", len(result.TxResults))
	}
}

func TestApplyBlock_GasUsedSummed(t *testing.T) {
	st := NewStateDB()
	st.SetAccount(addrA, &Account{Balance: 100_000_000 * core.OneQBC})

	tx1 := makeTx(core.TxTransfer, addrA, addrB, 0, 1, core.GasTransfer, core.MinGasPrice, nil)
	tx2 := makeTx(core.TxTransfer, addrA, addrB, 1, 1, core.GasTransfer, core.MinGasPrice, nil)
	blk := &core.Block{
		Header: core.BlockHeader{Height: 1},
		Txs:    []*core.Transaction{tx1, tx2},
	}

	result, err := ApplyBlock(st, blk, addrA, nil)
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}
	if result.GasUsed != core.GasTransfer*2 {
		t.Errorf("GasUsed: want %d, got %d", core.GasTransfer*2, result.GasUsed)
	}
}
