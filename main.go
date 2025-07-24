package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type SoundCloudDownloader struct {
	client *http.Client
}

func NewSoundCloudDownloader() *SoundCloudDownloader {
	return &SoundCloudDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (sc *SoundCloudDownloader) extractTrackInfo(soundcloudURL string) (string, string, error) {
	// Fetch the SoundCloud page
	resp, err := sc.client.Get(soundcloudURL)
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

	// Look for the client_id in the page source

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

func (sc *SoundCloudDownloader) getStreamURL(clientID, trackID string) (string, error) {
	// SoundCloud API endpoint for track info
	apiURL := fmt.Sprintf("https://api.soundcloud.com/i1/tracks/%s/streams?client_id=%s", trackID, clientID)

	resp, err := sc.client.Get(apiURL)
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

func (sc *SoundCloudDownloader) downloadTrack(soundcloudURL, outputDir string) error {
	fmt.Printf("Analyzing SoundCloud URL: %s\n", soundcloudURL)

	// Extract track info
	clientID, trackID, err := sc.extractTrackInfo(soundcloudURL)
	if err != nil {
		return fmt.Errorf("failed to extract track info: %v", err)
	}

	fmt.Printf("Track ID: %s\n", trackID)

	// Get stream URL
	streamURL, err := sc.getStreamURL(clientID, trackID)
	if err != nil {
		return fmt.Errorf("failed to get stream URL: %v", err)
	}

	fmt.Printf("Stream URL found\n")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate filename
	filename := fmt.Sprintf("soundcloud_%s.mp3", trackID)
	outputPath := filepath.Join(outputDir, filename)

	// Download the file
	fmt.Printf("Downloading to: %s\n", outputPath)

	resp, err := sc.client.Get(streamURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status: %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("Download completed successfully!\n")
	return nil
}

func main() {
	var outputDir string

	var rootCmd = &cobra.Command{
		Use:   "soundcloud-downloader [URL]",
		Short: "Download MP3 files from SoundCloud",
		Long: `A command line tool to download MP3 files from SoundCloud tracks.
		
Example:
  soundcloud-downloader "https://soundcloud.com/artist/track-name"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			soundcloudURL := args[0]

			// Validate URL
			if !strings.Contains(soundcloudURL, "soundcloud.com") {
				return fmt.Errorf("invalid SoundCloud URL")
			}

			// Set default output directory if not specified
			if outputDir == "" {
				outputDir = "downloads"
			}

			downloader := NewSoundCloudDownloader()
			return downloader.downloadTrack(soundcloudURL, outputDir)
		},
	}

	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "downloads", "Output directory for downloaded files")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
