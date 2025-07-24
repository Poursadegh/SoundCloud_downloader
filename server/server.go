package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pb "soundcloud-downloader/proto"
)

type downloadServer struct {
	pb.UnimplementedDownloadServiceServer
	downloads map[string]*DownloadInfo
	mu        sync.RWMutex
	client    *http.Client
}

type DownloadInfo struct {
	ID            string
	SoundCloudURL string
	Status        string
	FilePath      string
	FileSize      int64
	CreatedAt     time.Time
	CompletedAt   *time.Time
	ErrorMessage  string
	Progress      int32
}

func newDownloadServer() *downloadServer {
	return &downloadServer{
		downloads: make(map[string]*DownloadInfo),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *downloadServer) DownloadTrack(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	// Validate request
	if req.SoundcloudUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "soundcloud_url is required")
	}

	if !strings.Contains(req.SoundcloudUrl, "soundcloud.com") {
		return nil, status.Error(codes.InvalidArgument, "invalid SoundCloud URL")
	}

	// Generate download ID
	downloadID := generateDownloadID()

	// Create download info
	downloadInfo := &DownloadInfo{
		ID:            downloadID,
		SoundCloudURL: req.SoundcloudUrl,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	// Store download info
	s.mu.Lock()
	s.downloads[downloadID] = downloadInfo
	s.mu.Unlock()

	// Start download in background
	go s.performDownload(downloadID, req)

	return &pb.DownloadResponse{
		DownloadId: downloadID,
		Status:     "started",
		Message:    "Download started",
	}, nil
}

func (s *downloadServer) GetDownloadStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	if req.DownloadId == "" {
		return nil, status.Error(codes.InvalidArgument, "download_id is required")
	}

	s.mu.RLock()
	download, exists := s.downloads[req.DownloadId]
	s.mu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "download not found")
	}

	var completedAt string
	if download.CompletedAt != nil {
		completedAt = download.CompletedAt.Format(time.RFC3339)
	}

	return &pb.StatusResponse{
		DownloadId:      download.ID,
		Status:          download.Status,
		Message:         fmt.Sprintf("Download %s", download.Status),
		ProgressPercent: download.Progress,
		FilePath:        download.FilePath,
		FileSize:        download.FileSize,
		ErrorMessage:    download.ErrorMessage,
	}, nil
}

func (s *downloadServer) ListDownloads(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var downloads []*pb.DownloadInfo
	count := 0

	for _, download := range s.downloads {
		if count >= int(req.Limit) && req.Limit > 0 {
			break
		}

		var completedAt string
		if download.CompletedAt != nil {
			completedAt = download.CompletedAt.Format(time.RFC3339)
		}

		downloads = append(downloads, &pb.DownloadInfo{
			DownloadId:    download.ID,
			SoundcloudUrl: download.SoundCloudURL,
			Status:        download.Status,
			FilePath:      download.FilePath,
			FileSize:      download.FileSize,
			CreatedAt:     download.CreatedAt.Format(time.RFC3339),
			CompletedAt:   completedAt,
			ErrorMessage:  download.ErrorMessage,
		})

		count++
	}

	return &pb.ListResponse{
		Downloads:  downloads,
		TotalCount: int32(len(s.downloads)),
	}, nil
}

func (s *downloadServer) performDownload(downloadID string, req *pb.DownloadRequest) {
	s.updateDownloadStatus(downloadID, "downloading", 0, "")

	// Extract track info
	clientID, trackID, err := s.extractTrackInfo(req.SoundcloudUrl)
	if err != nil {
		s.updateDownloadStatus(downloadID, "failed", 0, err.Error())
		return
	}

	s.updateDownloadStatus(downloadID, "downloading", 25, "Track info extracted")

	// Get stream URL
	streamURL, err := s.getStreamURL(clientID, trackID)
	if err != nil {
		s.updateDownloadStatus(downloadID, "failed", 0, err.Error())
		return
	}

	s.updateDownloadStatus(downloadID, "downloading", 50, "Stream URL obtained")

	// Set output directory
	outputDir := req.OutputDirectory
	if outputDir == "" {
		outputDir = "downloads"
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		s.updateDownloadStatus(downloadID, "failed", 0, fmt.Sprintf("failed to create output directory: %v", err))
		return
	}

	// Generate filename
	filename := req.Filename
	if filename == "" {
		filename = fmt.Sprintf("soundcloud_%s.mp3", trackID)
	}
	if !strings.HasSuffix(filename, ".mp3") {
		filename += ".mp3"
	}

	outputPath := filepath.Join(outputDir, filename)

	// Download the file
	s.updateDownloadStatus(downloadID, "downloading", 75, "Downloading file")

	resp, err := s.client.Get(streamURL)
	if err != nil {
		s.updateDownloadStatus(downloadID, "failed", 0, fmt.Sprintf("failed to download file: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.updateDownloadStatus(downloadID, "failed", 0, fmt.Sprintf("failed to download file, status: %d", resp.StatusCode))
		return
	}

	file, err := os.Create(outputPath)
	if err != nil {
		s.updateDownloadStatus(downloadID, "failed", 0, fmt.Sprintf("failed to create output file: %v", err))
		return
	}
	defer file.Close()

	// Copy the response body to the file
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		s.updateDownloadStatus(downloadID, "failed", 0, fmt.Sprintf("failed to write file: %v", err))
		return
	}

	// Update final status
	now := time.Now()
	s.mu.Lock()
	if download, exists := s.downloads[downloadID]; exists {
		download.Status = "completed"
		download.FilePath = outputPath
		download.FileSize = written
		download.CompletedAt = &now
		download.Progress = 100
	}
	s.mu.Unlock()
}

func (s *downloadServer) updateDownloadStatus(downloadID, status string, progress int32, message string) {
	s.mu.Lock()
	if download, exists := s.downloads[downloadID]; exists {
		download.Status = status
		download.Progress = progress
		if message != "" {
			download.ErrorMessage = message
		}
	}
	s.mu.Unlock()
}

func (s *downloadServer) extractTrackInfo(soundcloudURL string) (string, string, error) {
	// Fetch the SoundCloud page
	resp, err := s.client.Get(soundcloudURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch SoundCloud page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch page, status: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Try to find client_id in the page source
	clientIDRegex := regexp.MustCompile(`client_id:"([^"]+)"`)
	matches := clientIDRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", "", fmt.Errorf("could not find client_id in page source")
	}

	clientID := matches[1]

	// Extract track ID from URL
	trackIDRegex := regexp.MustCompile(`soundcloud\.com/[^/]+/(\d+)`)
	trackMatches := trackIDRegex.FindStringSubmatch(soundcloudURL)
	if len(trackMatches) < 2 {
		return "", "", fmt.Errorf("could not extract track ID from URL")
	}

	trackID := trackMatches[1]

	return clientID, trackID, nil
}

func (s *downloadServer) getStreamURL(clientID, trackID string) (string, error) {
	// SoundCloud API endpoint for track info
	apiURL := fmt.Sprintf("https://api.soundcloud.com/i1/tracks/%s/streams?client_id=%s", trackID, clientID)

	resp, err := s.client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch stream info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get stream info, status: %d", resp.StatusCode)
	}

	// Parse the response to get the stream URL
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Look for the stream URL in the response
	streamURLRegex := regexp.MustCompile(`"http_mp3_128_url":"([^"]+)"`)
	matches := streamURLRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find stream URL in response")
	}

	streamURL := strings.ReplaceAll(matches[1], "\\u0026", "&")
	return streamURL, nil
}

func generateDownloadID() string {
	return fmt.Sprintf("dl_%d", time.Now().UnixNano())
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterDownloadServiceServer(s, newDownloadServer())

	// Register reflection service on gRPC server
	reflection.Register(s)

	fmt.Println("SoundCloud Download Server starting on :50051")
	if err := s.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
		os.Exit(1)
	}
}
