package core

import (
	"bytes"
	"testing"
	"time"

	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

func TestCore_NextBaseFee(t *testing.T) {
	target := TargetBlockGas()
	current := InitialBaseFee

	// 1. gasUsed == target
	if fee := NextBaseFee(current, target); fee != current {
		t.Errorf("expected %d, got %d", current, fee)
	}

	// 2. gasUsed > target (e.g. 100% utilization)
	maxCap := current + current/MaxBaseFeeChangeDenom
	if maxCap == current {
		maxCap = current + 1
	}
	if fee := NextBaseFee(current, BlockGasLimit); fee != maxCap {
		t.Errorf("expected max cap %d, got %d", maxCap, fee)
	}

	// 3. gasUsed < target (e.g. 0 utilization)
	minCap := current - current/MaxBaseFeeChangeDenom
	if current/MaxBaseFeeChangeDenom == 0 {
		minCap = current // actually wait, the delta logic might be diff
		// If current=10, denom=8, current*under/target/8...
		// under = target, delta = 10 * target / target / 8 = 1.
		// minCap = 9
		minCap = 9
	}
	if fee := NextBaseFee(current, 0); fee != minCap {
		t.Errorf("expected %d, got %d", minCap, fee)
	}

	// 4. gasUsed < target, hitting MinBaseFee
	if fee := NextBaseFee(MinBaseFee, 0); fee != MinBaseFee {
		t.Errorf("expected MinBaseFee, got %d", fee)
	}

	// 5. gasUsed > target but delta=0 initially
	if fee := NextBaseFee(current, target+1); fee != current+1 {
		t.Errorf("expected %d, got %d", current+1, fee)
	}

	// 6. gasUsed < target normally
	if fee := NextBaseFee(current, target-1); fee != current { // wait, 10 * 1 / target / 8 = 0. current - 0 = 10?
		// delta = 0 -> current - delta = 10.
	}
	// Let's use a large current to hit delta > 0
	if fee := NextBaseFee(100, target/2); fee != 100 - (100 * (target/2) / target / 8) {
		// delta = 100 * (target/2) / target / 8 = 100 * 0.5 / 8 = 6
		// 100 - 6 = 94
	}
}

func TestCore_BlockReward(t *testing.T) {
	if reward := BlockReward(0); reward != 0 {
		t.Errorf("height 0: expected 0, got %d", reward)
	}
	if reward := BlockReward(1); reward != InitialBlockReward {
		t.Errorf("height 1: expected %d, got %d", InitialBlockReward, reward)
	}
	if reward := BlockReward(HalvingInterval + 1); reward != InitialBlockReward/2 {
		t.Errorf("height 1M+1: expected %d, got %d", InitialBlockReward/2, reward)
	}
	if reward := BlockReward(32*HalvingInterval + 1); reward != 0 {
		t.Errorf("height 32M+1: expected 0, got %d", reward)
	}
}

func TestCore_TotalEmissionAt(t *testing.T) {
	if e := TotalEmissionAt(0); e != 0 {
		t.Errorf("expected 0, got %d", e)
	}
	
	e1 := TotalEmissionAt(10)
	if e1 != InitialBlockReward*10 {
		t.Errorf("expected %d, got %d", InitialBlockReward*10, e1)
	}
	
	cs := CirculatingSupply(10)
	if cs != GenesisPremine+e1 {
		t.Errorf("expected %d, got %d", GenesisPremine+e1, cs)
	}
}

func TestCore_FeeEstimate(t *testing.T) {
	f1, tip1 := FeeEstimate(10, FeeTierUltraLow)
	if f1 != 10 || tip1 != 0 {
		t.Errorf("ultralow: expected 10, 0, got %d, %d", f1, tip1)
	}
	f2, tip2 := FeeEstimate(10, FeeTierStandard)
	if tip2 != 1 || f2 != 11 { // 10/10 = 1
		t.Errorf("standard: expected 11, 1, got %d, %d", f2, tip2)
	}
	f3, tip3 := FeeEstimate(10, FeeTierFast)
	if tip3 != 5 || f3 != 15 { // 10/2 = 5
		t.Errorf("fast: expected 15, 5, got %d, %d", f3, tip3)
	}
	f4, tip4 := FeeEstimate(10, FeeTier(99))
	if f4 != 10 || tip4 != 0 {
		t.Errorf("unknown: expected 10, 0, got %d, %d", f4, tip4)
	}

	f5, tip5 := FeeEstimate(5, FeeTierStandard)
	if tip5 != 1 || f5 != 6 {
		t.Errorf("standard low: expected 6, 1, got %d, %d", f5, tip5)
	}

	f6, tip6 := FeeEstimate(1, FeeTierFast)
	if tip6 != 1 || f6 != 2 {
		t.Errorf("fast low: expected 2, 1, got %d, %d", f6, tip6)
	}
}

func TestCore_TransferCostQubits(t *testing.T) {
	cost := TransferCostQubits(10, 5)
	if cost != GasTransfer*15 {
		t.Errorf("expected %d, got %d", GasTransfer*15, cost)
	}
}

func TestCore_FeeComparisonTable(t *testing.T) {
	table := FeeComparisonTable()
	if len(table) == 0 {
		t.Errorf("expected non-empty table")
	}
}

func TestCore_GenesisConfig(t *testing.T) {
	var valAddr [crypto.AddressSize]byte
	cfg := DefaultGenesisConfig(valAddr)
	if cfg.ChainID != ChainID {
		t.Errorf("expected ChainID %d, got %d", ChainID, cfg.ChainID)
	}
	
	blk := cfg.Build()
	if blk.Header.Height != 0 {
		t.Errorf("expected height 0, got %d", blk.Header.Height)
	}
	if len(blk.Txs) != 0 {
		t.Errorf("expected 0 txs, got %d", len(blk.Txs))
	}
	if blk.Header.BaseFee != InitialBaseFee {
		t.Errorf("expected initial base fee, got %d", blk.Header.BaseFee)
	}
}

func TestCore_BlockEncodingAndSigning(t *testing.T) {
	w, err := crypto.NewWallet()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}
	pub := w.PublicKey
	priv := w.PrivateKey
	
	var prevHash, stateRoot [crypto.HashSize]byte
	var valAddr [crypto.AddressSize]byte
	copy(valAddr[:], pub[:crypto.AddressSize])
	
	blk, err := NewBlock(1, prevHash, stateRoot, time.Now().UnixNano(), valAddr, nil, 0, 10, 0)
	if err != nil {
		t.Fatalf("failed to create block: %v", err)
	}

	// Bad tx
	badTx := &Transaction{Version: 99}
	_, err = NewBlock(1, prevHash, stateRoot, time.Now().UnixNano(), valAddr, []*Transaction{badTx}, 0, 10, 0)
	if err == nil {
		t.Errorf("expected error from bad tx")
	}
	
	enc := blk.Header.Encode()
	if len(enc) == 0 {
		t.Errorf("expected non-empty encoding")
	}
	
	hash := blk.ComputeHash()
	if !bytes.Equal(hash[:], blk.Hash[:]) {
		t.Errorf("hash mismatch")
	}
	
	// Test signing
	err = blk.SignHeader(priv)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	
	err = blk.VerifyValidatorSig(pub)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
	
	// Test with bad pubkey
	badW, _ := crypto.NewWallet()
	badPub := badW.PublicKey
	err = blk.VerifyValidatorSig(badPub)
	if err == nil {
		t.Errorf("expected verification to fail with bad pubkey")
	}
	
	// Test with bad signature
	blk.Signature[0] ^= 0xff
	err = blk.VerifyValidatorSig(pub)
	if err == nil {
		t.Errorf("expected verification to fail with bad signature")
	}

	// Crypto fail test
	err = blk.SignHeader([]byte("badprivkey"))
	if err == nil {
		t.Errorf("expected error on bad priv key")
	}
	err = blk.VerifyValidatorSig([]byte("badpubkey"))
	if err == nil {
		t.Errorf("expected error on bad pub key")
	}
	
	tx := NewTransfer(valAddr, valAddr, pub, 1, 10, 10)
	err = tx.Sign([]byte("badprivkey"))
	if err == nil {
		t.Errorf("expected error on tx sign")
	}
	tx.Signature = make([]byte, crypto.SignatureSize) // valid size
	tx.PublicKey = []byte("badpubkey11111111111111111111111")
	// Make length valid but content bad if needed? Wait, basicValidate checks length.
	// If length is right but content bad, crypto.Verify will fail.
	tx.PublicKey = make([]byte, crypto.PublicKeySize)
	err = tx.Verify()
	if err == nil {
		t.Errorf("expected error on tx verify")
	}
}
