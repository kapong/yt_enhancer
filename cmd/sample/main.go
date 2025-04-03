package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"yt-autosub-replace/pkg/config"
	"yt-autosub-replace/pkg/gemini"
	"yt-autosub-replace/pkg/parser"
	"yt-autosub-replace/pkg/subtitle"

	"github.com/lrstanley/go-ytdlp"
)

const (
	slowDownload         = false
	defaultOutputPattern = "%(uploader)s-%(display_id)s"
	defaultProgressBar   = 40
)

// downloadOptions holds configuration for the download process
type downloadOptions struct {
	limitRate    string
	outputFormat string
	subLang      string
	subFormat    string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command line flags
	envFile := flag.String("env", ".env", "Environment file path")
	flag.Parse()

	// Validate command line arguments
	if len(flag.Args()) < 1 {
		return fmt.Errorf("usage: go run main.go <video_url> [custom_filename]")
	}

	url := flag.Arg(0)

	// Check if custom filename was provided as second argument
	var customFilename string
	if len(flag.Args()) > 1 {
		customFilename = flag.Arg(1)
	}

	// Load configuration
	cfg, err := loadConfig(*envFile)
	if err != nil {
		return err
	}

	// Install yt-dlp if needed
	fmt.Println("Checking yt-dlp installation...")
	ytdlp.MustInstall(context.TODO(), nil)

	// Download video and subtitles
	fmt.Printf("Downloading: %s\n", url)
	srv3Path, err := downloadVideo(url, customFilename)
	if err != nil {
		return fmt.Errorf("error downloading video: %w", err)
	}
	fmt.Printf("\nDownload complete!\nSaved to: %s\n", srv3Path)

	// Generate SRT file using Gemini API
	fmt.Println("Recreating subtitles with Gemini API")
	srtOutputPath := strings.TrimSuffix(srv3Path, ".srv3") + ".srt"

	if err := processSubtitles(cfg, srv3Path, srtOutputPath); err != nil {
		return fmt.Errorf("error processing subtitles: %w", err)
	}

	fmt.Printf("Successfully processed and created %s\n", srtOutputPath)
	return nil
}

// loadConfig loads the application configuration
func loadConfig(envFile string) (*config.Config, error) {
	// Load environment variables from .env file (optional)
	if err := config.LoadEnvFile(envFile); err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}
	return cfg, nil
}

// downloadVideo downloads a video and returns the subtitle file path
func downloadVideo(url string, customFilename string) (string, error) {
	// Determine output format
	outputPattern := defaultOutputPattern
	if customFilename != "" {
		outputPattern = customFilename
	}

	outputFormat := fmt.Sprintf("output/%s.%%(ext)s", outputPattern)

	opts := downloadOptions{
		outputFormat: outputFormat,
		subLang:      "th",
		subFormat:    "srv3",
	}

	if slowDownload {
		opts.limitRate = "2M"
	}

	return executeDownload(context.Background(), url, opts)
}

// executeDownload handles the actual download process with progress reporting
func executeDownload(ctx context.Context, url string, opts downloadOptions) (string, error) {
	// Configure downloader
	dl := ytdlp.New().
		FormatSort("res,ext:mp4:m4a").
		RecodeVideo("mp4").
		ForceOverwrites().
		WriteThumbnail().
		SubLangs(opts.subLang).
		SubFormat(opts.subFormat).
		WriteAutoSubs().
		Output(opts.outputFormat)

	if opts.limitRate != "" {
		dl = dl.LimitRate(opts.limitRate)
	}

	var subPath string
	// Setup progress handler
	dl = dl.ProgressFunc(100*time.Millisecond, func(prog ytdlp.ProgressUpdate) {
		fmt.Printf("\r%s %s %.1f%%",
			string(prog.Status),
			prog.Filename,
			prog.Percent())

		if prog.Status == ytdlp.ProgressStatusFinished && prog.Filename != "" {
			subPath = prog.Filename
		}
	})

	// Run the download
	_, err := dl.Run(ctx, url)
	if err != nil {
		return "", err
	}

	return subPath, nil
}

// processSubtitles handles the subtitle processing pipeline
func processSubtitles(cfg *config.Config, inputPath, outputPath string) error {
	// Parse the XML file
	timedText, err := parser.ParseXMLFile(inputPath)
	if err != nil {
		return fmt.Errorf("error parsing XML: %w", err)
	}

	// Extract word timings
	wordTimings := parser.ExtractWordTimings(timedText)
	if len(wordTimings) == 0 {
		return fmt.Errorf("no word timings extracted")
	}

	// Create a Gemini client and generate subtitles
	client := gemini.NewClient(cfg)
	subtitles, err := client.CreateSubtitles(wordTimings)
	if err != nil {
		return fmt.Errorf("error creating subtitles: %w", err)
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Write SRT file
	if err := subtitle.WriteSRT(subtitles, outputPath); err != nil {
		return fmt.Errorf("error writing SRT file: %w", err)
	}

	fmt.Printf("Successfully processed %d words into %d subtitle blocks\n",
		len(wordTimings), len(subtitles))
	return nil
}
