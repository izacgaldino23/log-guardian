package domain

const (
	SOURCE_STDIN = "stdin"
)

type LogEvent struct {
	Timestamp string
	Source    string
	Severity  string
	Message   string
}
