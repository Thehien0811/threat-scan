'use client'

import UploadDropzone from '@/components/UploadDropzone'

export default function UploadPage() {
  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold">Upload & Scan File</h2>
        <p className="text-slate-400">
          Drag and drop your file or click to upload. We'll scan it for threats.
        </p>
      </div>

      <UploadDropzone />
    </div>
  )
}
