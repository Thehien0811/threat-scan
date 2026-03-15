'use client'

import { ShieldCheck, AlertTriangle, Copy, Check } from 'lucide-react'
import { useState } from 'react'
import { ScanResponse, ScanResult } from '@/types/scan'
import { cn, getStatusColor, getStatusLabel } from '@/lib/utils'

export default function ScanResultCard({ result }: { result: ScanResponse }) {
  const [copied, setCopied] = useState(false)

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const statusColor = result.status === 'clean' ? 'text-safe' : 'text-infected'
  const statusBgColor = result.status === 'clean' ? 'bg-safe/20' : 'bg-infected/20'
  const statusIcon = result.status === 'clean' ? ShieldCheck : AlertTriangle

  const StatusIcon = statusIcon

  return (
    <div className="bg-gradient-to-br from-cyber-card to-slate-900 border border-slate-600 rounded-lg p-6 space-y-6">
      {/* Header with Status */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4 flex-1">
          <div className={cn('p-3 rounded-lg', statusBgColor)}>
            <StatusIcon className={cn('w-6 h-6', statusColor)} />
          </div>
          <div className="flex-1">
            <p className="text-sm text-slate-400">Scan Result</p>
            <h3 className={cn('text-2xl font-bold', statusColor)}>
              {getStatusLabel(result.status)}
            </h3>
          </div>
        </div>
      </div>

      {/* File Info */}
      <div className="space-y-3 border-t border-slate-700 pt-4">
        <div>
          <p className="text-sm text-slate-400 mb-1">Filename</p>
          <p className="font-mono text-sm bg-slate-900/50 rounded px-3 py-2">
            {result.filename || 'Unknown'}
          </p>
        </div>

        {result.sha256 && (
          <div>
            <p className="text-sm text-slate-400 mb-1">SHA256</p>
            <div className="flex items-center justify-between bg-slate-900/50 rounded px-3 py-2">
              <p className="font-mono text-xs text-slate-300 truncate">
                {result.sha256}
              </p>
              <button
                onClick={() => copyToClipboard(result.sha256 || '')}
                className="ml-2 p-1.5 hover:bg-slate-800 rounded transition"
              >
                {copied ? (
                  <Check className="w-4 h-4 text-safe" />
                ) : (
                  <Copy className="w-4 h-4 text-slate-400" />
                )}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Scan Results */}
      {result.results && result.results.length > 0 && (
        <div className="border-t border-slate-700 pt-4">
          <p className="text-sm text-slate-400 mb-3">Engine Results</p>
          <div className="space-y-2">
            {result.results.map((res: ScanResult, idx: number) => (
              <div
                key={idx}
                className="bg-slate-900/50 rounded px-3 py-2 flex items-center justify-between"
              >
                <div>
                  <p className="font-semibold capitalize">{res.engine}</p>
                  <p className="text-xs text-slate-400">{res.details || 'No details'}</p>
                </div>
                <div
                  className={cn(
                    'px-2 py-1 rounded text-xs font-semibold',
                    res.status === 'clean'
                      ? 'bg-safe/20 text-safe'
                      : 'bg-infected/20 text-infected'
                  )}
                >
                  {res.status === 'clean' ? '✓ Clean' : '⚠ Infected'}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Error Message */}
      {result.error_message && (
        <div className="bg-infected/20 border border-infected/50 rounded px-3 py-2">
          <p className="text-xs text-infected font-semibold mb-1">Error</p>
          <p className="text-xs text-slate-300">{result.error_message}</p>
        </div>
      )}
    </div>
  )
}
