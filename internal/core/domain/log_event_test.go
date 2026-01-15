package domain_test

import (
	"encoding/json"
	"log-guardian/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewLogEvent(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		message  string
		severity domain.LogLevel
		metadata map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "valid log event",
			source:   domain.SOURCE_FILE,
			message:  "Test message",
			severity: domain.LOG_LEVEL_INFO,
			metadata: map[string]interface{}{"key": "value"},
			wantErr:  false,
		},
		{
			name:     "log event with nil metadata",
			source:   domain.SOURCE_STDIN,
			message:  "Another message",
			severity: domain.LOG_LEVEL_ERROR,
			metadata: nil,
			wantErr:  false,
		},
		{
			name:     "log event with empty metadata",
			source:   domain.SOURCE_UNIX,
			message:  "Empty metadata test",
			severity: domain.LOG_LEVEL_DEBUG,
			metadata: map[string]interface{}{},
			wantErr:  false,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := domain.NewLogEvent(tt.source, tt.message, tt.severity, tt.metadata, mockGenerateID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, event)
			} else {
				require.NoError(t, err)
				require.NotNil(t, event)

				assert.Equal(t, tt.source, event.Source)
				assert.Equal(t, tt.message, event.Message)
				assert.Equal(t, tt.severity, event.Severity)

				if tt.metadata == nil {
					assert.Nil(t, event.Metadata)
				} else {
					assert.Equal(t, tt.metadata, event.Metadata)
				}

				assert.NotEmpty(t, event.ID)
				assert.NotZero(t, event.Timestamp)
				assert.WithinDuration(t, time.Now(), event.Timestamp, time.Second)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected *domain.LogLevel
	}{
		{
			name:     "DEBUG level",
			message:  "This is a DEBUG message",
			expected: domain.LOG_LEVEL_DEBUG.Pointer(),
		},
		{
			name:     "INFO level",
			message:  "INFO: Application started",
			expected: domain.LOG_LEVEL_INFO.Pointer(),
		},
		{
			name:     "WARNING level",
			message:  "WARNING: Low disk space",
			expected: domain.LOG_LEVEL_WARNING.Pointer(),
		},
		{
			name:     "ERROR level",
			message:  "ERROR: Connection failed",
			expected: domain.LOG_LEVEL_ERROR.Pointer(),
		},
		{
			name:     "FATAL level",
			message:  "FATAL: System crash",
			expected: domain.LOG_LEVEL_FATAL.Pointer(),
		},
		{
			name:     "no log level",
			message:  "This is a regular message",
			expected: nil,
		},
		{
			name:     "log level at beginning",
			message:  "ERROR Something went wrong",
			expected: domain.LOG_LEVEL_ERROR.Pointer(),
		},
		{
			name:     "log level at end",
			message:  "Something went wrong ERROR",
			expected: domain.LOG_LEVEL_ERROR.Pointer(),
		},
		{
			name:     "multiple log levels (first match)",
			message:  "DEBUG message with ERROR inside",
			expected: domain.LOG_LEVEL_DEBUG.Pointer(),
		},
		{
			name:     "log level as word boundary",
			message:  "This is not an ERRORCODE but ERROR",
			expected: domain.LOG_LEVEL_ERROR.Pointer(),
		},
		{
			name:     "empty message",
			message:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.ParseLogLevel(tt.message)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestLogEvent_AddMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	event, err := domain.NewLogEvent(
		domain.SOURCE_FILE,
		"Test message",
		domain.LOG_LEVEL_INFO,
		map[string]interface{}{"existing": "value"},
		mockGenerateID,
	)
	require.NoError(t, err)

	// Add new metadata
	event.AddMetadata("newKey", "newValue")
	assert.Equal(t, "newValue", event.Metadata["newKey"])
	assert.Equal(t, "value", event.Metadata["existing"])

	// Override existing metadata
	event.AddMetadata("existing", "newValue")
	assert.Equal(t, "newValue", event.Metadata["existing"])
}

func TestLogEvent_GetMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"stringKey": "stringValue",
		"intKey":    42,
		"boolKey":   true,
		"floatKey":  3.14,
		"objectKey": map[string]string{"nested": "value"},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	event, err := domain.NewLogEvent(
		domain.SOURCE_FILE,
		"Test message",
		domain.LOG_LEVEL_INFO,
		metadata,
		mockGenerateID,
	)
	require.NoError(t, err)

	tests := []struct {
		name        string
		key         string
		expectedVal interface{}
		expectedOk  bool
	}{
		{
			name:        "existing string key",
			key:         "stringKey",
			expectedVal: "stringValue",
			expectedOk:  true,
		},
		{
			name:        "existing int key",
			key:         "intKey",
			expectedVal: 42,
			expectedOk:  true,
		},
		{
			name:        "existing bool key",
			key:         "boolKey",
			expectedVal: true,
			expectedOk:  true,
		},
		{
			name:        "existing float key",
			key:         "floatKey",
			expectedVal: 3.14,
			expectedOk:  true,
		},
		{
			name:        "existing object key",
			key:         "objectKey",
			expectedVal: map[string]string{"nested": "value"},
			expectedOk:  true,
		},
		{
			name:        "non-existing key",
			key:         "nonExisting",
			expectedVal: nil,
			expectedOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := event.GetMetadata(tt.key)

			assert.Equal(t, tt.expectedOk, ok)

			if tt.expectedOk {
				assert.Equal(t, tt.expectedVal, val)
			} else {
				assert.Nil(t, val)
			}
		})
	}
}

func TestLogEvent_ToJSON(t *testing.T) {
	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": float64(123),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	event, err := domain.NewLogEvent(
		domain.SOURCE_STDIN,
		"Test JSON message",
		domain.LOG_LEVEL_WARNING,
		metadata,
		mockGenerateID,
	)
	require.NoError(t, err)

	jsonData, err := event.ToJSON()
	require.NoError(t, err)
	require.NotNil(t, jsonData)

	// Parse the JSON back to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, event.ID, parsed["id"])
	assert.Equal(t, event.Source, parsed["source"])
	assert.Equal(t, event.Message, parsed["message"])
	assert.Equal(t, string(event.Severity), parsed["severity"])
	assert.EqualValues(t, metadata, parsed["metadata"])

	// Verify timestamp format (should be RFC3339)
	timestampStr, ok := parsed["timestamp"].(string)
	require.True(t, ok)
	_, err = time.Parse(time.RFC3339, timestampStr)
	require.NoError(t, err)
}

func TestLogEvent_FromJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	originalMetadata := map[string]interface{}{
		"key1": "value1",
		"key2": float64(456),
	}

	originalEvent, err := domain.NewLogEvent(
		domain.SOURCE_UNIX,
		"Original message",
		domain.LOG_LEVEL_ERROR,
		originalMetadata,
		mockGenerateID,
	)
	require.NoError(t, err)

	// Serialize to JSON
	jsonData, err := originalEvent.ToJSON()
	require.NoError(t, err)

	// Create new event and deserialize
	newEvent := &domain.LogEvent{}
	err = newEvent.FromJSON(jsonData)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, originalEvent.ID, newEvent.ID)
	assert.Equal(t, originalEvent.Source, newEvent.Source)
	assert.Equal(t, originalEvent.Message, newEvent.Message)
	assert.Equal(t, originalEvent.Severity, newEvent.Severity)
	assert.Equal(t, originalEvent.Metadata, newEvent.Metadata)

	// Timestamps should be equal (within second precision)
	assert.Equal(t, originalEvent.Timestamp.Unix(), newEvent.Timestamp.Unix())
}

func TestLogEvent_FromJSON_InvalidData(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name:        "invalid JSON",
			jsonData:    `{"id": "123", "source":}`,
			expectError: true,
		},
		{
			name:        "empty JSON",
			jsonData:    "",
			expectError: true,
		},
		{
			name:        "valid JSON but missing fields",
			jsonData:    `{"id": "test"}`,
			expectError: false, // JSON is valid, fields will have zero values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &domain.LogEvent{}
			err := event.FromJSON([]byte(tt.jsonData))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogEvent_JSONRoundTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	originalMetadata := map[string]interface{}{
		"string":  "test",
		"number":  42.0,
		"float":   3.14159,
		"boolean": true,
		"array":   []interface{}{"a", "b", "c"},
		"object": map[string]interface{}{
			"nested": "value",
		},
	}

	originalEvent, err := domain.NewLogEvent(
		domain.SOURCE_FILE,
		"Round trip test message",
		domain.LOG_LEVEL_DEBUG,
		originalMetadata,
		mockGenerateID,
	)
	require.NoError(t, err)

	// Serialize to JSON
	jsonData, err := originalEvent.ToJSON()
	require.NoError(t, err)

	// Deserialize back to LogEvent
	roundTripEvent := &domain.LogEvent{}
	err = roundTripEvent.FromJSON(jsonData)
	require.NoError(t, err)

	// Verify complete round trip
	assert.Equal(t, originalEvent.ID, roundTripEvent.ID)
	assert.Equal(t, originalEvent.Source, roundTripEvent.Source)
	assert.Equal(t, originalEvent.Message, roundTripEvent.Message)
	assert.Equal(t, originalEvent.Severity, roundTripEvent.Severity)
	assert.Equal(t, originalEvent.Metadata, roundTripEvent.Metadata)
	assert.Equal(t, originalEvent.Timestamp.Unix(), roundTripEvent.Timestamp.Unix())
}

func TestLogEvent_Constants(t *testing.T) {
	assert.Equal(t, "stdin", domain.SOURCE_STDIN)
	assert.Equal(t, "file", domain.SOURCE_FILE)
	assert.Equal(t, "unix", domain.SOURCE_UNIX)

	assert.Equal(t, domain.LogLevel("DEBUG"), domain.LOG_LEVEL_DEBUG)
	assert.Equal(t, domain.LogLevel("INFO"), domain.LOG_LEVEL_INFO)
	assert.Equal(t, domain.LogLevel("WARNING"), domain.LOG_LEVEL_WARNING)
	assert.Equal(t, domain.LogLevel("ERROR"), domain.LOG_LEVEL_ERROR)
	assert.Equal(t, domain.LogLevel("FATAL"), domain.LOG_LEVEL_FATAL)
}

func TestLogEvent_WithNilMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGenerateID := domain.NewMockIDGenerator(ctrl)
	mockGenerateID.EXPECT().Generate().AnyTimes().Return("test-id", nil)

	event, err := domain.NewLogEvent(
		domain.SOURCE_STDIN,
		"Test with nil metadata",
		domain.LOG_LEVEL_INFO,
		nil,
		mockGenerateID,
	)
	require.NoError(t, err)

	// Should be able to add metadata to nil map
	event.AddMetadata("newKey", "newValue")

	val, ok := event.GetMetadata("newKey")
	assert.True(t, ok)
	assert.Equal(t, "newValue", val)

	// Original nil map should now be initialized
	assert.NotNil(t, event.Metadata)
	assert.Len(t, event.Metadata, 1)
}
