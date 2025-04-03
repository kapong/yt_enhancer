package parser

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"

	"yt_enhancer/pkg/models"
)

// ParseXMLFile reads and parses an XML file containing timed text
func ParseXMLFile(filePath string) (models.TimedText, error) {
	var timedText models.TimedText

	// Read the XML file
	xmlData, err := os.ReadFile(filePath)
	if err != nil {
		return timedText, fmt.Errorf("error reading file: %w", err)
	}

	// Remove the filepath comment line if present
	xmlContent := string(xmlData)
	lines := strings.Split(xmlContent, "\n")
	for i, line := range lines {
		if strings.Contains(line, "// filepath:") {
			xmlContent = strings.Join(lines[i+1:], "\n")
			break
		}
	}

	// Parse the XML
	err = xml.Unmarshal([]byte(xmlContent), &timedText)
	if err != nil {
		return timedText, fmt.Errorf("error parsing XML: %w", err)
	}

	return timedText, nil
}

// ExtractWordTimings extracts word timings from a TimedText structure
func ExtractWordTimings(timedText models.TimedText) []models.WordTiming {
	var wordTimings []models.WordTiming
	wordID := 0

	for _, paragraph := range timedText.Body.Paragraphs {
		// Skip empty paragraphs or those without sentences
		if len(paragraph.Sentences) == 0 {
			continue
		}

		paragraphTime, _ := strconv.Atoi(paragraph.Time)

		for _, sentence := range paragraph.Sentences {
			sentenceTime, _ := strconv.Atoi(sentence.Time)
			startTime := paragraphTime + sentenceTime

			// Skip empty sentences
			if strings.TrimSpace(sentence.Text) == "" {
				continue
			}

			wordTimings = append(wordTimings, models.WordTiming{
				ID:        wordID,
				Word:      strings.TrimSpace(sentence.Text),
				StartTime: startTime,
			})

			wordID++
		}
	}

	return wordTimings
}
