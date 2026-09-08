package bdd_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/qbc-qubitscoin/qubitscoin/internal/contracts/dex"
)

var _ = Describe("QubitSwap AMM", func() {

	var pool *dex.QubitSwap

	BeforeEach(func() {
		pool = dex.NewQubitSwap()
	})

	// ── Add Liquidity ──────────────────────────────────────────────────────

	Describe("AddLiquidity", func() {
		Context("when adding to an empty pool", func() {
			It("should set reserves correctly", func() {
				pool.AddLiquidity(5000, 10000)
				Expect(pool.ReserveA).To(Equal(uint64(5000)))
				Expect(pool.ReserveB).To(Equal(uint64(10000)))
			})
		})

		Context("when adding liquidity multiple times", func() {
			It("should accumulate reserves", func() {
				pool.AddLiquidity(1000, 2000)
				pool.AddLiquidity(500, 1000)
				Expect(pool.ReserveA).To(Equal(uint64(1500)))
				Expect(pool.ReserveB).To(Equal(uint64(3000)))
			})
		})

		Context("when adding zero amounts", func() {
			It("should leave reserves unchanged", func() {
				pool.AddLiquidity(0, 0)
				Expect(pool.ReserveA).To(Equal(uint64(0)))
				Expect(pool.ReserveB).To(Equal(uint64(0)))
			})
		})
	})

	// ── Swap A for B ───────────────────────────────────────────────────────

	Describe("SwapAforB", func() {
		Context("when the pool is empty", func() {
			It("should return failure", func() {
				out, ok := pool.SwapAforB(100)
				Expect(ok).To(BeFalse())
				Expect(out).To(Equal(uint64(0)))
			})
		})

		Context("when input is zero", func() {
			It("should return failure", func() {
				pool.AddLiquidity(10000, 10000)
				out, ok := pool.SwapAforB(0)
				Expect(ok).To(BeFalse())
				Expect(out).To(Equal(uint64(0)))
			})
		})

		Context("given a balanced pool of 10,000 A and 10,000 B", func() {
			BeforeEach(func() {
				pool.AddLiquidity(10_000, 10_000)
			})

			It("should succeed for a small swap", func() {
				out, ok := pool.SwapAforB(100)
				Expect(ok).To(BeTrue())
				Expect(out).To(BeNumerically(">", uint64(0)))
			})

			It("should apply the 0.3% fee (output less than naive)", func() {
				amountA := uint64(1000)
				naiveOut := (uint64(10_000) * amountA) / (uint64(10_000) + amountA)
				out, ok := pool.SwapAforB(amountA)
				Expect(ok).To(BeTrue())
				Expect(out).To(BeNumerically("<", naiveOut))
			})

			It("should update reserves after a swap", func() {
				amountA := uint64(500)
				out, ok := pool.SwapAforB(amountA)
				Expect(ok).To(BeTrue())
				Expect(pool.ReserveA).To(Equal(uint64(10_000) + amountA))
				Expect(pool.ReserveB).To(Equal(uint64(10_000) - out))
			})

			It("should maintain the k = x*y invariant", func() {
				kBefore := pool.ReserveA * pool.ReserveB
				_, ok := pool.SwapAforB(1000)
				Expect(ok).To(BeTrue())
				kAfter := pool.ReserveA * pool.ReserveB
				Expect(kAfter).To(BeNumerically(">=", kBefore))
			})

			It("should not drain the pool completely", func() {
				_, _ = pool.SwapAforB(1_000_000_000)
				Expect(pool.ReserveB).To(BeNumerically(">", uint64(0)))
			})
		})

		Context("given an asymmetric pool (1,000 A : 100,000 B)", func() {
			BeforeEach(func() {
				pool.AddLiquidity(1_000, 100_000)
			})

			It("should give a high B output for a small A input", func() {
				out, ok := pool.SwapAforB(10)
				Expect(ok).To(BeTrue())
				// 10 A in a 1000:100000 pool should buy ~990 B
				Expect(out).To(BeNumerically(">", uint64(500)))
			})
		})

		Context("when running multiple sequential swaps", func() {
			BeforeEach(func() {
				pool.AddLiquidity(100_000, 100_000)
			})

			It("should maintain k invariant across all swaps", func() {
				for i := 0; i < 20; i++ {
					kBefore := pool.ReserveA * pool.ReserveB
					_, ok := pool.SwapAforB(100)
					Expect(ok).To(BeTrue(), "swap %d should succeed", i)
					kAfter := pool.ReserveA * pool.ReserveB
					Expect(kAfter).To(BeNumerically(">=", kBefore), "k invariant violated at swap %d", i)
				}
			})
		})
	})
})

// Ensure the package compiles even if time is imported
var _ = time.Second
