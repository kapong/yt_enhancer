package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yt-autosub-replace/pkg/config"
	"yt-autosub-replace/pkg/models"
)

// Client is a client for the Gemini API
type Client struct {
	config     *config.Config
	httpClient *http.Client
	debugMode  bool
	debugDir   string
}

// Response structures for Gemini API
type Response struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content struct {
		Parts []Part `json:"parts"`
	} `json:"content"`
}

type Part struct {
	Text string `json:"text,omitempty"`
}

// NewClient creates a new Gemini API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Extended timeout for processing the entire transcript
		},
		debugMode: cfg.DebugMode,
		debugDir:  cfg.DebugDir,
	}
}

// CreateSubtitles creates subtitle blocks from word timings using Gemini API
func (c *Client) CreateSubtitles(wordTimings []models.WordTiming) ([]models.Subtitle, error) {
	// Create debug directory if it doesn't exist
	if c.debugMode && c.debugDir != "" {
		if err := os.MkdirAll(c.debugDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create debug directory: %w", err)
		}
	}

	// Process in batches of maximum 300 words
	var allSubtitles []models.Subtitle
	var startIndex int = 0
	var batchNum int = 1
	var batchSize int = 300

	for startIndex < len(wordTimings) {
		// Calculate batch size (maximum 300 words)
		endIndex := startIndex + batchSize
		if endIndex > len(wordTimings) {
			endIndex = len(wordTimings)
		}

		// Get the current batch
		currentBatch := wordTimings[startIndex:endIndex]

		fmt.Printf("Processing batch %d: words %d to %d (total: %d)\n",
			batchNum, startIndex, endIndex-1, len(currentBatch))

		// Process the current batch
		subtitles, lastWordIndex, err := c.processBatch(
			currentBatch,
			wordTimings,
			startIndex,
			batchNum,
		)
		if err != nil {
			return nil, err
		}

		// Add the processed subtitles to our result
		allSubtitles = append(allSubtitles, subtitles...)

		// Update the start index for the next batch
		startIndex = lastWordIndex
		batchNum++

		if len(currentBatch) < batchSize {
			break
		}
	}

	// Post-process to ensure consistent transitions between subtitle blocks
	if len(allSubtitles) > 1 {
		for i := 1; i < len(allSubtitles); i++ {
			// Ensure no subtitle end time is after the next subtitle's start time
			if allSubtitles[i-1].EndMs > allSubtitles[i].StartMs {
				allSubtitles[i-1].EndMs = allSubtitles[i].StartMs - 100 // 100ms gap
			}
		}
	}

	return allSubtitles, nil
}

// processBatch processes a batch of word timings and returns the created subtitles,
// along with the index of the last processed word
func (c *Client) processBatch(batch []models.WordTiming, allWords []models.WordTiming,
	startIndex int, batchNum int) ([]models.Subtitle, int, error) {

	// Include the global start index information in the request to maintain proper indexing
	prompt := buildBatchPrompt(batch, startIndex > 0)

	// Add the global start index to help the model understand word positions
	if startIndex > 0 {
		indexInfo := fmt.Sprintf("\nIMPORTANT: These words start at global index %d in the full transcript.\n", startIndex)
		prompt = strings.Replace(prompt, "TRANSCRIPT DATA:", "TRANSCRIPT DATA:"+indexInfo, 1)
	}

	// Debug: Save prompt to file
	if c.debugMode && c.debugDir != "" {
		promptFile := filepath.Join(c.debugDir, fmt.Sprintf("batch_%d_prompt.txt", batchNum))
		if err := os.WriteFile(promptFile, []byte(prompt), 0644); err != nil {
			fmt.Printf("Warning: Failed to save debug prompt: %v\n", err)
		} else if c.debugMode {
			fmt.Printf("Saved prompt to %s\n", promptFile)
		}
	}

	// Create the Gemini API request with temperature parameter
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     c.config.GeminiTemperature,
			"maxOutputTokens": c.config.GeminiMaxTokens,
		},
	}

	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshaling request: %w", err)
	}

	// Make the API request using the specified model
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		c.config.GeminiModel, c.config.GeminiAPIKey)

	if c.debugMode {
		fmt.Printf("Sending request to Gemini API (model: %s)\n", c.config.GeminiModel)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error making API request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading response: %w", err)
	}

	// Debug: Save raw response to file
	if c.debugMode && c.debugDir != "" {
		respFile := filepath.Join(c.debugDir, fmt.Sprintf("batch_%d_response.json", batchNum))
		if err := os.WriteFile(respFile, respBody, 0644); err != nil {
			fmt.Printf("Warning: Failed to save debug response: %v\n", err)
		} else if c.debugMode {
			fmt.Printf("Saved raw response to %s\n", respFile)
		}
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Process the response
	subtitles, lastWordIndex, err := parseBatchResponse(respBody, batch, startIndex)
	if err != nil {
		return nil, 0, err
	}

	// Debug: Log processed subtitles info
	if c.debugMode {
		fmt.Printf("Batch %d: Processed %d words into %d subtitles (last word index: %d)\n",
			batchNum, len(batch), len(subtitles), lastWordIndex)

		// Save processed subtitles to file
		if c.debugDir != "" {
			subtitlesJSON, _ := json.MarshalIndent(subtitles, "", "  ")
			subFile := filepath.Join(c.debugDir, fmt.Sprintf("batch_%d_subtitles.json", batchNum))
			if err := os.WriteFile(subFile, subtitlesJSON, 0644); err != nil {
				fmt.Printf("Warning: Failed to save debug subtitles: %v\n", err)
			} else {
				fmt.Printf("Saved processed subtitles to %s\n", subFile)
			}
		}
	}

	return subtitles, lastWordIndex, nil
}

// Helper function to build the prompt for a batch
func buildBatchPrompt(wordTimings []models.WordTiming, isContinuation bool) string {
	continueText := ""
	if isContinuation {
		continueText = `
IMPORTANT: This is a continuation from a previous batch. 
The first words may be from an incomplete sentence.
Use the "id" field of each word as the absolute index in the transcript.
The st_id values in your response should reference these absolute "id" values.
If the first words continue a sentence from the previous batch, start with those words.
DO NOT repeat sentence beginnings from previous batches, but continue them properly.
`
	}

	prompt := `Convert these word-level transcript timings into subtitle blocks.
Language: Thai, English (few words)
Format: JSON object with sentences array where each element has:
st_id (index of the first word in subtitle), st_ms (start time in milliseconds), 
lw_ms (last word start time in milliseconds), and text (subtitle text).

REQUIREMENTS:
1. General formatting:
   - Combine fragments into complete, grammatical sentences
   - DO Fix spelling, spacing, punctuation and capitalization
   - DO NOT add/remove any words
   - DO NOT translate the content
   - Natural length of sentences are 10-20 words
   - Avoid long sentences with more than 30 words

2. Subtitle structure:
   - Each subtitle should form a complete, natural thought or sentence
   - Each subtitle should end at a natural pause or break point
   - Keep related phrases together in the same subtitle
   - Each subtitle's st_ms must match the first word's start_ms exactly
   - Each subtitle's lw_ms must match the last word's start_ms exactly
   - Continue from the previous batch if this is a continuation

3. Special handling:
   - Look for natural sentence boundaries - DO NOT split mid-sentence
   - Temperature readings (e.g., "อุณหภูมิต่ำสุด 22 องศา อุณหภูมิสูงสุด 39 องศา") must be in their own blocks
   - For long lists (provinces, etc.), DO NOT split into multiple blocks, must be in their own blocks
` + continueText + `   
RETURN FORMAT:
Return ONLY a clean JSON object with exactly this format:
[{"st_id": 0,"st_ms": 123,"lw_ms": 456,"text": "Subtitle text here"},...]

TRANSCRIPT DATA:
`

	wordTimingJSON, _ := json.MarshalIndent(wordTimings, "", "  ")
	return prompt + string(wordTimingJSON)
}

// Helper function to parse the batch response
func parseBatchResponse(respBody []byte, wordTimings []models.WordTiming, startIndex int) ([]models.Subtitle, int, error) {
	var geminiResp Response
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, 0, fmt.Errorf("error parsing API response: %w", err)
	}

	// Validate response structure
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, 0, fmt.Errorf("no content in the API response")
	}

	// Get the JSON content
	jsonContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean up the JSON content to remove any markdown formatting or comments
	jsonContent = cleanJsonContent(jsonContent)

	// Parse the complete response object - using direct array instead of sentences property
	var subtitleInputs []models.SubtitleInput
	if err := json.Unmarshal([]byte(jsonContent), &subtitleInputs); err != nil {
		return nil, 0, fmt.Errorf("failed to parse JSON response: %w\nResponse was: %s", err, jsonContent)
	}

	// Calculate the last word index processed in this batch
	var lastWordIndex int
	if len(subtitleInputs) > 0 {
		// Get StartWordIndex of the last subtitle in the batch
		lastWordIndex = subtitleInputs[len(subtitleInputs)-1].StartWordIndex
	} else {
		// If no sentences were returned, assume all words in the batch were processed
		lastWordIndex = startIndex
	}

	return processSubtitles(subtitleInputs), lastWordIndex, nil
}

// Helper function to clean JSON content from API response
func cleanJsonContent(jsonContent string) string {
	jsonContent = strings.TrimSpace(jsonContent)
	if strings.HasPrefix(jsonContent, "```json") {
		jsonContent = strings.TrimPrefix(jsonContent, "```json")
		if idx := strings.LastIndex(jsonContent, "```"); idx != -1 {
			jsonContent = jsonContent[:idx]
		}
	} else if strings.HasPrefix(jsonContent, "```") {
		jsonContent = strings.TrimPrefix(jsonContent, "```")
		if idx := strings.LastIndex(jsonContent, "```"); idx != -1 {
			jsonContent = jsonContent[:idx]
		}
	}
	return strings.TrimSpace(jsonContent)
}

// Helper function to process subtitles and calculate end times
func processSubtitles(inputSubtitles []models.SubtitleInput) []models.Subtitle {
	var subtitles []models.Subtitle
	for i, sub := range inputSubtitles {
		endMs := 0

		// If we have last_word_start_ms information, use it to estimate display duration
		if sub.LastWordStartMs > 0 {
			// Add a reasonable display duration for the last word (about 1500ms)
			endMs = sub.LastWordStartMs + 1500
		}

		// If this is not the last subtitle, adjust end time based on next subtitle
		if i < len(inputSubtitles)-1 {
			nextStart := inputSubtitles[i+1].StartMs - 100 // 100ms gap between subtitles
			if endMs == 0 || nextStart < endMs {
				endMs = nextStart
			}
		}

		// If endMs is still 0 or too close to start time, set a minimum duration
		if endMs <= sub.StartMs || endMs-sub.StartMs < 1000 {
			endMs = sub.StartMs + 1000 // Minimum 1 second display
		}

		subtitles = append(subtitles, models.Subtitle{
			StartMs: sub.StartMs,
			EndMs:   endMs,
			Text:    sub.Text,
		})
	}
	return subtitles
}
