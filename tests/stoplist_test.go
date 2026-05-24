package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func StopListHidesTerm(t *testing.T) {
	body := bytes.NewBufferString(`{"term":"кроссовки"}`)
	resp, err := client.Post(address()+"/v1/stoplist", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	require.Eventually(t, func() bool {
		top := getTop(t)
		return len(top.Items) == 1 && top.Items[0] == (TopItem{Query: "носки", Count: 1})
	}, 5*time.Second, 100*time.Millisecond)

	resp, err = request(http.MethodDelete, address()+"/v1/stoplist/"+url.PathEscape("кроссовки"), nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var stopList StopListReply
	resp, err = client.Get(address() + "/v1/stoplist")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stopList))
	require.NotContains(t, stopList.Items, "кроссовки")
}
