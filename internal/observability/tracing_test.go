package observability

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetupTracing(t *testing.T) {
	ctx := context.Background()
	shutdown, err := SetupTracing(ctx, TracingConfig{ServiceName: "test", Writer: io.Discard})
	require.NoError(t, err)
	require.NoError(t, shutdown(ctx))
}
