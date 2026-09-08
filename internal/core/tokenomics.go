package core

// Emission schedule constants.
//
// Total max supply: 100M QBC
//   - 10M QBC pre-mined at genesis (10 %)
//   - 90M QBC emitted as block rewards via halving schedule
//
// Reward halves every HalvingInterval blocks.
// Sum of the infinite geometric series:
//
//	InitialBlockReward * HalvingInterval * 2 = 45 QBC * 1,000,000 * 2 = 90M QBC ✓
const (
	InitialBlockReward uint64 = 45 * OneQBC         // 45 QBC at genesis era
	HalvingInterval    uint64 = 1_000_000           // halve every 1M blocks (~23 days at 2s/block)
	GenesisPremine     uint64 = 10_000_000 * OneQBC // 10M QBC to bootstrap the validator
)

// BlockReward returns the block subsidy (in qubits) at the given block height.
//
// Height 0 (genesis) always returns 0 — genesis has no coinbase.
// After 32 halving, the reward rounds down to 0 and stays there.
//
// Era 0: heights 1 – 1,000,000 → 45 QBC
// Era 1: heights 1,000,001 – 2,000,000 → 22.5 QBC (integer: 22_500_000_000)
// Era 2: heights 2,000,001 – 3,000,000 → 11.25 QBC …
func BlockReward(height uint64) uint64 {
	if height == 0 {
		return 0
	}
	era := (height - 1) / HalvingInterval
	if era >= 32 { // after 32 halvings reward is negligible / zero
		return 0
	}
	return InitialBlockReward >> era
}

// TotalEmissionAt returns the cumulative block reward emission up to (and
// including) the given height — useful for supply tracking / dashboards.
func TotalEmissionAt(height uint64) uint64 {
	if height == 0 {
		return 0
	}
	var total uint64
	for era := uint64(0); era < 32; era++ {
		start := era*HalvingInterval + 1
		end := start + HalvingInterval - 1
		if start > height {
			break
		}
		if end > height {
			end = height
		}
		reward := InitialBlockReward >> era
		total += reward * (end - start + 1)
	}
	return total
}

// CirculatingSupply returns the total QBC in existence at the given height:
// genesis pre-mine + block reward emission so far.
func CirculatingSupply(height uint64) uint64 {
	return GenesisPremine + TotalEmissionAt(height)
}
