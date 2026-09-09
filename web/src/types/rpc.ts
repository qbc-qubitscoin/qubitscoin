export interface RPCRequest<T = any[]> {
  jsonrpc: '2.0'
  method: string
  params: T
  id: number
}

export interface RPCResponse<T = any> {
  jsonrpc: '2.0'
  id: number
  result?: T
  error?: {
    code: number
    message: string
    data?: any
  }
}

export interface ChainInfo {
  chainId: string
  height: number
  tipHash: string
  validators: number
  round: number
  mempoolSize: number
  baseFee: number
  gasLimit: number
  gasTargetRatio: number
}

export interface TransactionInfo {
  hash: string
  type: string
  from: string
  to: string
  value: string
  nonce: number
  gasLimit: number
  maxFeePerGas: number
}

export interface BlockInfo {
  height: number
  hash: string
  parentHash: string
  timestamp: number
  proposer: string
  txCount: number
  transactions: TransactionInfo[]
  gasUsed: number
  baseFee: number
}

export interface FeeEstimate {
  baseFee: number
  slow: number
  standard: number
  fast: number
}

export interface Proposal {
  id: number
  title: string
  description: string
  proposer: string
  votesFor: number
  votesAgainst: number
  quorum: number
  status: 'Active' | 'Passed' | 'Rejected' | 'Executed'
  deadline: number
}

export interface CarbonCredit {
  id: string
  vintage: number
  project: string
  standard: string
  issuer: string
  owner: string
  amountTons: number
  isRetired: boolean
}
