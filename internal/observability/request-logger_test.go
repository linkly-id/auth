package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/linkly-id/auth/internal/conf"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const apiTestConfig = "../../hack/test.env"

func TestLogger(t *testing.T) {
	var logBuffer bytes.Buffer
	config, err := conf.LoadGlobal(apiTestConfig)
	require.NoError(t, err)

	config.Logging.Level = "info"
	require.NoError(t, ConfigureLogging(&config.Logging))

	// logrus should write to the buffer so we can check if the logs are output correctly
	logrus.SetOutput(&logBuffer)

	// add request id header
	config.API.RequestIDHeader = "X-Request-ID"
	addRequestIdHandler := AddRequestID(config)

	logHandler := NewStructuredLogger(logrus.StandardLogger(), config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "http://example.com/path", nil)
	req.Header.Add("X-Request-ID", "test-request-id")
	require.NoError(t, err)
	addRequestIdHandler(logHandler).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var logs map[string]interface{}
	require.NoError(t, json.NewDecoder(&logBuffer).Decode(&logs))
	require.Equal(t, "api", logs["component"])
	require.Equal(t, http.MethodPost, logs["method"])
	require.Equal(t, "/path", logs["path"])
	require.Equal(t, "test-request-id", logs["request_id"])
	require.NotNil(t, logs["time"])
}

func TestExcludeHealthFromLogs(t *testing.T) {
	var logBuffer bytes.Buffer
	config, err := conf.LoadGlobal(apiTestConfig)
	require.NoError(t, err)

	config.Logging.Level = "info"
	require.NoError(t, ConfigureLogging(&config.Logging))

	// logrus should write to the buffer so we can check if the logs are output correctly
	logrus.SetOutput(&logBuffer)

	logHandler := NewStructuredLogger(logrus.StandardLogger(), config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://example.com/health", nil)
	require.NoError(t, err)
	logHandler.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	require.Empty(t, logBuffer)
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	le := &logEntry{Entry: logrus.NewEntry(logrus.StandardLogger())}
	{
		got := GetLogEntryFromContext(ctx)
		if got == le {
			t.Fatal("exp new log entry")
		}
	}

	ctx = SetLogEntryWithContext(ctx, le)
	{
		got := GetLogEntryFromContext(ctx)
		if got != le {
			t.Fatal("exp new log entry")
		}
	}
}
