import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ScanResult, ThreatDetail } from "@/types/scan";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1_048_576) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1_048_576).toFixed(1)} MB`;
}

export function formatDuration(ms: number): string {
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`;
}

export const getStatusLabel = (status: string) => {
  return status === "clean" ? "SAFE" : "INFECTED"
}

export function getFileType(filename: string): string {
  const ext = filename.split(".").pop()?.toLowerCase() ?? "";
  const map: Record<string, string> = {
    exe: "PE32", dll: "DLL", bat: "Batch", sh: "Shell",
    zip: "ZIP", rar: "RAR", "7z": "7-Zip",
    pdf: "PDF", doc: "Word", docx: "Word",
    xls: "Excel", xlsx: "Excel", pptx: "PowerPoint",
    js: "JavaScript", ts: "TypeScript", py: "Python",
  };
  return map[ext] ?? ext.toUpperCase();
}

function randomHex(len: number) {
  return Array.from({ length: len }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]).join("");
}

export function simulateScanResult(file: File): ScanResult {
  const ext = file.name.split(".").pop()?.toLowerCase() ?? "";
  const dangerous = ["exe","dll","sh","bat","cmd","vbs","ps1"];
  const warn = ["docx","doc","xlsm","xls"];
  const isThreat = dangerous.includes(ext) && Math.random() > 0.45;
  const isWarn = !isThreat && warn.includes(ext) && Math.random() > 0.55;

  const threats: ThreatDetail[] = [];
  if (isThreat) {
    const pool: ThreatDetail[] = [
      { name: "Trojan.Win32.Inject", severity: "critical", description: "Injects malicious code into running processes." },
      { name: "Backdoor.Shell.RC",   severity: "critical", description: "Opens a remote shell for unauthorized access." },
      { name: "PUP.KeyGen",          severity: "high",     description: "Key-generator with bundled adware payload." },
      { name: "Suspicious API calls",severity: "high",     description: "Sensitive OS APIs called in suspicious sequence." },
    ];
    threats.push(...pool.slice(0, Math.floor(Math.random() * 2) + 1));
  } else if (isWarn) {
    threats.push({ name: "Macro code detected", severity: "medium", description: "Embedded macro scripts found in document." });
  }

  return {
    id: Math.random().toString(36).slice(2),
    filename: file.name,
    fileType: getFileType(file.name),
    fileSize: file.size,
    sha256: randomHex(64),
    status: isThreat ? "infected" : isWarn ? "warning" : "safe",
    threats,
    engineHits: threats.length,
    totalEngines: 3,
    scanDuration: Math.floor(Math.random() * 2800 + 300),
    scannedAt: new Date(),
  };
}

export const MOCK_HISTORY: ScanResult[] = [
  {
    id: "1", filename: "setup_v2.exe", fileType: "PE32", fileSize: 4_404_019,
    sha256: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
    status: "infected", engineHits: 2, totalEngines: 3, scanDuration: 1200,
    scannedAt: new Date("2026-03-15T10:42:00"),
    threats: [
      { name: "Trojan.Win32.Inject", severity: "critical", description: "Injects malicious code into running processes." },
      { name: "Suspicious API calls", severity: "high", description: "Sensitive OS APIs called in suspicious sequence." },
    ],
  },
  {
    id: "2", filename: "report_q1.pdf", fileType: "PDF", fileSize: 913_408,
    sha256: "b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3d4e5f6b2c3",
    status: "safe", engineHits: 0, totalEngines: 3, scanDuration: 380,
    scannedAt: new Date("2026-03-15T10:18:00"), threats: [],
  },
  {
    id: "3", filename: "backup.zip", fileType: "ZIP", fileSize: 50_435_482,
    sha256: "c3d4e5f6c3d4e5f6c3d4e5f6c3d4e5f6c3d4e5f6c3d4e5f6c3d4e5f6c3d4e5f6",
    status: "safe", engineHits: 0, totalEngines: 3, scanDuration: 3800,
    scannedAt: new Date("2026-03-15T09:54:00"), threats: [],
  },
  {
    id: "4", filename: "macros.docx", fileType: "Word", fileSize: 1_153_434,
    sha256: "d4e5f6a1d4e5f6a1d4e5f6a1d4e5f6a1d4e5f6a1d4e5f6a1d4e5f6a1d4e5f6a1",
    status: "warning", engineHits: 1, totalEngines: 3, scanDuration: 940,
    scannedAt: new Date("2026-03-15T09:31:00"),
    threats: [{ name: "Macro code detected", severity: "medium", description: "Embedded macro scripts found in document." }],
  },
  {
    id: "5", filename: "chrome_patch.exe", fileType: "PE32", fileSize: 2_936_013,
    sha256: "e5f6a1b2e5f6a1b2e5f6a1b2e5f6a1b2e5f6a1b2e5f6a1b2e5f6a1b2e5f6a1b2",
    status: "safe", engineHits: 0, totalEngines: 3, scanDuration: 1100,
    scannedAt: new Date("2026-03-14T17:22:00"), threats: [],
  },
  {
    id: "6", filename: "runner.sh", fileType: "Shell", fileSize: 12_288,
    sha256: "f6a1b2c3f6a1b2c3f6a1b2c3f6a1b2c3f6a1b2c3f6a1b2c3f6a1b2c3f6a1b2c3",
    status: "infected", engineHits: 2, totalEngines: 3, scanDuration: 220,
    scannedAt: new Date("2026-03-14T15:44:00"),
    threats: [
      { name: "Backdoor.Shell.RC", severity: "critical", description: "Opens a remote shell for unauthorized access." },
      { name: "Suspicious API calls", severity: "high", description: "Sensitive OS APIs called in suspicious sequence." },
    ],
  },
  {
    id: "7", filename: "data.xlsx", fileType: "Excel", fileSize: 3_565_158,
    sha256: "a7b8c9d0a7b8c9d0a7b8c9d0a7b8c9d0a7b8c9d0a7b8c9d0a7b8c9d0a7b8c9d0",
    status: "safe", engineHits: 0, totalEngines: 3, scanDuration: 700,
    scannedAt: new Date("2026-03-14T14:20:00"), threats: [],
  },
  {
    id: "8", filename: "keygen.exe", fileType: "PE32", fileSize: 1_887_437,
    sha256: "b8c9d0a1b8c9d0a1b8c9d0a1b8c9d0a1b8c9d0a1b8c9d0a1b8c9d0a1b8c9d0a1",
    status: "infected", engineHits: 2, totalEngines: 3, scanDuration: 800,
    scannedAt: new Date("2026-03-13T16:10:00"),
    threats: [
      { name: "PUP.KeyGen", severity: "high", description: "Key-generator with bundled adware payload." },
      { name: "Suspicious API calls", severity: "high", description: "Sensitive OS APIs called in suspicious sequence." },
    ],
  },
];