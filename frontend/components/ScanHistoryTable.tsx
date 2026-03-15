'use client'

import { useEffect, useState } from 'react'
import { Loader2, ShieldCheck, AlertTriangle, Calendar } from 'lucide-react'
import { ScanHistoryItem } from '@/types/scan'
import { cn, getStatusColor } from '@/lib/utils'
import axios from 'axios'

export default function ScanHistoryTable() {
  const [history, setHistory] = useState<ScanHistoryItem[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchHistory()
  }, [])

  const fetchHistory = async () => {
    try {
      setIsLoading(true)
      const response = await axios.get('http://localhost:4000/history')
      setHistory(response.data.history || [])
    } catch (err) {
      const errorMsg = axios.isAxiosError(err)
        ? err.response?.data?.error || err.message
        : 'Failed to fetch history'
      setError(errorMsg)
    } finally {
      setIsLoading(false)
    }
  }

  if (isLoading) {
    return (
      <div className="bg-cyber-card border border-slate-700 rounded-lg p-12 flex flex-col items-center justify-center">
        <Loader2 className="w-8 h-8 text-safe animate-spin mb-3" />
        <p className="text-slate-400">Loading scan history...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-cyber-card border border-slate-700 rounded-lg p-6">
        <p className="text-infected">{error}</p>
        <button
          onClick={fetchHistory}
          className="mt-4 px-4 py-2 bg-safe text-white rounded hover:bg-safe/90 transition"
        >
          Retry
        </button>
      </div>
    )
  }

  if (history.length === 0) {
    return (
      <div className="bg-cyber-card border border-slate-700 rounded-lg p-12 text-center">
        <Calendar className="w-12 h-12 text-slate-500 mx-auto mb-3" />
        <p className="text-slate-400">No scans yet. Start by uploading a file!</p>
      </div>
    )
  }

  return (
    <div className="bg-cyber-card border border-slate-700 rounded-lg overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-slate-700 bg-slate-900/50">
              <th className="px-6 py-3 text-left text-xs font-semibold text-slate-400">Status</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-slate-400">Filename</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-slate-400">SHA256</th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-slate-400">Time</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-700">
            {history.map((item, idx) => (
              <tr key={idx} className="hover:bg-slate-900/30 transition">
                <td className="px-6 py-4">
                  <div className="flex items-center gap-2">
                    {item.status === 'clean' ? (
                      <ShieldCheck className="w-4 h-4 text-safe" />
                    ) : (
                      <AlertTriangle className="w-4 h-4 text-infected" />
                    )}
                    <span
                      className={cn(
                        'px-2 py-1 rounded text-xs font-semibold',
                        item.status === 'clean'
                          ? 'bg-safe/20 text-safe'
                          : item.status === 'infected'
                          ? 'bg-infected/20 text-infected'
                          : 'bg-slate-700/50 text-slate-300'
                      )}
                    >
                      {item.status === 'clean' ? '✓ Safe' : item.status === 'infected' ? '⚠ Infected' : '? Unknown'}
                    </span>
                  </div>
                </td>
                <td className="px-6 py-4 font-mono text-sm text-slate-300">
                  {item.filename}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-slate-400 max-w-xs truncate">
                  {item.sha256}
                </td>
                <td className="px-6 py-4 text-sm text-slate-400">
                  {new Date(item.timestamp * 1000).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
