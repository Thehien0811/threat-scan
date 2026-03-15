'use client'

import ScanHistoryTable from '@/components/ScanHistoryTable'

export default function HistoryPage() {
  return (
    <div className="space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold">Scan History</h2>
        <p className="text-slate-400">
          View all previously scanned files and their threat detection results.
        </p>
      </div>

      <ScanHistoryTable />
    </div>
  )
}
