package domain

const (
	SOURCE_STDIN = "stdin"
	SOURCE_FILE  = "file"
	SOURCE_UNIX  = "unix"
)

type LogEvent struct {
	Timestamp string
	Source    string
	Severity  string
	Message   string
}
