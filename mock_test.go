package stefunny_test

import (
	"context"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mocks struct {
	t           *testing.T
	ctrl        *gomock.Controller
	sfn         *mock.MockSFnService
	eventBridge *mock.MockEventBridgeService
	scheduler   *mock.MockSchedulerService
}

func NewMocks(t *testing.T) *mocks {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := &mocks{
		t:           t,
		ctrl:        ctrl,
		sfn:         mock.NewMockSFnService(ctrl),
		eventBridge: mock.NewMockEventBridgeService(ctrl),
		scheduler:   mock.NewMockSchedulerService(ctrl),
	}
	return m
}

func (m *mocks) Finish() {
	m.t.Helper()
	m.ctrl.Finish()
}

func newMockApp(t *testing.T, path string, m *mocks) *stefunny.App {
	t.Helper()
	l := stefunny.NewConfigLoader(nil, nil)
	ctx := context.Background()
	cfg, err := l.Load(ctx, path)
	require.NoError(t, err)
	m.sfn.EXPECT().SetAliasName("current").Return().AnyTimes()
	app, err := stefunny.New(
		ctx, cfg,
		stefunny.WithSFnService(m.sfn),
		stefunny.WithEventBridgeService(m.eventBridge),
		stefunny.WithSchedulerService(m.scheduler),
	)
	require.NoError(t, err)
	return app
}
