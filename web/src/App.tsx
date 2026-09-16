import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { Header } from './components/Header'
import { Tabs, TabKey } from './components/Tabs'
import { DashboardTab } from './components/DashboardTab'
import { ExplorerTab } from './components/ExplorerTab'
import { WalletTab } from './components/WalletTab'
import { DexTab } from './components/DexTab'
import { GreenDaoTab } from './components/GreenDaoTab'
import { CarbonXTab } from './components/CarbonXTab'
import { RpcConsoleTab } from './components/RpcConsoleTab'
import { QBCClient } from './services/rpcClient'
import './App.css'

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabKey>('dashboard')
  const [isDark, setIsDark] = useState<boolean>(() => {
    const saved = localStorage.getItem('qbc_theme')
    if (saved) return saved === 'dark'
    return window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)').matches : true
  })

  // Detect node origin if embedded or fallback to localhost:8545
  const initialEndpoint = useMemo(() => {
    if (typeof window !== 'undefined' && window.location.port && window.location.port !== '5173') {
      return window.location.origin
    }
    return 'http://localhost:8545'
  }, [])

  const [endpoint, setEndpoint] = useState<string>(initialEndpoint)
  const client = useMemo(() => new QBCClient(endpoint), [endpoint])

  const [connected, setConnected] = useState(false)
  const [height, setHeight] = useState(0)

  const checkLiveness = useCallback(async () => {
    try {
      const info = await client.getChainInfo()
      setConnected(true)
      setHeight(info.height)
    } catch {
      setConnected(false)
    }
  }, [client])

  useEffect(() => {
    checkLiveness()
    const interval = setInterval(checkLiveness, 5000)
    return () => clearInterval(interval)
  }, [checkLiveness])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light')
    localStorage.setItem('qbc_theme', isDark ? 'dark' : 'light')
  }, [isDark])

  const toggleTheme = () => setIsDark((prev) => !prev)

  return (
    <div className={`app-root ${isDark ? 'theme-dark' : 'theme-light'}`}>
      <Header
        endpoint={endpoint}
        connected={connected}
        height={height}
        isDark={isDark}
        onToggleTheme={toggleTheme}
        onEndpointChange={setEndpoint}
      />

      <div className="main-layout">
        <Tabs activeTab={activeTab} onSelectTab={setActiveTab} />

        <main className="content-container">
          {activeTab === 'dashboard' && <DashboardTab client={client} />}
          {activeTab === 'explorer' && <ExplorerTab client={client} />}
          {activeTab === 'wallet' && <WalletTab client={client} />}
          {activeTab === 'dex' && <DexTab />}
          {activeTab === 'greendao' && <GreenDaoTab />}
          {activeTab === 'carbonx' && <CarbonXTab />}
          {activeTab === 'rpc-console' && <RpcConsoleTab client={client} />}
        </main>
      </div>

      <footer className="footer-container">
        <div className="footer-left">
          <span>QubitsCoin Core v1.0.0</span>
          <span className="dot-sep">•</span>
          <span>NIST FIPS 204 (ML-DSA-65)</span>
          <span className="dot-sep">•</span>
          <span>NIST FIPS 203 (ML-KEM-768)</span>
        </div>
        <div className="footer-right">
          <span>Pure-Go WASM Runtime • EIP-1559 Elastic Fee</span>
        </div>
      </footer>
    </div>
  )
}

export default App
