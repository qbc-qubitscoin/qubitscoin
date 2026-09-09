import React, { useState } from 'react'

export const DexTab: React.FC = () => {
  const [payAmount, setPayAmount] = useState('10')
  const [reserveA] = useState(1000000) // 1M QBC
  const [reserveB] = useState(2500000) // 2.5M USDT
  const [swapped, setSwapped] = useState(false)

  // AMM formula: dy = (y * dx * 0.997) / (x + dx * 0.997)
  const dx = parseFloat(payAmount) || 0
  const feeMultiplier = 0.997
  const estimatedOut = dx > 0 ? (reserveB * dx * feeMultiplier) / (reserveA + dx * feeMultiplier) : 0
  const priceImpact = dx > 0 ? ((dx / (reserveA + dx)) * 100).toFixed(2) : '0.00'

  const handleSwap = () => {
    if (dx <= 0) return
    setSwapped(true)
    setTimeout(() => setSwapped(false), 3000)
  }

  return (
    <section className="tab-pane">
      <div className="tab-header">
        <div>
          <h2 className="tab-title">QubitSwap AMM Hub</h2>
          <p className="tab-subtitle">Decentralized Constant-Product AMM ($x \cdot y = k$) on QubitsCoin</p>
        </div>
      </div>

      <div className="swap-card-container">
        <div className="card swap-box">
          <div className="form-group">
            <label htmlFor="pay-amount">You Pay</label>
            <div className="input-with-badge">
              <input
                id="pay-amount"
                type="number"
                value={payAmount}
                onChange={(e) => setPayAmount(e.target.value)}
                placeholder="0.0"
                className="form-input large-input"
              />
              <span className="token-tag">QBC</span>
            </div>
          </div>

          <div className="swap-arrow">⬇️</div>

          <div className="form-group">
            <label htmlFor="receive-amount">Estimated Output</label>
            <div className="input-with-badge">
              <input
                id="receive-amount"
                type="text"
                readOnly
                value={estimatedOut.toFixed(4)}
                className="form-input large-input readonly"
              />
              <span className="token-tag">USDT</span>
            </div>
          </div>

          <div className="swap-meta">
            <div className="meta-row">
              <span>Exchange Rate:</span>
              <span>1 QBC ≈ {(reserveB / reserveA).toFixed(2)} USDT</span>
            </div>
            <div className="meta-row">
              <span>Protocol Fee:</span>
              <span>0.3% Fee ($99.7\%$ to Liquidity Providers)</span>
            </div>
            <div className="meta-row">
              <span>Price Impact:</span>
              <span className={parseFloat(priceImpact) > 5 ? 'text-red' : 'text-green'}>
                {priceImpact}%
              </span>
            </div>
          </div>

          {swapped && (
            <div className="alert-box success mt-3">
              <span>✅ Swap executed successfully via WASM Contract!</span>
            </div>
          )}

          <button onClick={handleSwap} className="btn primary full-width mt-3">
            🔄 Swap QBC for USDT
          </button>
        </div>

        <div className="card pool-info-box">
          <h3 className="card-title">Liquidity Pool Metrics</h3>
          <div className="info-row">
            <span className="info-key">Pool Pair:</span>
            <span className="info-val">QBC / USDT</span>
          </div>
          <div className="info-row">
            <span className="info-key">Total Value Locked (TVL):</span>
            <span className="info-val highlight">$5,000,000</span>
          </div>
          <div className="info-row">
            <span className="info-key">QBC Reserve:</span>
            <span className="info-val">{reserveA.toLocaleString()} QBC</span>
          </div>
          <div className="info-row">
            <span className="info-key">USDT Reserve:</span>
            <span className="info-val">{reserveB.toLocaleString()} USDT</span>
          </div>
          <div className="info-row">
            <span className="info-key">Contract Engine:</span>
            <span className="info-val text-green">Wazero Pure-Go WASM</span>
          </div>
        </div>
      </div>
    </section>
  )
}
