package api_test

import "testing"

func TestE2E(t *testing.T) {
	waitService(t)

	t.Run("health", Health)
	t.Run("empty top", EmptyTop)
	t.Run("search events update top", SearchEventsUpdateTop)
	t.Run("stop list hides term", StopListHidesTerm)
	t.Run("metrics", Metrics)
}
