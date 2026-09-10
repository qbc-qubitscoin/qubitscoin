import React, { useState } from 'react'
import type { CarbonCredit } from '../types/rpc'

const INITIAL_CREDITS: CarbonCredit[] = [
  {
    id: 'CRB-2026-NOR-001',
    vintage: 2026,
    project: 'Boreal Forest Conservation',
    standard: 'Verra VCS-941',
    issuer: 'Nordic Forestry Alliance',
    owner: 'QBC1A2B3C4D5E6F7890123456789012345678901',
    amountTons: 1500,
    isRetired: false,
  },
  {
    id: 'CRB-2025-WND-042',
    vintage: 2025,
    project: 'Offshore Wind Generation',
    standard: 'Gold Standard GS-330',
    issuer: 'Global Clean Energy Trust',
    owner: 'QBC9Z8Y7X6W5V4U3210987654321098765432109',
    amountTons: 3200,
    isRetired: false,
  },
]

export const CarbonXTab: React.FC = () => {
  const [credits, setCredits] = useState<CarbonCredit[]>(INITIAL_CREDITS)
  const [retiredMsg, setRetiredMsg] = useState<string | null>(null)

  const handleRetire = (id: string) => {
    setCredits((prev) =>
      prev.map((c) => (c.id === id ? { ...c, isRetired: true } : c))
    )
    setRetiredMsg(`Credit ${id} has been permanently retired on-chain.`)
    setTimeout(() => setRetiredMsg(null), 4000)
  }

  return (
    <section className="tab-pane">
      <div className="tab-header">
        <div>
          <h2 className="tab-title">CarbonX ESG Credit Registry</h2>
          <p className="tab-subtitle">Verifiable on-chain carbon credits and environmental offset retirement</p>
        </div>
      </div>

      {retiredMsg && (
        <div className="alert-box success mb-4">
          <span>✅ {retiredMsg}</span>
        </div>
      )}

      <div className="credits-grid">
        {credits.map((c) => (
          <div key={c.id} className="card credit-card">
            <div className="card-header-flex">
              <span className="badge info">{c.standard}</span>
              <span className={`badge ${c.isRetired ? 'secondary' : 'success'}`}>
                {c.isRetired ? 'Retired' : 'Active'}
              </span>
            </div>

            <h3 className="credit-title">{c.project}</h3>
            <div className="info-row">
              <span className="info-key">Certificate ID:</span>
              <code>{c.id}</code>
            </div>
            <div className="info-row">
              <span className="info-key">Vintage:</span>
              <span>{c.vintage}</span>
            </div>
            <div className="info-row">
              <span className="info-key">Issuer:</span>
              <span>{c.issuer}</span>
            </div>
            <div className="info-row">
              <span className="info-key">Offset Volume:</span>
              <span className="highlight text-lg">{c.amountTons.toLocaleString()} tCO2e</span>
            </div>

            <div className="mt-4">
              {c.isRetired ? (
                <button className="btn secondary full-width" disabled>
                  🛡️ Certificate Retired Permanently
                </button>
              ) : (
                <button
                  onClick={() => handleRetire(c.id)}
                  className="btn primary full-width"
                  aria-label="Retire"
                >
                  🌱 Retire Certificate
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
