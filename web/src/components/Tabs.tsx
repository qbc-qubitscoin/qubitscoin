import React from 'react'

export type TabKey =
  | 'dashboard'
  | 'explorer'
  | 'wallet'
  | 'dex'
  | 'greendao'
  | 'carbonx'
  | 'rpc-console'

export interface TabsProps {
  activeTab: TabKey
  onSelectTab: (tab: TabKey) => void
}

interface TabDefinition {
  key: TabKey
  label: string
  icon: string
}

const TABS: TabDefinition[] = [
  { key: 'dashboard', label: 'Dashboard', icon: '📊' },
  { key: 'explorer', label: 'Explorer', icon: '🔍' },
  { key: 'wallet', label: 'Quantum Wallet', icon: '🔑' },
  { key: 'dex', label: 'QubitSwap', icon: '🔄' },
  { key: 'greendao', label: 'GreenDAO', icon: '🌱' },
  { key: 'carbonx', label: 'CarbonX', icon: '🌍' },
  { key: 'rpc-console', label: 'RPC Console', icon: '⚡' },
]

export const Tabs: React.FC<TabsProps> = ({ activeTab, onSelectTab }) => {
  return (
    <nav className="tabs-nav" role="tablist" aria-label="Main Navigation">
      {TABS.map((tab) => {
        const isActive = activeTab === tab.key
        return (
          <button
            key={tab.key}
            role="tab"
            aria-selected={isActive}
            className={`tab-btn ${isActive ? 'active' : ''}`}
            onClick={() => onSelectTab(tab.key)}
          >
            <span className="tab-icon">{tab.icon}</span>
            <span className="tab-label">{tab.label}</span>
          </button>
        )
      })}
    </nav>
  )
}
