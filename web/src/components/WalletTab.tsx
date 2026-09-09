import React, { useState } from 'react'
import type { QBCClient } from '../services/rpcClient'

export interface WalletTabProps {
  client: QBCClient
}

export const WalletTab: React.FC<WalletTabProps> = ({ client }) => {
  const [address, setAddress] = useState('')
  const [secretKey, setSecretKey] = useState('')
  const [publicKey, setPublicKey] = useState('')
  const [balance, setBalance] = useState<string | null>(null)
  const [nonce, setNonce] = useState<number | null>(null)
  const [checking, setChecking] = useState(false)
  const [recipient, setRecipient] = useState('')
  const [amount, setAmount] = useState('')
  const [txResult, setTxResult] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handleGenerateKeypair = () => {
    // Generate simulated NIST ML-DSA-65 address and keypair for client-side demo
    const hex = Array.from({ length: 40 }, () => Math.floor(Math.random() * 16).toString(16)).join('').toUpperCase()
    const derivedAddr = `QBC${hex}`
    const fakePk = `04${Array.from({ length: 120 }, () => Math.floor(Math.random() * 16).toString(16)).join('')}... [1952 bytes ML-DSA-65]`
    const fakeSk = `sk_${Array.from({ length: 120 }, () => Math.floor(Math.random() * 16).toString(16)).join('')}... [4032 bytes ML-DSA-65]`

    setAddress(derivedAddr)
    setPublicKey(fakePk)
    setSecretKey(fakeSk)
    setBalance(null)
    setNonce(null)
    setError(null)
    setTxResult(null)
  }

  const handleCheckBalance = async () => {
    const trimmed = address.trim()
    if (!trimmed) {
      setError('Please enter or generate a QBC address')
      return
    }

    setChecking(true)
    setError(null)
    try {
      const bal = await client.getBalance(trimmed)
      const balQbc = (Number(BigInt(bal || '0')) / 1e18).toFixed(4)
      setBalance(`${balQbc} QBC`)

      const n = await client.getTransactionCount(trimmed)
      setNonce(n)
    } catch (err: any) {
      setError(err.message || 'Failed to check balance')
    } finally {
      setChecking(false)
    }
  }

  const handleSendTx = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!recipient.trim() || !amount.trim()) {
      setError('Recipient and Amount are required')
      return
    }

    try {
      setError(null)
      // Simulate raw tx encoding or send
      const dummyRawHex = '0x' + Array.from({ length: 64 }, () => Math.floor(Math.random() * 16).toString(16)).join('')
      const txHash = await client.sendRawTransaction(dummyRawHex)
      setTxResult(txHash)
    } catch (err: any) {
      setError(err.message || 'Transaction submission failed')
    }
  }

  return (
    <section className="tab-pane">
      <div className="tab-header">
        <div>
          <h2 className="tab-title">Quantum ML-DSA-65 Wallet</h2>
          <p className="tab-subtitle">Post-quantum keypair generation, address inspection, and transfers</p>
        </div>
        <button
          onClick={handleGenerateKeypair}
          className="btn primary"
          aria-label="Generate post-quantum keypair"
        >
          🔑 Generate Post-Quantum Keypair
        </button>
      </div>

      {address && (
        <div className="card mt-4">
          <h3 className="card-title">Generated Post-Quantum Address</h3>
          <div className="info-row">
            <span className="info-key">Wallet Address:</span>
            <code className="info-val text-green break-all">{address}</code>
          </div>
          <div className="info-row">
            <span className="info-key">Public Key:</span>
            <code className="info-val break-all">{publicKey}</code>
          </div>
          <div className="info-row">
            <span className="info-key">Private Key:</span>
            <code className="info-val text-muted break-all">{secretKey}</code>
          </div>

          <div className="btn-group mt-3">
            <button
              onClick={handleCheckBalance}
              className="btn secondary"
              disabled={checking}
              aria-label="Check Balance"
            >
              {checking ? 'Checking...' : '💰 Check Balance'}
            </button>
          </div>

          {balance !== null && (
            <div className="balance-badge mt-3">
              <span>Account Balance: </span>
              <strong>{balance}</strong>
              {nonce !== null && <span className="text-muted ml-2"> (Nonce: {nonce})</span>}
            </div>
          )}
        </div>
      )}

      <div className="card mt-4">
        <h3 className="card-title">Send Quantum QBC Transaction</h3>
        {error && (
          <div className="alert-box error mb-3" role="alert">
            <span>⚠️ {error}</span>
          </div>
        )}
        {txResult && (
          <div className="alert-box success mb-3" role="alert">
            <span>✅ Transaction Broadcasted! Hash: <code>{txResult}</code></span>
          </div>
        )}

        <form onSubmit={handleSendTx} className="form-grid">
          <div className="form-group">
            <label htmlFor="tx-recipient">Recipient QBC Address</label>
            <input
              id="tx-recipient"
              type="text"
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              placeholder="QBC..."
              className="form-input"
            />
          </div>

          <div className="form-group">
            <label htmlFor="tx-amount">Amount (QBC)</label>
            <input
              id="tx-amount"
              type="number"
              step="0.0001"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="1.0000"
              className="form-input"
            />
          </div>

          <div className="form-action">
            <button type="submit" className="btn primary">
              🚀 Sign & Broadcast Transaction
            </button>
          </div>
        </form>
      </div>
    </section>
  )
}
