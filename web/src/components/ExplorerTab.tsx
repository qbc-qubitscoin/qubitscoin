import React, { useState } from 'react'
import type { QBCClient } from '../services/rpcClient'
import type { BlockInfo } from '../types/rpc'

export interface ExplorerTabProps {
  client: QBCClient
}

export const ExplorerTab: React.FC<ExplorerTabProps> = ({ client }) => {
  const [query, setQuery] = useState('')
  const [block, setBlock] = useState<BlockInfo | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSearch = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    const trimmed = query.trim()
    if (!trimmed) return

    setLoading(true)
    setError(null)
    setBlock(null)

    try {
      if (/^\d+$/.test(trimmed)) {
        const height = parseInt(trimmed, 10)
        const b = await client.getBlockByHeight(height)
        setBlock(b)
      } else {
        const b = await client.getBlockByHash(trimmed)
        setBlock(b)
      }
    } catch (err: any) {
      setError(err.message || 'Block not found')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="tab-pane">
      <div className="tab-header">
        <div>
          <h2 className="tab-title">Block & Transaction Explorer</h2>
          <p className="tab-subtitle">Inspect blocks, post-quantum transactions, and state roots</p>
        </div>
      </div>

      <form onSubmit={handleSearch} className="search-bar-form">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Block height or 32-byte hash (e.g., 105 or 0000abcd...)"
          className="search-input"
          aria-label="Search query"
        />
        <button
          type="submit"
          className="btn primary"
          aria-label="Search"
          disabled={loading || !query.trim()}
        >
          {loading ? 'Searching...' : '🔍 Search'}
        </button>
      </form>

      {error && (
        <div className="alert-box error mt-4" role="alert">
          <span>⚠️ {error}</span>
        </div>
      )}

      {loading && (
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Fetching block from blockchain...</p>
        </div>
      )}

      {block && (
        <div className="card mt-4">
          <div className="card-header-flex">
            <h3 className="card-title">Block #{block.height} Details</h3>
            <span className="badge success">{block.txCount} Transactions</span>
          </div>

          <div className="detail-table">
            <div className="detail-row">
              <span className="detail-key">Block Hash</span>
              <code className="detail-val break-all">{block.hash}</code>
            </div>
            <div className="detail-row">
              <span className="detail-key">Parent Hash</span>
              <code className="detail-val break-all">{block.parentHash}</code>
            </div>
            <div className="detail-row">
              <span className="detail-key">Proposer (Validator)</span>
              <code className="detail-val text-green break-all">{block.proposer}</code>
            </div>
            <div className="detail-row">
              <span className="detail-key">Timestamp</span>
              <span className="detail-val">{new Date(block.timestamp * 1000).toUTCString()}</span>
            </div>
            <div className="detail-row">
              <span className="detail-key">Gas Used</span>
              <span className="detail-val">{block.gasUsed.toLocaleString()} gas</span>
            </div>
            <div className="detail-row">
              <span className="detail-key">Base Fee</span>
              <span className="detail-val">{(block.baseFee / 1000000).toFixed(4)} nanoQBC</span>
            </div>
          </div>

          <h4 className="sub-title mt-4">Transactions in this Block</h4>
          {block.transactions && block.transactions.length > 0 ? (
            <div className="table-responsive">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Tx Hash</th>
                    <th>Type</th>
                    <th>From</th>
                    <th>To</th>
                    <th>Value (QBC)</th>
                    <th>Nonce</th>
                    <th>Gas Limit</th>
                  </tr>
                </thead>
                <tbody>
                  {block.transactions.map((tx) => (
                    <tr key={tx.hash}>
                      <td><code className="tx-hash-badge">{tx.hash.slice(0, 12)}...</code></td>
                      <td><span className="type-badge">{tx.type}</span></td>
                      <td><code className="addr-badge">{tx.from.slice(0, 10)}...</code></td>
                      <td><code className="addr-badge">{tx.to ? `${tx.to.slice(0, 10)}...` : 'Contract Deploy'}</code></td>
                      <td className="text-right">
                        {(Number(BigInt(tx.value || '0')) / 1e18).toFixed(4)}
                      </td>
                      <td>{tx.nonce}</td>
                      <td>{tx.gasLimit.toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="empty-text">No transactions included in this block.</p>
          )}
        </div>
      )}
    </section>
  )
}
