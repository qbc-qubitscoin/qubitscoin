package bdd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/qbc-qubitscoin/qubitscoin/internal/core"
	"github.com/qbc-qubitscoin/qubitscoin/internal/crypto"
)

// ── BDD: Transaction Lifecycle ────────────────────────────────────────────────

var _ = Describe("Transaction Lifecycle", func() {
	var (
		senderWallet   *crypto.Wallet
		receiverWallet *crypto.Wallet
	)

	BeforeEach(func() {
		var err error
		senderWallet, err = crypto.NewWallet()
		Expect(err).NotTo(HaveOccurred())
		receiverWallet, err = crypto.NewWallet()
		Expect(err).NotTo(HaveOccurred())
	})

	// ── Signing & Verification ─────────────────────────────────────────────

	Describe("Sign and Verify", func() {
		Context("when a valid transfer is signed with the sender's private key", func() {
			It("should pass verification", func() {
				tx := core.NewTransfer(
					senderWallet.Address,
					receiverWallet.Address,
					senderWallet.PublicKey,
					0, 100*core.OneQBC, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				Expect(tx.Verify()).To(Succeed())
			})
		})

		Context("when the transaction amount is tampered after signing", func() {
			It("should fail verification", func() {
				tx := core.NewTransfer(
					senderWallet.Address,
					receiverWallet.Address,
					senderWallet.PublicKey,
					0, 100*core.OneQBC, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				tx.Amount = 999_999 * core.OneQBC // tamper
				Expect(tx.Verify()).To(HaveOccurred())
			})
		})

		Context("when the wrong public key is used for verification", func() {
			It("should fail because From address won't match", func() {
				tx := core.NewTransfer(
					senderWallet.Address,
					receiverWallet.Address,
					senderWallet.PublicKey,
					0, 1, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				tx.PublicKey = receiverWallet.PublicKey // swap public key
				Expect(tx.Verify()).To(HaveOccurred())
			})
		})
	})

	// ── Hash Determinism ───────────────────────────────────────────────────

	Describe("ComputeHash", func() {
		Context("when called twice on the same signed transaction", func() {
			It("should return identical hashes", func() {
				tx := core.NewTransfer(
					senderWallet.Address, receiverWallet.Address,
					senderWallet.PublicKey, 1, 50*core.OneQBC, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				Expect(tx.ComputeHash()).To(Equal(tx.ComputeHash()))
			})
		})

		Context("when two transactions have different nonces", func() {
			It("should produce different hashes", func() {
				tx1 := core.NewTransfer(senderWallet.Address, receiverWallet.Address, senderWallet.PublicKey, 0, 1, core.MinGasPrice)
				tx2 := core.NewTransfer(senderWallet.Address, receiverWallet.Address, senderWallet.PublicKey, 1, 1, core.MinGasPrice)
				Expect(tx1.Sign(senderWallet.PrivateKey)).To(Succeed())
				Expect(tx2.Sign(senderWallet.PrivateKey)).To(Succeed())
				Expect(tx1.Hash).NotTo(Equal(tx2.Hash))
			})
		})
	})

	// ── Basic Validation ───────────────────────────────────────────────────

	Describe("BasicValidate", func() {
		Context("when a fully valid signed transaction is validated", func() {
			It("should pass without errors", func() {
				tx := core.NewTransfer(
					senderWallet.Address, receiverWallet.Address,
					senderWallet.PublicKey, 0, 1, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				Expect(tx.BasicValidate()).To(Succeed())
			})
		})

		Context("when the version is unsupported", func() {
			It("should fail BasicValidate", func() {
				tx := core.NewTransfer(
					senderWallet.Address, receiverWallet.Address,
					senderWallet.PublicKey, 0, 1, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				tx.Version = 99
				Expect(tx.BasicValidate()).To(HaveOccurred())
			})
		})

		Context("when gas price is below the minimum", func() {
			It("should fail BasicValidate", func() {
				tx := core.NewTransfer(
					senderWallet.Address, receiverWallet.Address,
					senderWallet.PublicKey, 0, 1, core.MinGasPrice,
				)
				Expect(tx.Sign(senderWallet.PrivateKey)).To(Succeed())
				tx.GasPrice = core.MinGasPrice - 1
				Expect(tx.BasicValidate()).To(HaveOccurred())
			})
		})
	})

	// ── NewTransfer constructor ────────────────────────────────────────────

	Describe("NewTransfer", func() {
		It("should set Version=1, Type=TxTransfer, and preserve all fields", func() {
			tx := core.NewTransfer(
				senderWallet.Address, receiverWallet.Address,
				senderWallet.PublicKey, 7, 250*core.OneQBC, core.MinGasPrice,
			)
			Expect(tx.Version).To(Equal(uint8(1)))
			Expect(tx.Type).To(Equal(core.TxTransfer))
			Expect(tx.Nonce).To(Equal(uint64(7)))
			Expect(tx.Amount).To(Equal(uint64(250 * core.OneQBC)))
			Expect(tx.GasLimit).To(Equal(core.GasTransfer))
			Expect(tx.GasPrice).To(Equal(core.MinGasPrice))
			Expect(tx.From).To(Equal(senderWallet.Address))
			Expect(tx.To).To(Equal(receiverWallet.Address))
		})
	})
})
