import React, { useState } from 'react'
import type { QBCClient } from '../services/rpcClient'

export interface RpcConsoleTabProps {
  client: QBCClient
}

interface MethodTemplate {
  method: string
  params: any[]
}

const TEMPLATES: Record<string, MethodTemplate> = {
  chainInfo: { method: 'qbc_chainInfo', params: [] },
  feeEstimate: { method: 'qbc_feeEstimate', params: [] },
  gasPrice: { method: 'qbc_gasPrice', params: [] },
  blockByHeight: { method: 'qbc_blockByHeight', params: [1] },
  getBalance: { method: 'qbc_getBalance', params: ['QBC0000000000000000000000000000000000000000'] },
}

export const RpcConsoleTab: React.FC<RpcConsoleTabProps> = ({ client }) => {
  const [selectedTemplate, setSelectedTemplate] = useState('chainInfo')
  const [method, setMethod] = useState('qbc_chainInfo')
  const [paramsJson, setParamsJson] = useState('[]')
  const [responseOutput, setResponseOutput] = useState<string | null>(null)
  const [executing, setExecuting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleTemplateSelect = (key: string) => {
    setSelectedTemplate(key)
    const tpl = TEMPLATES[key]
    if (tpl) {
      setMethod(tpl.method)
      setParamsJson(JSON.stringify(tpl.params, null, 2))
    }
  }

  const handleExecute = async () => {
    setExecuting(true)
    setError(null)
    setResponseOutput(null)

    try {
      let parsedParams = []
      if (paramsJson.trim()) {
        parsedParams = JSON.parse(paramsJson)
      }
      const res = await client.dispatch(method, parsedParams)
      setResponseOutput(JSON.stringify(res, null, 2))
    } catch (err: any) {
      setError(err.message || 'RPC Call Failed')
    } finally {
      setExecuting(false)
    }
  }

  return (
    <section className="tab-pane">
      <div className="tab-header">
        <div>
          <h2 className="tab-title">Interactive JSON-RPC 2.0 Console</h2>
          <p className="tab-subtitle">Test and debug node RPC endpoints directly</p>
        </div>
      </div>

      <div className="card">
        <div className="form-group mb-3">
          <label>Pre-filled Method Templates</label>
          <div className="btn-group">
            {Object.keys(TEMPLATES).map((key) => (
              <button
                key={key}
                type="button"
                className={`btn-sm ${selectedTemplate === key ? 'primary' : 'secondary'}`}
                onClick={() => handleTemplateSelect(key)}
              >
                {TEMPLATES[key].method}
              </button>
            ))}
          </div>
        </div>

        <div className="form-grid">
          <div className="form-group">
            <label htmlFor="rpc-method">Method</label>
            <input
              id="rpc-method"
              type="text"
              value={method}
              onChange={(e) => setMethod(e.target.value)}
              className="form-input"
            />
          </div>

          <div className="form-group">
            <label htmlFor="rpc-params">Parameters (JSON array)</label>
            <textarea
              id="rpc-params"
              rows={3}
              value={paramsJson}
              onChange={(e) => setParamsJson(e.target.value)}
              className="form-input code-font"
            />
          </div>
        </div>

        <div className="form-action mt-3">
          <button
            onClick={handleExecute}
            className="btn primary"
            disabled={executing}
            aria-label="Execute RPC Call"
          >
            {executing ? 'Executing...' : '⚡ Execute RPC Call'}
          </button>
        </div>
      </div>

      {error && (
        <div className="alert-box error mt-4" role="alert">
          <span>⚠️ {error}</span>
        </div>
      )}

      {responseOutput && (
        <div className="card mt-4">
          <h3 className="card-title">Response (JSON-RPC 2.0 Result)</h3>
          <pre className="code-block">
            <code>{responseOutput}</code>
          </pre>
        </div>
      )}
    </section>
  )
}
