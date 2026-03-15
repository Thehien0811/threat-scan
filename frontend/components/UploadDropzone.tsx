'use client'

import { useState, useRef } from 'react'
import { Upload, Loader2, AlertCircle } from 'lucide-react'
import axios from 'axios'
import ScanResultCard from './ScanResultCard'
import { ScanResponse } from '@/types/scan'
import { cn } from '@/lib/utils'

export default function UploadDropzone() {
  const [isDragActive, setIsDragActive] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [result, setResult] = useState<ScanResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragActive(e.type === 'dragenter' || e.type === 'dragover')
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragActive(false)
    
    const files = e.dataTransfer.files
    if (files && files[0]) {
      uploadFile(files[0])
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.currentTarget.files
    if (files && files[0]) {
      uploadFile(files[0])
    }
  }

  const uploadFile = async (file: File) => {
    setIsLoading(true)
    setProgress(0)
    setError(null)
    setResult(null)

    try {
      const formData = new FormData()
      formData.append('file', file)

      const response = await axios.post('http://localhost:4000/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        onUploadProgress: (progressEvent) => {
          const total = progressEvent.total || 1
          setProgress(Math.round((progressEvent.loaded / total) * 100))
        },
      })

      setResult({
        status: response.data.status,
        results: response.data.results || [],
        error_message: response.data.error,
        filename: file.name,
        sha256: response.data.sha256,
      })
    } catch (err) {
      const errorMsg = axios.isAxiosError(err)
        ? err.response?.data?.error || err.message
        : 'Failed to upload file'
      setError(errorMsg)
    } finally {
      setIsLoading(false)
      setProgress(0)
    }
  }

  return (
    <div className="space-y-6">
      {/* Dropzone */}
      <div
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
        className={cn(
          'relative border-2 border-dashed rounded-lg p-12 text-center cursor-pointer transition-all',
          isDragActive
            ? 'border-safe bg-safe/10 shadow-lg shadow-safe/20'
            : 'border-slate-600 bg-slate-900/30 hover:border-slate-400'
        )}
      >
        <input
          ref={inputRef}
          type="file"
          onChange={handleChange}
          className="hidden"
          accept="*/*"
        />

        {!isLoading ? (
          <>
            <Upload className={cn(
              'w-12 h-12 mx-auto mb-4 transition-colors',
              isDragActive ? 'text-safe' : 'text-slate-400'
            )} />
            <p className="text-lg font-semibold mb-2">
              {isDragActive ? 'Drop your file here' : 'Drag and drop your file here'}
            </p>
            <p className="text-slate-400">
              or click to browse
            </p>
          </>
        ) : (
          <div className="space-y-4">
            <Loader2 className="w-12 h-12 mx-auto text-safe animate-spin" />
            <p className="text-lg font-semibold">Uploading... {progress}%</p>
            <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
              <div
                className="bg-gradient-to-r from-safe to-blue-500 h-full transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-infected/20 border border-infected/50 rounded-lg p-4 flex gap-3">
          <AlertCircle className="w-5 h-5 text-infected flex-shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold text-infected">Upload Failed</p>
            <p className="text-slate-300 text-sm">{error}</p>
          </div>
        </div>
      )}

      {/* Result Card */}
      {result && <ScanResultCard result={result} />}
    </div>
  )
}
