import React, { useState, useEffect, useCallback } from 'react'
import type { QBCClient } from '../services/rpcClient'
import type { ChainInfo } from '../types/rpc'

export interface DashboardTabProps {
  client: QBCClient
}

export const DashboardTab: React.FC<DashboardTabProps> = ({ client }) => {
  const [chainInfo, setChainInfo] = useState<ChainInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStatus = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const info = await client.getChainInfo()
      setChainInfo(info)
    } catch (err: any) {
      setError(err.message || 'Failed to connect to QBC Node')
    } finally {
      setLoading(false)
    }
  }, [client])

  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])

  return (
    <section className="tab-pane">
      <div className="tab-header">
        <div>
          <h2 className="tab-title">Live Network Dashboard</h2>
          <p className="tab-subtitle">Real-time status of the quantum-resistant QubitsCoin consensus network</p>
        </div>
        <button
          onClick={fetchStatus}
          className="btn primary"
          aria-label="Refresh"
          disabled={loading}
        >
          {loading ? 'Refreshing...' : '🔄 Refresh'}
        </button>
      </div>

      {error && (
        <div className="alert-box error" role="alert">
          <span>⚠️ {error}</span>
          <button onClick={fetchStatus} className="btn-sm">Retry</button>
        </div>
      )}

      {loading && !chainInfo && (
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Loading node status...</p>
        </div>
      )}

      {chainInfo && (
        <>
          <div className="metrics-grid">
            <div className="metric-card">
              <span className="metric-label">Network ID</span>
              <span className="metric-value highlight">{chainInfo.chainId}</span>
              <span className="metric-foot">NIST ML-DSA-65 Validated</span>
            </div>

            <div className="metric-card">
              <span className="metric-label">Block Height</span>
              <span className="metric-value">{chainInfo.height}</span>
              <span className="metric-foot">2-second BFT slot</span>
            </div>

            <div className="metric-card">
              <span className="metric-label">Active Validators</span>
              <span className="metric-value">{chainInfo.validators}</span>
              <span className="metric-foot">Round {chainInfo.round} Proposer</span>
            </div>

            <div className="metric-card">
              <span className="metric-label">Mempool Size</span>
              <span className="metric-value">{chainInfo.mempoolSize} txs</span>
              <span className="metric-foot">Priority Fee Heap</span>
            </div>

            <div className="metric-card">
              <span className="metric-label">Base Fee (EIP-1559)</span>
              <span className="metric-value">{(chainInfo.baseFee / 1000000).toFixed(2)} nanoQBC</span>
              <span className="metric-foot">Dynamic Gas Elasticity</span>
            </div>

            <div className="metric-card">
              <span className="metric-label">Gas Target Ratio</span>
              <span className="metric-value">{(chainInfo.gasTargetRatio * 100).toFixed(1)}%</span>
              <span className="metric-foot">Target: 50% / Block</span>
            </div>
          </div>

          <div className="card mt-4">
            <h3 className="card-title">Consensus State & Best Tip</h3>
            <div className="info-row">
              <span className="info-key">Tip Block Hash:</span>
              <code className="info-val break-all">{chainInfo.tipHash}</code>
            </div>
            <div className="info-row">
              <span className="info-key">Block Gas Limit:</span>
              <span className="info-val">{chainInfo.gasLimit.toLocaleString()} gas</span>
            </div>
            <div className="info-row">
              <span className="info-key">Signature Scheme:</span>
              <span className="info-val text-green">ML-DSA-65 (NIST FIPS 204 Standard)</span>
            </div>
            <div className="info-row">
              <span className="info-key">Key Encapsulation:</span>
              <span className="info-val text-cyan">ML-KEM-768 (NIST FIPS 203 Standard)</span>
            </div>
          </div>
        </>
      )}
    </section>
  )
}
