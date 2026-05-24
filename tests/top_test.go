package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func SearchEventsUpdateTop(t *testing.T) {
	publishSearchEvent(t, "кроссовки", "u1", "s1")
	publishSearchEvent(t, "кроссовки", "u2", "s2")
	publishSearchEvent(t, "носки", "u3", "s3")

	require.Eventually(t, func() bool {
		top := getTop(t)
		return len(top.Items) == 2 &&
			top.Items[0] == (TopItem{Query: "кроссовки", Count: 2}) &&
			top.Items[1] == (TopItem{Query: "носки", Count: 1})
	}, 5*time.Second, 100*time.Millisecond)
}
