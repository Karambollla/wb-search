package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	searchv1 "github.com/Karambollla/wb-search/api/proto/searchv1"
	grpcadapter "github.com/Karambollla/wb-search/internal/adapters/grpc"
	httpadapter "github.com/Karambollla/wb-search/internal/adapters/http"
	"github.com/Karambollla/wb-search/internal/core"
)

type testApp struct {
	ctx         context.Context
	service     *core.Service
	httpHandler http.Handler
	grpcServer  *grpcadapter.Server
}

func prepareApp(t *testing.T) *testApp {
	t.Helper()

	service := core.NewService(core.RealClock{}, core.ServiceConfig{
		Window:         time.Minute,
		BucketDuration: time.Second,
		FraudTTL:       time.Minute,
		MaxTopSize:     10,
	})

	return &testApp{
		ctx:         context.Background(),
		service:     service,
		httpHandler: httpadapter.NewServer(service, nil, nil).Handler(),
		grpcServer:  grpcadapter.NewServer(service),
	}
}

func ingest(t *testing.T, app *testApp, events ...core.SearchEvent) {
	t.Helper()

	for _, event := range events {
		require.NoError(t, app.service.Ingest(app.ctx, event))
	}
}

func rebuildTop(t *testing.T, app *testApp) {
	t.Helper()

	require.NoError(t, app.service.RebuildTop(app.ctx))
}

func HTTPTop(t *testing.T, app *testApp) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/v1/top?limit=10", nil)
	rec := httptest.NewRecorder()
	app.httpHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Items []core.TopItem `json:"items"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	require.Equal(t, expectedTop(), body.Items)
}

func GRPCTop(t *testing.T, app *testApp) {
	t.Helper()

	resp, err := app.grpcServer.GetTop(app.ctx, &searchv1.GetTopRequest{Limit: 10})
	require.NoError(t, err)

	items := make([]core.TopItem, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, core.TopItem{
			Query: item.GetQuery(),
			Count: item.GetCount(),
		})
	}
	require.Equal(t, expectedTop(), items)
}

func expectedTop() []core.TopItem {
	return []core.TopItem{
		{Query: "кроссовки", Count: 2},
		{Query: "носки", Count: 1},
	}
}

func defaultEvents() []core.SearchEvent {
	return []core.SearchEvent{
		{Query: "кроссовки", UserID: "u1", SessionID: "s1"},
		{Query: "кроссовки", UserID: "u2", SessionID: "s2"},
		{Query: "носки", UserID: "u3", SessionID: "s3"},
	}
}
