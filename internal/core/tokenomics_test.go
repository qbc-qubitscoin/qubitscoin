package core

import "testing"

func TestBlockReward_GenesisIsZero(t *testing.T) {
	if r := BlockReward(0); r != 0 {
		t.Errorf("height 0: want 0, got %d", r)
	}
}

func TestBlockReward_Era0(t *testing.T) {
	for _, h := range []uint64{1, 500_000, HalvingInterval} {
		r := BlockReward(h)
		if r != InitialBlockReward {
			t.Errorf("height %d (era 0): want %d, got %d", h, InitialBlockReward, r)
		}
	}
}

func TestBlockReward_Era1(t *testing.T) {
	want := InitialBlockReward >> 1
	for _, h := range []uint64{HalvingInterval + 1, HalvingInterval + 500_000, 2 * HalvingInterval} {
		r := BlockReward(h)
		if r != want {
			t.Errorf("height %d (era 1): want %d, got %d", h, want, r)
		}
	}
}

func TestBlockReward_HalvingBoundary(t *testing.T) {
	// Last block of era 0 should be full reward.
	if r := BlockReward(HalvingInterval); r != InitialBlockReward {
		t.Errorf("last block of era 0: want %d, got %d", InitialBlockReward, r)
	}
	// First block of era 1 should be half.
	if r := BlockReward(HalvingInterval + 1); r != InitialBlockReward>>1 {
		t.Errorf("first block of era 1: want %d, got %d", InitialBlockReward>>1, r)
	}
}

func TestBlockReward_After32Halvings(t *testing.T) {
	h := 32*HalvingInterval + 1
	if r := BlockReward(h); r != 0 {
		t.Errorf("after 32 halving: want 0, got %d", r)
	}
}

func TestBlockReward_MonotonicallyDecreasing(t *testing.T) {
	prev := BlockReward(1)
	for era := uint64(1); era <= 10; era++ {
		h := era*HalvingInterval + 1
		cur := BlockReward(h)
		if cur >= prev {
			t.Errorf("era %d reward %d should be less than era %d reward %d",
				era, cur, era-1, prev)
		}
		prev = cur
	}
}

func TestTotalEmissionAt_Zero(t *testing.T) {
	if e := TotalEmissionAt(0); e != 0 {
		t.Errorf("height 0: want 0, got %d", e)
	}
}

func TestTotalEmissionAt_SingleBlock(t *testing.T) {
	e := TotalEmissionAt(1)
	if e != InitialBlockReward {
		t.Errorf("height 1: want %d, got %d", InitialBlockReward, e)
	}
}

func TestTotalEmissionAt_FullEra0(t *testing.T) {
	e := TotalEmissionAt(HalvingInterval)
	want := InitialBlockReward * HalvingInterval
	if e != want {
		t.Errorf("full era 0: want %d, got %d", want, e)
	}
}

func TestTotalEmissionAt_NeverExceedsMaxEmission(t *testing.T) {
	// 90M QBC max emission from mining.
	maxEmission := uint64(90_000_000) * OneQBC
	// Check beyond all halving.
	for _, h := range []uint64{
		HalvingInterval, 2 * HalvingInterval, 10 * HalvingInterval, 33 * HalvingInterval,
	} {
		e := TotalEmissionAt(h)
		if e > maxEmission {
			t.Errorf("height %d: emission %d exceeds max %d", h, e, maxEmission)
		}
	}
}

func TestCirculatingSupply_IncludesPremine(t *testing.T) {
	s := CirculatingSupply(0)
	if s != GenesisPremine {
		t.Errorf("height 0: want premine %d, got %d", GenesisPremine, s)
	}
}

func TestCirculatingSupply_GrowsWithHeight(t *testing.T) {
	s1 := CirculatingSupply(1)
	s2 := CirculatingSupply(1000)
	if s2 <= s1 {
		t.Errorf("supply at 1000 (%d) should exceed supply at 1 (%d)", s2, s1)
	}
}

func TestCirculatingSupply_NeverExceedsMaxSupply(t *testing.T) {
	// Check well beyond all halvings.
	s := CirculatingSupply(50 * HalvingInterval)
	if s > MaxSupply {
		t.Errorf("supply %d exceeds MaxSupply %d", s, MaxSupply)
	}
}
