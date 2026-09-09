import React, { useState } from 'react'

export interface HeaderProps {
  endpoint: string
  connected: boolean
  height: number
  isDark: boolean
  onToggleTheme: () => void
  onEndpointChange: (url: string) => void
}

export const Header: React.FC<HeaderProps> = ({
  endpoint,
  connected,
  height,
  isDark,
  onToggleTheme,
  onEndpointChange,
}) => {
  const [editingUrl, setEditingUrl] = useState(false)
  const [tempUrl, setTempUrl] = useState(endpoint)

  const handleSaveEndpoint = () => {
    onEndpointChange(tempUrl)
    setEditingUrl(false)
  }

  return (
    <header className="header-container">
      <div className="brand-section">
        <div className="brand-logo">⚛</div>
        <div>
          <h1 className="brand-title">QubitsCoin (QBC)</h1>
          <span className="brand-badge">NIST ML-DSA-65 • Mainnet</span>
        </div>
      </div>

      <div className="status-section">
        <div className="metric-pill">
          <span className="pill-dot"></span>
          <span>Block #{height}</span>
        </div>

        <div className={`status-pill ${connected ? 'online' : 'offline'}`}>
          <span className="status-indicator"></span>
          <span>{connected ? 'Online' : 'Offline'}</span>
        </div>

        {editingUrl ? (
          <div className="endpoint-edit">
            <input
              type="text"
              value={tempUrl}
              onChange={(e) => setTempUrl(e.target.value)}
              className="endpoint-input"
              aria-label="RPC Endpoint"
            />
            <button onClick={handleSaveEndpoint} className="btn-sm">Save</button>
            <button onClick={() => setEditingUrl(false)} className="btn-sm secondary">Cancel</button>
          </div>
        ) : (
          <button
            onClick={() => {
              setTempUrl(endpoint)
              setEditingUrl(true)
            }}
            className="endpoint-badge"
            title="Click to edit RPC Endpoint"
          >
            {endpoint}
          </button>
        )}

        <button
          onClick={onToggleTheme}
          className="theme-toggle-btn"
          aria-label="Toggle theme"
          title={`Switch to ${isDark ? 'Light' : 'Dark'} Mode`}
        >
          {isDark ? '☀️' : '🌙'}
        </button>
      </div>
    </header>
  )
}
