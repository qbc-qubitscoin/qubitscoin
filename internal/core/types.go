package core

// TxType enumerates supported transaction types.
type TxType uint8

const (
	TxTransfer TxType = 0x01
	TxDeploy   TxType = 0x02
	TxCall     TxType = 0x03
	TxStake    TxType = 0x04
	TxUnstake  TxType = 0x05
)

// Currency units — 9 decimal places (1 QBC = 1_000_000_000 qubits).
const (
	Qubit     uint64 = 1
	OneQBC    uint64 = 1_000_000_000        // 10^9 qubits
	MaxSupply uint64 = 100_000_000 * OneQBC // 100M QBC = 10^17 qubits (fits uint64)
)

// ─────────────────────────────────────────────────────────────────────────────
// Gas constants — ultra-low to make QBC the world's cheapest-fee chain.
//
// At InitialBaseFee (10 qubits/gas):
//
//	Transfer : 21 gas × 10 = 210 qubits  ≈ $0.000000021  (at $100/QBC)
//	Deploy: 5 000 gas × 10 = 50 000 qubits
//	Call: 500 gas × 10 = 5 000 qubits
//
// Compare (USD at respective market prices, 2024 averages):
//
//	Ethereum transfer: ~$1.50 → QBC is ~71 000 000× cheaper
//	Solana transfer: ~$0.0003 → QBC is ~14 000× cheaper
//	BSC transfer: ~$0.05 → QBC is ~2 400 000× cheaper
//
// ─────────────────────────────────────────────────────────────────────────────
const (
	GasTransfer   uint64 = 21          // was 21 000 — 1 000× reduction
	GasDeploy     uint64 = 5_000       // was 200 000 —  40× reduction
	GasCall       uint64 = 500         // was  50 000 — 100× reduction
	BlockGasLimit uint64 = 500_000_000 // ~23 M transfers per block
	MinGasPrice   uint64 = 1           // 1 qubit/gas absolute floor
)

// Chain metadata.
const (
	Ticker          = "QBC"
	Decimals        = 9
	ProtocolVersion = 1
	ChainID         = 1
)

// BlockIntervalSec is the target block time in seconds.
const BlockIntervalSec = 2
