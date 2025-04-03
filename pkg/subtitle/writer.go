package subtitle

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"yt-autosub-replace/pkg/models"
)

// WriteSRT writes subtitles to an SRT file
func WriteSRT(subtitles []models.Subtitle, outputPath string) error {
	var srtBuilder strings.Builder

	for i, subtitle := range subtitles {
		// Convert milliseconds to SRT timestamp format
		startTime := millisecondsToSRTTimestamp(subtitle.StartMs)
		endTime := millisecondsToSRTTimestamp(subtitle.EndMs)

		// Write SRT entry
		srtBuilder.WriteString(fmt.Sprintf("%d\n", i+1))
		srtBuilder.WriteString(fmt.Sprintf("%s --> %s\n", startTime, endTime))
		srtBuilder.WriteString(fmt.Sprintf("%s\n\n", subtitle.Text))
	}

	return os.WriteFile(outputPath, []byte(srtBuilder.String()), 0644)
}

// WriteJSON writes subtitles to a JSON file
func WriteJSON(subtitles []models.Subtitle, outputPath string) error {
	data, err := json.MarshalIndent(subtitles, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// Helper function to convert milliseconds to SRT timestamp format (HH:MM:SS,MMM)
func millisecondsToSRTTimestamp(ms int) string {
	duration := time.Duration(ms) * time.Millisecond
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	milliseconds := ms % 1000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}
