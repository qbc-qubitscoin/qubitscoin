import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { Header } from '../components/Header'
import { Tabs } from '../components/Tabs'
import { DashboardTab } from '../components/DashboardTab'
import { ExplorerTab } from '../components/ExplorerTab'
import { WalletTab } from '../components/WalletTab'
import { DexTab } from '../components/DexTab'
import { GreenDaoTab } from '../components/GreenDaoTab'
import { CarbonXTab } from '../components/CarbonXTab'
import { RpcConsoleTab } from '../components/RpcConsoleTab'
import type { QBCClient } from '../services/rpcClient'
import type { ChainInfo, BlockInfo, FeeEstimate } from '../types/rpc'

// Mock client factory
const createMockClient = (overrides = {}): QBCClient => {
  const mockInfo: ChainInfo = {
    chainId: 'qubitscoin-mainnet',
    height: 105,
    tipHash: 'a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90',
    validators: 4,
    round: 1,
    mempoolSize: 2,
    baseFee: 1000000,
    gasLimit: 30000000,
    gasTargetRatio: 0.5,
  }

  const mockBlock: BlockInfo = {
    height: 105,
    hash: '0000abcd12345678901234567890123456789012345678901234567890abcdef',
    parentHash: '0000000000000000000000000000000000000000000000000000000000000000',
    timestamp: 1725890000,
    proposer: 'QBC1A2B3C4D5E6F7890123456789012345678901',
    txCount: 1,
    transactions: [
      {
        hash: 'txhash123',
        type: 'transfer',
        from: 'QBC1A2B3C4D5E6F7890123456789012345678901',
        to: 'QBC9Z8Y7X6W5V4U3210987654321098765432109',
        value: '5000000000000000000',
        nonce: 1,
        gasLimit: 21000,
        maxFeePerGas: 1000000,
      },
    ],
    gasUsed: 21000,
    baseFee: 1000000,
  }

  const mockFee: FeeEstimate = {
    baseFee: 1000000,
    slow: 1000000,
    standard: 1200000,
    fast: 1500000,
  }

  return {
    getEndpoint: vi.fn().mockReturnValue('http://127.0.0.1:8545'),
    setEndpoint: vi.fn(),
    getChainInfo: vi.fn().mockResolvedValue(mockInfo),
    getBlockByHeight: vi.fn().mockResolvedValue(mockBlock),
    getBlockByHash: vi.fn().mockResolvedValue(mockBlock),
    getBalance: vi.fn().mockResolvedValue('50000000000000000000'),
    getTransactionCount: vi.fn().mockResolvedValue(3),
    sendRawTransaction: vi.fn().mockResolvedValue('0xcreatedtxhash123'),
    getFeeEstimate: vi.fn().mockResolvedValue(mockFee),
    getGasPrice: vi.fn().mockResolvedValue(1000000),
    dispatch: vi.fn().mockResolvedValue({ status: 'ok' }),
    dispatchBatch: vi.fn().mockResolvedValue([]),
    ...overrides,
  } as unknown as QBCClient
}

describe('React Component Suite (TDD)', () => {
  it('Header renders branding and theme toggle', () => {
    const onToggleTheme = vi.fn()
    render(
      <Header
        endpoint="http://127.0.0.1:8545"
        connected={true}
        height={105}
        isDark={true}
        onToggleTheme={onToggleTheme}
        onEndpointChange={vi.fn()}
      />
    )

    expect(screen.getByText(/QubitsCoin/i)).toBeInTheDocument()
    expect(screen.getByText(/Block #105/i)).toBeInTheDocument()
    expect(screen.getByText(/Online/i)).toBeInTheDocument()

    const themeBtn = screen.getByRole('button', { name: /toggle theme/i })
    fireEvent.click(themeBtn)
    expect(onToggleTheme).toHaveBeenCalled()
  })

  it('Tabs component renders all navigation options and handles switching', () => {
    const onSelect = vi.fn()
    render(<Tabs activeTab="dashboard" onSelectTab={onSelect} />)

    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Explorer')).toBeInTheDocument()
    expect(screen.getByText('Quantum Wallet')).toBeInTheDocument()
    expect(screen.getByText('QubitSwap')).toBeInTheDocument()
    expect(screen.getByText('GreenDAO')).toBeInTheDocument()
    expect(screen.getByText('CarbonX')).toBeInTheDocument()
    expect(screen.getByText('RPC Console')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Explorer'))
    expect(onSelect).toHaveBeenCalledWith('explorer')
  })

  it('DashboardTab displays chain metrics and refreshes', async () => {
    const client = createMockClient()
    render(<DashboardTab client={client} />)

    expect(screen.getByText(/Loading node status/i)).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText('qubitscoin-mainnet')).toBeInTheDocument()
    })

    expect(screen.getByText('105')).toBeInTheDocument()
    expect(screen.getByText('4')).toBeInTheDocument()

    const refreshBtn = screen.getByRole('button', { name: /refresh/i })
    await act(async () => {
      fireEvent.click(refreshBtn)
    })
    expect(client.getChainInfo).toHaveBeenCalledTimes(2)
  })

  it('ExplorerTab searches and displays block details', async () => {
    const client = createMockClient()
    render(<ExplorerTab client={client} />)

    const searchInput = screen.getByPlaceholderText(/Block height or 32-byte hash/i)
    fireEvent.change(searchInput, { target: { value: '105' } })

    const searchBtn = screen.getByRole('button', { name: /search/i })
    fireEvent.click(searchBtn)

    await waitFor(() => {
      expect(screen.getByText(/Block #105 Details/i)).toBeInTheDocument()
    })

    expect(screen.getByText(/QBC1A2B3C4D5E6F7890123456789012345678901/i)).toBeInTheDocument()
    expect(screen.getByText(/txhash123/i)).toBeInTheDocument()
  })

  it('WalletTab generates post-quantum keypair and fetches balance', async () => {
    const client = createMockClient()
    render(<WalletTab client={client} />)

    const genBtn = screen.getByRole('button', { name: /generate post-quantum keypair/i })
    fireEvent.click(genBtn)

    expect(screen.getByText(/Generated Post-Quantum Address/i)).toBeInTheDocument()

    const checkBalBtn = screen.getByRole('button', { name: /check balance/i })
    fireEvent.click(checkBalBtn)

    await waitFor(() => {
      expect(screen.getByText(/50.0000 QBC/i)).toBeInTheDocument()
    })
  })

  it('DexTab calculates AMM swap and output amount', () => {
    render(<DexTab />)

    const inputAmount = screen.getByLabelText(/You Pay/i)
    fireEvent.change(inputAmount, { target: { value: '10' } })

    expect(screen.getByText(/Estimated Output/i)).toBeInTheDocument()
    expect(screen.getByText(/0.3% Fee/i)).toBeInTheDocument()
  })

  it('GreenDaoTab lists proposals and registers vote', () => {
    render(<GreenDaoTab />)

    expect(screen.getByText(/Renewable Validator Subsidy/i)).toBeInTheDocument()
    const voteBtn = screen.getAllByRole('button', { name: /Vote For/i })[0]
    fireEvent.click(voteBtn)

    expect(screen.getByText(/Vote Recorded/i)).toBeInTheDocument()
  })

  it('CarbonXTab displays ESG credits and allows retirement', () => {
    render(<CarbonXTab />)

    expect(screen.getByText(/Boreal Forest Conservation/i)).toBeInTheDocument()
    const retireBtn = screen.getAllByRole('button', { name: /Retire/i })[0]
    fireEvent.click(retireBtn)

    expect(screen.getByText('Retired')).toBeInTheDocument()
  })

  it('RpcConsoleTab executes custom RPC method and displays result', async () => {
    const client = createMockClient({
      dispatch: vi.fn().mockResolvedValue({ height: 999, status: 'synced' }),
    })
    render(<RpcConsoleTab client={client} />)

    const executeBtn = screen.getByRole('button', { name: /Execute RPC Call/i })
    fireEvent.click(executeBtn)

    await waitFor(() => {
      expect(screen.getByText(/"status": "synced"/i)).toBeInTheDocument()
    })
  })
})
