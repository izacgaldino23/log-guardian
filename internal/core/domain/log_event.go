package domain

const (
	SOURCE_STDIN = "stdin"
	SOURCE_FILE  = "file"
)

type LogEvent struct {
	Timestamp string
	Source    string
	Severity  string
	Message   string
}
