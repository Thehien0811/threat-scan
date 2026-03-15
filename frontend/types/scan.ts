export type ScanStatus = "safe" | "infected" | "warning" | "scanning";

export interface ThreatDetail {
  name: string;
  severity: "critical" | "high" | "medium" | "low";
  description: string;
}

/* Engine result */
export interface EngineScanResult {
  engine: string;
  status: ScanStatus;
  details?: string;
}

/* File scan summary */
export interface ScanResult {
  id: string;
  filename: string;
  fileType: string;
  fileSize: number;
  sha256: string;
  status: ScanStatus;
  threats: ThreatDetail[];
  engineHits: number;
  totalEngines: number;
  scanDuration: number;
  scannedAt: Date;
}

/* Upload state */
export interface UploadedFile {
  file: File;
  id: string;
  progress: number;
  status: "pending" | "scanning" | "done" | "error";
  phase: "idle" | "signature" | "heuristic" | "behavioral" | "complete";
  result?: ScanResult;
}

/* History */
export interface ScanHistoryItem {
  id: string;
  filename: string;
  fileType: string;
  fileSize: number;
  sha256: string;
  status: ScanStatus;
  threats: ThreatDetail[];
  engineHits: number;
  totalEngines: number;
  scanDuration: number;
  scannedAt: Date;
}

/* API response */
export interface ScanResponse {
  status: ScanStatus;
  filename?: string;
  sha256?: string;
  results?: EngineScanResult[];
  error_message?: string;
}

/**
 * Return color class for scan status
 */
export function getStatusColor(status: ScanStatus): string {
  switch (status) {
    case "safe":
      return "text-green-600";
    case "infected":
      return "text-red-600";
    case "warning":
      return "text-yellow-600";
    case "scanning":
      return "text-blue-600";
    default:
      return "text-gray-600";
  } 
}
