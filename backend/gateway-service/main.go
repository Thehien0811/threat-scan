package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	pb "github.com/threat-scan/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UploadResponse struct {
	Status  string   `json:"status"`
	Results []string `json:"results"`
	Error   string   `json:"error,omitempty"`
	SHA256  string   `json:"sha256"`
}

type Server struct {
	scanClient pb.ScanServiceClient
}

func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {

	// CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File not found", http.StatusBadRequest)
		return
	}
	defer file.Close()

	storageDir := "/var/file-storage"
	os.MkdirAll(storageDir, os.ModePerm)

	filePath := filepath.Join(storageDir, header.Filename)
	outFile, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(outFile, hasher)

	_, err = io.Copy(multiWriter, file)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	sha256sum := hex.EncodeToString(hasher.Sum(nil))

	scanRes, err := s.scanClient.Scan(r.Context(), &pb.ScanRequest{
		Filename: header.Filename,
		Sha256:   sha256sum,
		Filepath: header.Filename, // Assuming file is saved with original name
	})

	if err != nil {
		http.Error(w, "Failed to initiate scan", http.StatusInternalServerError)
		return
	}

	response := UploadResponse{
		Status:  scanRes.Status,
		Results: []string{},
		SHA256:  sha256sum,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {

	conn, err := grpc.Dial(
		"threat-scan-service:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		panic(err)
	}

	client := pb.NewScanServiceClient(conn)

	server := &Server{
		scanClient: client,
	}

	http.HandleFunc("/upload", server.uploadHandler)

	println("Server running on :4000")

	err = http.ListenAndServe(":4000", nil)
	if err != nil {
		panic(err)
	}
}
