package tests

import (
	"testing"
)

func TestSearchFlowThroughCoreHTTPAndGRPC(t *testing.T) {
	app := prepareApp(t)
	ingest(t, app, defaultEvents()...)
	rebuildTop(t, app)

	t.Run("http returns top", func(t *testing.T) {
		HTTPTop(t, app)
	})
	t.Run("grpc returns top", func(t *testing.T) {
		GRPCTop(t, app)
	})
}
