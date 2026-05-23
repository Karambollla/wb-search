//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natsservertest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	natsadapter "github.com/Karambollla/wb-search/internal/adapters/nats"
)

func TestSearchEventFlowThroughNATSHTTPAndGRPC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	natsServer := startNATSServer(t)
	defer natsServer.Shutdown()

	app := prepareApp(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	subscriber, err := natsadapter.NewSubscriber(natsServer.ClientURL(), "search.events", app.service, log)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, subscriber.Close())
	}()
	go subscriber.Start(ctx)

	nc, err := nats.Connect(natsServer.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	publishSearchEvent(t, nc, "кроссовки", "u1", "s1")
	publishSearchEvent(t, nc, "кроссовки", "u2", "s2")
	publishSearchEvent(t, nc, "носки", "u3", "s3")
	require.NoError(t, nc.Flush())

	require.Eventually(t, func() bool {
		rebuildTop(t, app)
		top, err := app.service.GetTop(app.ctx, 10)
		if err != nil {
			return false
		}
		return len(top) >= 2 && top[0].Query == "кроссовки" && top[0].Count == 2
	}, 2*time.Second, 10*time.Millisecond)

	t.Run("http returns top", func(t *testing.T) {
		HTTPTop(t, app)
	})
	t.Run("grpc returns top", func(t *testing.T) {
		GRPCTop(t, app)
	})
}

func startNATSServer(t *testing.T) *server.Server {
	t.Helper()

	return natsservertest.RunRandClientPortServer()
}

func publishSearchEvent(t *testing.T, nc *nats.Conn, query, userID, sessionID string) {
	t.Helper()

	payload, err := json.Marshal(map[string]string{
		"query":      query,
		"user_id":    userID,
		"session_id": sessionID,
	})
	require.NoError(t, err)
	require.NoError(t, nc.Publish("search.events", payload))
}
