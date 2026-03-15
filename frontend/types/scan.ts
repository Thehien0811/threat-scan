export type ScanStatus = "safe" | "infected" | "warning" | "scanning";

export interface ThreatDetail {
  name: string;
  severity: "critical" | "high" | "medium" | "low";
  description: string;
}

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

export interface UploadedFile {
  file: File;
  id: string;
  progress: number;
  status: "pending" | "scanning" | "done" | "error";
  phase: "idle" | "signature" | "heuristic" | "behavioral" | "complete";
  result?: ScanResult;
}