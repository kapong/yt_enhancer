package models

import "encoding/xml"

// XML Structure definitions to parse the timedtext format
type TimedText struct {
	XMLName xml.Name `xml:"timedtext"`
	Body    Body     `xml:"body"`
}

type Body struct {
	Paragraphs []Paragraph `xml:"p"`
}

type Paragraph struct {
	Time      string     `xml:"t,attr"`
	Duration  string     `xml:"d,attr"`
	A         string     `xml:"a,attr"`
	W         string     `xml:"w,attr"`
	Content   string     `xml:",chardata"`
	Sentences []Sentence `xml:"s"`
}

type Sentence struct {
	Time string `xml:"t,attr"`
	Ac   string `xml:"ac,attr"`
	Text string `xml:",chardata"`
}

// WordTiming represents a single word with its timing information
type WordTiming struct {
	ID        int    `json:"id"`       // Global index of the word in the transcript
	Word      string `json:"word"`     // The word text
	StartTime int    `json:"start_ms"` // Start time in milliseconds
}

// Subtitle represents a subtitle block with start time, end time, and text
type Subtitle struct {
	StartMs int    `json:"start_ms"`
	EndMs   int    `json:"end_ms"`
	Text    string `json:"text"`
}

// SubtitleInput is used to parse the API response
type SubtitleInput struct {
	StartWordIndex  int    `json:"st_id"`
	StartMs         int    `json:"st_ms"`
	LastWordStartMs int    `json:"lw_ms"`
	Text            string `json:"text"`
	Incomplete      bool   `json:"incomplete"`
}
