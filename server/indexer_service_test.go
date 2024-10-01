package server

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/require"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

var (
	failedResponse = &coretypes.ResultStatus{
		SyncInfo: coretypes.SyncInfo{
			LatestBlockHeight: 0,
		},
	}

	successfulResponse = &coretypes.ResultStatus{
		SyncInfo: coretypes.SyncInfo{
			LatestBlockHeight: 1,
		},
	}
)

type statusClientMock struct {
	// retries left before success response
	retriesLeft uint
}

func newStatusClientMock(retriesLeft uint) *statusClientMock {
	return &statusClientMock{
		retriesLeft: retriesLeft,
	}
}

func (m *statusClientMock) Status(context.Context) (*coretypes.ResultStatus, error) {
	if m.retriesLeft == 0 {
		return successfulResponse, nil
	}

	m.retriesLeft--
	return failedResponse, nil
}

func TestWaitUntilClientReady(t *testing.T) {
	for _, tc := range []struct {
		desc        string
		retriesLeft uint
	}{
		{
			desc:        "return successful response right away",
			retriesLeft: 0,
		},
		{
			desc:        "return successful response after one retry",
			retriesLeft: 1,
		},
		{
			desc:        "return successful response after 10 retries",
			retriesLeft: 10,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ctxb := context.Background()
			mock := newStatusClientMock(tc.retriesLeft)

			err := waitUntilClientReady(ctxb, mock, backoff.NewConstantBackOff(time.Nanosecond))
			require.NoError(t, err)
			require.Equal(t, uint(0), mock.retriesLeft)
		})
	}
}

func TestWaitUntilClientReadyTimeout(t *testing.T) {
	ctxb := context.Background()
	// create a mock client which always returns an error
	mock := newStatusClientMock(math.MaxUint)

	exponentialBackOff := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Millisecond),
		backoff.WithMaxInterval(time.Millisecond*10),
		backoff.WithMaxElapsedTime(time.Millisecond*100),
	)

	err := waitUntilClientReady(ctxb, mock, exponentialBackOff)
	// make sure error is propagated in case of timeout
	require.Error(t, err)
	require.Contains(t, err.Error(), "node isn't ready, possibly in state sync process")
}
