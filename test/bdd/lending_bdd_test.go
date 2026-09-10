package bdd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/qbc-qubitscoin/qubitscoin/internal/contracts/lending"
)

var _ = Describe("QubitLend Protocol", func() {

	var market *lending.QubitLend

	BeforeEach(func() {
		market = lending.NewQubitLend()
		market.UpdateOraclePrice(100) // default: 1 QBC = $100
	})

	// ── Deposit Collateral ─────────────────────────────────────────────────

	Describe("DepositCollateral", func() {
		Context("when a new user deposits", func() {
			It("should create a position with the correct collateral", func() {
				market.DepositCollateral("alice", 10)
				Expect(market.Positions["alice"]).NotTo(BeNil())
				Expect(market.Positions["alice"].CollateralQBC).To(Equal(uint64(10)))
			})
		})

		Context("when a user deposits multiple times", func() {
			It("should accumulate collateral", func() {
				market.DepositCollateral("alice", 5)
				market.DepositCollateral("alice", 5)
				Expect(market.Positions["alice"].CollateralQBC).To(Equal(uint64(10)))
			})
		})
	})

	// ── Borrow QUSD ────────────────────────────────────────────────────────

	Describe("BorrowQUSD", func() {
		Context("given a user with 10 QBC collateral at $100 each ($1000 total)", func() {
			BeforeEach(func() {
				market.DepositCollateral("alice", 10) // $1000 collateral
			})

			Context("when borrowing within the 150% collateralization ratio ($666 max)", func() {
				It("should succeed for $500 borrow", func() {
					Expect(market.BorrowQUSD("alice", 500)).To(BeTrue())
				})

				It("should record the debt", func() {
					market.BorrowQUSD("alice", 500)
					Expect(market.Positions["alice"].DebtQUSD).To(Equal(uint64(500)))
				})
			})

			Context("when attempting to borrow beyond the 150% CR limit ($700)", func() {
				It("should fail and leave debt unchanged", func() {
					Expect(market.BorrowQUSD("alice", 700)).To(BeFalse())
					Expect(market.Positions["alice"].DebtQUSD).To(Equal(uint64(0)))
				})
			})

			Context("when accumulating debt across multiple borrows", func() {
				It("should succeed up to the limit and then fail", func() {
					Expect(market.BorrowQUSD("alice", 300)).To(BeTrue())
					Expect(market.BorrowQUSD("alice", 300)).To(BeTrue())
					// Total: $600. Next $100 would be $700 > $666 limit
					Expect(market.BorrowQUSD("alice", 100)).To(BeFalse())
				})
			})
		})

		Context("when user has no position", func() {
			It("should fail", func() {
				Expect(market.BorrowQUSD("ghost", 100)).To(BeFalse())
			})
		})
	})

	// ── Liquidation ────────────────────────────────────────────────────────

	Describe("Liquidate", func() {
		Context("given a healthy position (CR > 150%)", func() {
			BeforeEach(func() {
				market.DepositCollateral("bob", 10) // $1000
				market.BorrowQUSD("bob", 600)       // CR = 166%
			})

			It("should NOT be liquidatable at current price", func() {
				Expect(market.Liquidate("bob")).To(BeFalse())
			})

			Context("when the oracle price drops to $80 (CR falls to 133%)", func() {
				BeforeEach(func() {
					market.UpdateOraclePrice(80)
				})

				It("should be liquidatable", func() {
					Expect(market.Liquidate("bob")).To(BeTrue())
				})

				It("should zero out collateral and debt after liquidation", func() {
					market.Liquidate("bob")
					Expect(market.Positions["bob"].CollateralQBC).To(Equal(uint64(0)))
					Expect(market.Positions["bob"].DebtQUSD).To(Equal(uint64(0)))
				})
			})
		})

		Context("when there is no debt", func() {
			BeforeEach(func() {
				market.DepositCollateral("charlie", 5)
			})

			It("should return false — nothing to liquidate", func() {
				Expect(market.Liquidate("charlie")).To(BeFalse())
			})
		})

		Context("when the user does not exist", func() {
			It("should return false", func() {
				Expect(market.Liquidate("nobody")).To(BeFalse())
			})
		})

		// ── Market Stress Scenario ─────────────────────────────────────────

		Context("simulating a market crash (Phase 11 exit criteria)", func() {
			It("should liquidate all underwater positions", func() {
				By("setting up 3 users with varying positions")
				market.DepositCollateral("user1", 10)
				market.BorrowQUSD("user1", 600) // healthy at $100

				market.DepositCollateral("user2", 20)
				market.BorrowQUSD("user2", 1000) // healthy at $100

				market.DepositCollateral("user3", 5)
				market.BorrowQUSD("user3", 200) // healthy at $100

				By("crashing the oracle price to $70")
				market.UpdateOraclePrice(70)
				// user1: $700 collateral, $600 debt → CR 116% → underwater
				// user2: $1400 collateral, $1000 debt → CR 140% → underwater
				// user3: $350 collateral, $200 debt → CR 175% → healthy

				By("liquidating underwater positions")
				Expect(market.Liquidate("user1")).To(BeTrue(), "user1 should be underwater")
				Expect(market.Liquidate("user2")).To(BeTrue(), "user2 should be underwater")
				Expect(market.Liquidate("user3")).To(BeFalse(), "user3 should still be healthy")
			})
		})
	})

	// ── Oracle Price Update ────────────────────────────────────────────────

	Describe("UpdateOraclePrice", func() {
		It("should affect collateral value calculations", func() {
			market.DepositCollateral("dave", 10)   // $1000 at $100
			Expect(market.BorrowQUSD("dave", 600)).To(BeTrue())

			// Halve the price: $500 collateral, $600 debt → underwater
			market.UpdateOraclePrice(50)
			Expect(market.Liquidate("dave")).To(BeTrue())
		})
	})
})
