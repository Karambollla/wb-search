package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

const (
	defaultAddress = "http://localhost:8080"
	defaultNATSURL = "nats://localhost:4222"
)

var client = http.Client{
	Timeout: 10 * time.Second,
}

type TopItem struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

type TopReply struct {
	Items []TopItem `json:"items"`
}

type StopListReply struct {
	Items []string `json:"items"`
}

func TestPreflight(t *testing.T) {
	waitService(t)
}

func Health(t *testing.T) {
	resp, err := client.Get(address() + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func EmptyTop(t *testing.T) {
	top := getTop(t)
	require.Empty(t, top.Items)
}

func getTop(t *testing.T) TopReply {
	t.Helper()

	resp, err := client.Get(address() + "/v1/top?limit=10")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var top TopReply
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&top))
	return top
}

func publishSearchEvent(t *testing.T, query, userID, sessionID string) {
	t.Helper()

	nc, err := nats.Connect(natsURL())
	require.NoError(t, err)
	defer nc.Close()

	payload, err := json.Marshal(map[string]string{
		"query":      query,
		"user_id":    userID,
		"session_id": sessionID,
	})
	require.NoError(t, err)
	require.NoError(t, nc.Publish("search.events", payload))
	require.NoError(t, nc.Flush())
}

func waitService(t *testing.T) {
	t.Helper()

	require.Eventually(t, func() bool {
		resp, err := client.Get(address() + "/healthz")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 30*time.Second, 500*time.Millisecond)
}

func request(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func address() string {
	if value := os.Getenv("ADDRESS"); value != "" {
		return value
	}
	return defaultAddress
}

func natsURL() string {
	if value := os.Getenv("NATS_URL"); value != "" {
		return value
	}
	return defaultNATSURL
}
