// Package tests is a collection of epochtime implementation test cases.
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	consensus "github.com/oasisprotocol/oasis-core/go/consensus/api"
	"github.com/oasisprotocol/oasis-core/go/epochtime/api"
)

const recvTimeout = 5 * time.Second

// EpochtimeSetableImplementationTest exercises the basic functionality of
// a setable (mock) epochtime backend.
func EpochtimeSetableImplementationTest(t *testing.T, backend api.Backend) {
	require := require.New(t)

	// Ensure that the backend is setable.
	require.Implements((*api.SetableBackend)(nil), backend, "epoch time backend is mock")
	timeSource := (backend).(api.SetableBackend)

	parameters, err := timeSource.ConsensusParameters(context.Background(), consensus.HeightLatest)
	require.NoError(err, "ConsensusParameters")
	require.True(parameters.DebugMockBackend, "expected debug backend")

	epoch, err := timeSource.GetEpoch(context.Background(), consensus.HeightLatest)
	require.NoError(err, "GetEpoch")

	var e api.EpochTime

	ch, sub := timeSource.WatchEpochs()
	defer sub.Close()
	select {
	case e = <-ch:
		require.Equal(epoch, e, "WatchEpochs initial")
	case <-time.After(recvTimeout):
		t.Fatalf("failed to receive current epoch on WatchEpochs")
	}

	latestCh, subCh := timeSource.WatchLatestEpoch()
	defer subCh.Close()
	select {
	case e = <-latestCh:
		require.Equal(epoch, e, "WatchLatestEpoch initial")
	case <-time.After(recvTimeout):
		t.Fatalf("failed to receive current epoch on WatchLatestEpoch")
	}

	epoch++
	err = timeSource.SetEpoch(context.Background(), epoch)
	require.NoError(err, "SetEpoch")

	select {
	case e = <-ch:
		require.Equal(epoch, e, "WatchEpochs after set")
	case <-time.After(recvTimeout):
		t.Fatalf("failed to receive epoch notification after transition")
	}

	select {
	case e = <-latestCh:
		require.Equal(epoch, e, "WatchLatestEpoch after set")
	case <-time.After(recvTimeout):
		t.Fatalf("failed to receive latest epoch after transition")
	}

	e, err = timeSource.GetEpoch(context.Background(), consensus.HeightLatest)
	require.NoError(err, "GetEpoch after set")
	require.Equal(epoch, e, "GetEpoch after set, epoch")
}

// MustAdvanceEpoch advances the epoch by the specified increment, and returns
// the new epoch.
func MustAdvanceEpoch(t *testing.T, backend api.SetableBackend, increment uint64) api.EpochTime {
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), recvTimeout)
	defer cancel()

	epoch, err := backend.GetEpoch(ctx, consensus.HeightLatest)
	require.NoError(err, "GetEpoch")

	for i := uint64(0); i < increment; i++ {
		ctx, cancel = context.WithTimeout(context.Background(), recvTimeout)
		defer cancel()
		epoch++
		err = backend.SetEpoch(ctx, epoch)
		require.NoError(err, "SetEpoch")
	}

	return epoch
}
