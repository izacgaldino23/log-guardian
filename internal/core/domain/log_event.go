package domain

import (
	"encoding/json"
	"regexp"
	"time"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type IDGenerator interface {
	Generate() (string, error)
}

const (
	SOURCE_STDIN = "stdin"
	SOURCE_FILE  = "file"
	SOURCE_UNIX  = "unix"

	LOG_LEVEL_DEBUG   LogLevel = "DEBUG"
	LOG_LEVEL_INFO    LogLevel = "INFO"
	LOG_LEVEL_WARNING LogLevel = "WARNING"
	LOG_LEVEL_ERROR   LogLevel = "ERROR"
	LOG_LEVEL_FATAL   LogLevel = "FATAL"
)

type LogLevel string

var regexLogLevel = regexp.MustCompile(`\b(DEBUG|INFO|WARNING|ERROR|FATAL)\b`)

type LogEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Severity  LogLevel               `json:"severity"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func NewLogEvent(source string, message string, severity LogLevel, metadata map[string]interface{}, idGen IDGenerator) (*LogEvent, error) {
	id, err := idGen.Generate()
	if err != nil {
		return nil, err
	}

	return &LogEvent{
		Source:    source,
		Severity:  severity,
		Message:   message,
		Metadata:  metadata,
		Timestamp: time.Now(),
		ID:        id,
	}, nil
}

// ParseLogLevel parses the log level from the message
func ParseLogLevel(message string) *LogLevel {
	matches := regexLogLevel.FindStringSubmatch(message)
	if len(matches) > 1 {
		level := LogLevel(matches[0])
		return &level
	}

	return nil
}

// AddMetadata add metadata in the log
func (le *LogEvent) AddMetadata(key, value string) {
	if le.Metadata == nil {
		le.Metadata = make(map[string]interface{})
	}
	le.Metadata[key] = value
}

// GetMetadata get metadata from the log
func (le LogEvent) GetMetadata(key string) (any, bool) {
	value, ok := le.Metadata[key]
	return value, ok
}

// ToJSON serializes the log event to JSON
func (le LogEvent) ToJSON() ([]byte, error) {
	return json.Marshal(le)
}

// FromJSON deserializes the log event from JSON
func (le *LogEvent) FromJSON(data []byte) error {
	return json.Unmarshal(data, le)
}

// Pointer returns a pointer to the log level
func (ll LogLevel) Pointer() *LogLevel {
	return &ll
}
