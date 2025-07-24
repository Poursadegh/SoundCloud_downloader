package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "soundcloud-downloader/proto"
)

type DownloadClient struct {
	client pb.DownloadServiceClient
	conn   *grpc.ClientConn
}

func NewDownloadClient(serverAddr string) (*DownloadClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	client := pb.NewDownloadServiceClient(conn)
	return &DownloadClient{
		client: client,
		conn:   conn,
	}, nil
}

func (dc *DownloadClient) Close() {
	if dc.conn != nil {
		dc.conn.Close()
	}
}

func (dc *DownloadClient) DownloadTrack(soundcloudURL, outputDir, filename string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &pb.DownloadRequest{
		SoundcloudUrl:   soundcloudURL,
		OutputDirectory: outputDir,
		Filename:        filename,
	}

	resp, err := dc.client.DownloadTrack(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start download: %v", err)
	}

	fmt.Printf("Download started with ID: %s\n", resp.DownloadId)
	fmt.Printf("Status: %s - %s\n", resp.Status, resp.Message)

	// Monitor download progress
	return dc.monitorDownload(resp.DownloadId)
}

func (dc *DownloadClient) monitorDownload(downloadID string) error {
	fmt.Println("Monitoring download progress...")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		status, err := dc.client.GetDownloadStatus(ctx, &pb.StatusRequest{DownloadId: downloadID})
		cancel()

		if err != nil {
			return fmt.Errorf("failed to get download status: %v", err)
		}

		fmt.Printf("Status: %s (%d%%) - %s\n", status.Status, status.ProgressPercent, status.Message)

		if status.Status == "completed" {
			fmt.Printf("Download completed successfully!\n")
			fmt.Printf("File saved to: %s\n", status.FilePath)
			fmt.Printf("File size: %d bytes\n", status.FileSize)
			return nil
		}

		if status.Status == "failed" {
			return fmt.Errorf("download failed: %s", status.ErrorMessage)
		}

		time.Sleep(2 * time.Second)
	}
}

func (dc *DownloadClient) ListDownloads(limit int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.ListRequest{
		Limit:  limit,
		Offset: 0,
	}

	resp, err := dc.client.ListDownloads(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to list downloads: %v", err)
	}

	fmt.Printf("Total downloads: %d\n", resp.TotalCount)
	fmt.Println("Recent downloads:")
	fmt.Println("ID\t\tStatus\t\tFile Path")
	fmt.Println("--\t\t------\t\t---------")

	for _, download := range resp.Downloads {
		fmt.Printf("%s\t%s\t%s\n", download.DownloadId, download.Status, download.FilePath)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: client <command> [args...]")
		fmt.Println("Commands:")
		fmt.Println("  download <url> [output_dir] [filename]")
		fmt.Println("  list [limit]")
		os.Exit(1)
	}

	client, err := NewDownloadClient("localhost:50051")
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	command := os.Args[1]

	switch command {
	case "download":
		if len(os.Args) < 3 {
			fmt.Println("Usage: client download <url> [output_dir] [filename]")
			os.Exit(1)
		}

		url := os.Args[2]
		outputDir := "downloads"
		filename := ""

		if len(os.Args) > 3 {
			outputDir = os.Args[3]
		}
		if len(os.Args) > 4 {
			filename = os.Args[4]
		}

		if err := client.DownloadTrack(url, outputDir, filename); err != nil {
			fmt.Printf("Download failed: %v\n", err)
			os.Exit(1)
		}

	case "list":
		limit := int32(10)
		if len(os.Args) > 2 {
			if _, err := fmt.Sscanf(os.Args[2], "%d", &limit); err != nil {
				fmt.Printf("Invalid limit: %s\n", os.Args[2])
				os.Exit(1)
			}
		}

		if err := client.ListDownloads(limit); err != nil {
			fmt.Printf("Failed to list downloads: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
