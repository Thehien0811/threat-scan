import { ShieldCheck, Upload, History, Zap } from 'lucide-react'
import Link from 'next/link'

export default function Dashboard() {
  return (
    <div className="space-y-10">
      {/* Hero Section */}
      <div className="text-center space-y-4 py-12">
        <div className="flex justify-center">
          <ShieldCheck className="w-16 h-16 text-safe animate-pulse" />
        </div>
        <h2 className="text-4xl font-bold bg-gradient-to-r from-safe via-blue-400 to-safe bg-clip-text text-transparent">
          Cybersecurity Dashboard
        </h2>
        <p className="text-slate-400 max-w-2xl mx-auto">
          Scan files for threats, track history, and protect against malware in real-time.
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-cyber-card border border-slate-700 rounded-lg p-6 hover:border-safe/50 transition">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Files Scanned</p>
              <p className="text-3xl font-bold">42</p>
            </div>
            <Zap className="w-8 h-8 text-safe" />
          </div>
        </div>
        <div className="bg-cyber-card border border-slate-700 rounded-lg p-6 hover:border-safe/50 transition">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Safe Files</p>
              <p className="text-3xl font-bold">40</p>
            </div>
            <ShieldCheck className="w-8 h-8 text-safe" />
          </div>
        </div>
        <div className="bg-cyber-card border border-slate-700 rounded-lg p-6 hover:border-infected/50 transition">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-slate-400 text-sm">Threats Detected</p>
              <p className="text-3xl font-bold">2</p>
            </div>
            <div className="w-8 h-8 bg-infected/20 rounded-lg flex items-center justify-center">
              <span className="text-infected font-bold">!</span>
            </div>
          </div>
        </div>
      </div>

      {/* Action Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Link href="/upload">
          <div className="bg-gradient-to-br from-cyber-card to-slate-900 border border-slate-700 rounded-lg p-8 hover:border-safe/50 hover:shadow-lg hover:shadow-safe/20 transition cursor-pointer h-full">
            <Upload className="w-12 h-12 text-safe mb-4" />
            <h3 className="text-xl font-bold mb-2">Upload & Scan</h3>
            <p className="text-slate-400">
              Scan files for threats using advanced malware detection engines.
            </p>
          </div>
        </Link>
        <Link href="/history">
          <div className="bg-gradient-to-br from-cyber-card to-slate-900 border border-slate-700 rounded-lg p-8 hover:border-safe/50 hover:shadow-lg hover:shadow-safe/20 transition cursor-pointer h-full">
            <History className="w-12 h-12 text-safe mb-4" />
            <h3 className="text-xl font-bold mb-2">Scan History</h3>
            <p className="text-slate-400">
              View all scans, results, and threat detections in one place.
            </p>
          </div>
        </Link>
      </div>
    </div>
  )
}
