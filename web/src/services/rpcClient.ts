import type {
  RPCRequest,
  RPCResponse,
  ChainInfo,
  BlockInfo,
  FeeEstimate,
} from '../types/rpc'

export class RPCError extends Error {
  code: number
  data?: any

  constructor(message: string, code: number, data?: any) {
    super(message)
    this.name = 'RPCError'
    this.code = code
    this.data = data
  }
}

export class QBCClient {
  private endpoint: string
  private idCounter: number = 0

  constructor(endpoint: string = 'http://localhost:8545') {
    this.endpoint = endpoint
  }

  getEndpoint(): string {
    return this.endpoint
  }

  setEndpoint(url: string): void {
    this.endpoint = url
  }

  private nextId(): number {
    return ++this.idCounter
  }

  async dispatch<T = any>(method: string, params: any[] = []): Promise<T> {
    const id = this.nextId()
    const reqBody: RPCRequest = {
      jsonrpc: '2.0',
      method,
      params,
      id,
    }

    const response = await fetch(this.endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(reqBody),
    })

    if (!response.ok) {
      throw new Error(`HTTP Error ${response.status}: ${response.statusText}`)
    }

    const json: RPCResponse<T> = await response.json()

    if (json.error) {
      throw new RPCError(json.error.message, json.error.code, json.error.data)
    }

    return json.result as T
  }

  async dispatchBatch(calls: { method: string; params: any[] }[]): Promise<any[]> {
    if (calls.length === 0) return []

    const reqBatch: RPCRequest[] = calls.map((c) => ({
      jsonrpc: '2.0',
      method: c.method,
      params: c.params,
      id: this.nextId(),
    }))

    const response = await fetch(this.endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(reqBatch),
    })

    if (!response.ok) {
      throw new Error(`HTTP Error ${response.status}: ${response.statusText}`)
    }

    const jsonList: RPCResponse[] = await response.json()
    return jsonList.map((item) => {
      if (item.error) {
        throw new RPCError(item.error.message, item.error.code, item.error.data)
      }
      return item.result
    })
  }

  async getChainInfo(): Promise<ChainInfo> {
    return this.dispatch<ChainInfo>('qbc_chainInfo', [])
  }

  async getBlockByHeight(height: number): Promise<BlockInfo> {
    return this.dispatch<BlockInfo>('qbc_blockByHeight', [height])
  }

  async getBlockByHash(hash: string): Promise<BlockInfo> {
    return this.dispatch<BlockInfo>('qbc_blockByHash', [hash])
  }

  async getBalance(address: string): Promise<string> {
    return this.dispatch<string>('qbc_getBalance', [address])
  }

  async getTransactionCount(address: string): Promise<number> {
    return this.dispatch<number>('qbc_getTransactionCount', [address])
  }

  async sendRawTransaction(rawHex: string): Promise<string> {
    return this.dispatch<string>('qbc_sendRawTransaction', [rawHex])
  }

  async getFeeEstimate(): Promise<FeeEstimate> {
    return this.dispatch<FeeEstimate>('qbc_feeEstimate', [])
  }

  async getGasPrice(): Promise<number> {
    return this.dispatch<number>('qbc_gasPrice', [])
  }
}

export const defaultClient = new QBCClient()
