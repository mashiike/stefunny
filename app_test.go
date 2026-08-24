package stefunny

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file uses package stefunny (not stefunny_test) because it asserts on
// unexported App/SFnServiceImpl state to directly exercise the lazy service
// construction path (app.sfnService etc.) that New(ctx, cfg) alone takes when
// no With*Service option is given — the path every other test in this
// package bypasses by injecting a mock service up front.

func TestApp_LazyServiceConstruction(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	ctx := context.Background()
	app, err := New(ctx, NewDefaultConfig())
	require.NoError(t, err)
	require.Nil(t, app.sfnSvc, "sfnSvc must not be constructed until first use")

	app.SetAliasName("stage")
	require.Nil(t, app.sfnSvc, "SetAliasName before first use must not force construction")

	svc1, err := app.sfnService(ctx)
	require.NoError(t, err)
	impl, ok := svc1.(*SFnServiceImpl)
	require.True(t, ok)
	require.Equal(t, "stage", impl.aliasName, "alias set before construction must propagate to the lazily-created service")

	svc2, err := app.sfnService(ctx)
	require.NoError(t, err)
	require.Same(t, svc1, svc2, "subsequent calls must reuse the same constructed service")
}

func TestApp_LazyServiceConstruction_Error(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_PROFILE", "stefunny-test-profile-does-not-exist")

	app, err := New(context.Background(), NewDefaultConfig())
	require.NoError(t, err)

	_, err = app.sfnService(context.Background())
	require.ErrorContains(t, err, "failed to get SFN client")
}

func TestApp_LazyServiceConstruction_EventBridgeAndScheduler(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	cases := []struct {
		name   string
		getter func(app *App, ctx context.Context) (any, error)
	}{
		{"eventBridgeService", func(app *App, ctx context.Context) (any, error) { return app.eventBridgeService(ctx) }},
		{"schedulerService", func(app *App, ctx context.Context) (any, error) { return app.schedulerService(ctx) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			app, err := New(ctx, NewDefaultConfig())
			require.NoError(t, err)

			svc1, err := c.getter(app, ctx)
			require.NoError(t, err)
			require.NotNil(t, svc1)

			svc2, err := c.getter(app, ctx)
			require.NoError(t, err)
			require.Same(t, svc1, svc2, "subsequent calls must reuse the same constructed service")
		})
	}
}

func TestApp_LazyServiceConstruction_EventBridgeAndScheduler_Error(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_PROFILE", "stefunny-test-profile-does-not-exist")

	cases := []struct {
		name    string
		getter  func(app *App, ctx context.Context) (any, error)
		wantErr string
	}{
		{"eventBridgeService", func(app *App, ctx context.Context) (any, error) { return app.eventBridgeService(ctx) }, "failed to get EventBridge client"},
		{"schedulerService", func(app *App, ctx context.Context) (any, error) { return app.schedulerService(ctx) }, "failed to get Scheduler client"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app, err := New(context.Background(), NewDefaultConfig())
			require.NoError(t, err)

			_, err = c.getter(app, context.Background())
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}
