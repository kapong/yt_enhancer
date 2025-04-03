# YT-Enhancer: YouTube Subtitle Enhancer

A Go-based tool for improving YouTube auto-generated subtitles using Google's Gemini AI. This project helps convert YouTube's raw srv3 subtitle files into well-formatted SRT files with natural sentence breaks and proper timing.

## Overview

YouTube's automatic subtitles (in srv3 format) often contain word-by-word timing data but lack proper sentence structure. This tool:

1. Downloads YouTube videos with auto-generated subtitles
2. Processes the subtitle data through Google's Gemini AI
3. Creates properly formatted SRT files with natural language structure
4. Maintains accurate timing synchronization with the video

## Features

- **Video Download**: Integrated YouTube video downloading via [go-ytdlp](https://github.com/lrstanley/go-ytdlp)
- **AI-Enhanced Subtitles**: Uses Gemini AI to improve subtitle readability
- **Batch Processing**: Handles large subtitles by processing them in manageable batches
- **Debug Options**: Includes debug mode for troubleshooting
- **Two Command Tools**:
  - `yt_enhancer`: Download videos and process subtitles in one step
  - `convert_srt`: Process existing srv3 files to SRT format

## Installation

### Prerequisites
- Go 1.18 or higher
- Google Gemini API key

### Setup
1. Clone the repository
2. Create a `.env` file with your Gemini API key:
   ```
   GEMINI_API_KEY=your_api_key_here
   ```
3. Build the tools:
```bash
go build -o bin/yt_enhancer ./cmd/yt_enhancer
go build -o bin/convert_srt ./cmd/convert_srt
```

## Usage

### Download and Process in One Step

```bash
./bin/yt_enhancer "https://www.youtube.com/watch?v=VIDEO_ID" [custom_filename]
```

This will:
- Download the YouTube video
- Extract auto-generated subtitles
- Process them through Gemini API
- Generate an SRT file

### Process Existing srv3 Files

```bash
./bin/convert_srt [-env=.env] [-o=output.srt] [-debug] [-debug-dir=debug] input.srv3
```

Options:
- `-env`: Path to environment file (default: `.env`)
- `-o`: Output file path (default: same as input with `.srt` extension)
- `-debug`: Enable debug mode
- `-debug-dir`: Directory to store debug files (default: `debug`)

## How It Works

1. **Subtitle Extraction**: Parses the srv3 XML file to extract word-level timing data
2. **Batch Processing**: Divides large subtitle files into manageable batches
3. **AI Processing**: Sends word timings to Gemini API for intelligent sentence formation
4. **Timing Adjustment**: Calculates appropriate display durations for each subtitle
5. **SRT Generation**: Creates properly formatted SRT files with exact timing information

## Project Structure

- **cmd/**: Command-line tools
  - **yt_enhancer/**: Video download and subtitle processor
  - **convert_srt/**: Standalone srv3 to SRT converter
- **pkg/**: Core functionality
  - **config/**: Configuration handling
  - **gemini/**: Gemini API client
  - **models/**: Data structures
  - **parser/**: srv3 XML parsing
  - **subtitle/**: SRT file generation

## Example Output

The tool converts raw word timings from YouTube into readable subtitles with natural sentence breaks while maintaining synchronization with the video content.

## License

This project is licensed under the MIT License.

## Acknowledgments

This project was developed with assistance from GitHub Copilot and other AI tools. The code, documentation, and project structure were created using AI-powered pair programming techniques.