package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"yt_enhancer/pkg/config"
	"yt_enhancer/pkg/gemini"
	"yt_enhancer/pkg/parser"
	"yt_enhancer/pkg/subtitle"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command line flags
	envFile := flag.String("env", ".env", "Environment file path")
	outputFile := flag.String("o", "", "Output file path (default: same as input with .srt extension)")
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	debugDir := flag.String("debug-dir", "debug", "Directory to store debug files")
	flag.Parse()

	// Validate command line arguments
	if len(flag.Args()) < 1 {
		return fmt.Errorf("usage: convert_srt [-env=.env] [-o=output.srt] [-debug] [-debug-dir=debug] input.srv3")
	}

	inputPath := flag.Arg(0)

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(inputPath), ".srv3") {
		return fmt.Errorf("input file must have .srv3 extension")
	}

	// Determine output path
	outputPath := *outputFile
	if outputPath == "" {
		outputPath = strings.TrimSuffix(inputPath, ".srv3") + ".srt"
	}

	// Load configuration
	cfg, err := loadConfig(*envFile)
	if err != nil {
		return err
	}

	// Override config with command line flags if provided
	if *debugMode {
		cfg.DebugMode = true
	}
	if *debugDir != "" {
		cfg.DebugDir = *debugDir
	}

	fmt.Printf("Converting %s to %s\n", inputPath, outputPath)

	// Process the subtitles
	if err := processSubtitles(cfg, inputPath, outputPath); err != nil {
		return fmt.Errorf("error processing subtitles: %w", err)
	}

	fmt.Printf("Successfully converted to %s\n", outputPath)
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
