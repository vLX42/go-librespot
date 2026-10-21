//go:build test_unit

package daemon

import (
	"testing"

	"github.com/devgianlu/go-librespot/dealer"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// needsStateUpdatesMessage is a cluster update naming another device active,
// carrying needs as its needs_state_updates setting (nil leaves it out).
func needsStateUpdatesMessage(t *testing.T, needs *bool) dealer.Message {
	t.Helper()

	payload, err := proto.Marshal(&connectpb.ClusterUpdate{
		Cluster: &connectpb.Cluster{
			ActiveDeviceId:    otherDevice,
			PlayerState:       &connectpb.PlayerState{},
			NeedsStateUpdates: needs,
		},
	})
	require.NoError(t, err)

	return dealer.Message{Uri: "hm://connect-state/v1/cluster", Payload: payload}
}

func newNeedsStateUpdatesPlayer(t *testing.T) *AppPlayer {
	t.Helper()

	p := newTestAppPlayer(t)
	p.app.cfg = &Config{}
	p.app.deviceId = thisDevice
	p.hasSpotConnId = true
	return p
}

func queuedReasons(p *AppPlayer) []connectpb.PutStateReason {
	var reasons []connectpb.PutStateReason
	for _, push := range drain(p.statePush) {
		reasons = append(reasons, push.reason)
	}
	return reasons
}

// While the backend needs no updates, an inactive device leaves out the
// player-state and volume pushes, but still reports every transition.
func TestNeedsStateUpdatesOffSkipsStateAndVolume(t *testing.T) {
	p := newNeedsStateUpdatesPlayer(t)
	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, proto.Bool(false))))

	p.pushState(connectpb.PutStateReason_PLAYER_STATE_CHANGED)
	p.pushState(connectpb.PutStateReason_VOLUME_CHANGED)
	p.pushState(connectpb.PutStateReason_NEW_DEVICE)
	p.pushState(connectpb.PutStateReason_BECAME_INACTIVE)

	require.Equal(t, []connectpb.PutStateReason{
		connectpb.PutStateReason_NEW_DEVICE,
		connectpb.PutStateReason_BECAME_INACTIVE,
	}, queuedReasons(p))
}

// The controllers follow playback through the active device, so it reports
// whatever the setting says.
func TestNeedsStateUpdatesOffStillReportsWhileActive(t *testing.T) {
	p := newNeedsStateUpdatesPlayer(t)
	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, proto.Bool(false))))
	p.state.setActive(true)

	p.pushState(connectpb.PutStateReason_PLAYER_STATE_CHANGED)

	require.Equal(t, []connectpb.PutStateReason{connectpb.PutStateReason_PLAYER_STATE_CHANGED}, queuedReasons(p))
}

// Asking for updates again gets the current state at once.
func TestNeedsStateUpdatesOnSendsStateAtOnce(t *testing.T) {
	p := newNeedsStateUpdatesPlayer(t)
	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, proto.Bool(false))))
	require.Empty(t, queuedReasons(p))

	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, proto.Bool(true))))
	require.Equal(t, []connectpb.PutStateReason{connectpb.PutStateReason_PLAYER_STATE_CHANGED}, queuedReasons(p))

	// Already on: a repeat of the setting sends nothing more.
	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, proto.Bool(true))))
	require.Empty(t, queuedReasons(p))
}

// A cluster that leaves the setting out keeps whatever was set before.
func TestNeedsStateUpdatesAbsentChangesNothing(t *testing.T) {
	p := newNeedsStateUpdatesPlayer(t)
	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, nil)))
	p.pushState(connectpb.PutStateReason_PLAYER_STATE_CHANGED)
	require.Len(t, queuedReasons(p), 1, "reports by default")

	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, proto.Bool(false))))
	require.NoError(t, p.handleDealerMessage(needsStateUpdatesMessage(t, nil)))
	p.pushState(connectpb.PutStateReason_PLAYER_STATE_CHANGED)
	require.Empty(t, queuedReasons(p), "still off")
}

// The put state response carries the cluster too.
func TestNeedsStateUpdatesFromPutStateResponse(t *testing.T) {
	p := newNeedsStateUpdatesPlayer(t)

	p.applyStatePushResult(statePushResult{needsStateUpdates: proto.Bool(false)})
	p.pushState(connectpb.PutStateReason_VOLUME_CHANGED)
	require.Empty(t, queuedReasons(p))
}
