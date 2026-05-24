package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Metrics(t *testing.T) {
	resp, err := client.Get(address() + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
