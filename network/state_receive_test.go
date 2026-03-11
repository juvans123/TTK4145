// network/state_receive_test.go
package network

import (
	"heis/config"
	"testing"
	"time"
)

// runReceiveTest kaller den faktiske RunStateReceive fra state.go
func runReceiveTest(myID string, messages []config.ElevatorState, waitMs int) []config.ElevatorState {
	// Buffer må være >= antall meldinger siden state.go bruker blokkerende send
	netRx       := make(chan config.ElevatorState, len(messages))
	peerStateCh := make(chan config.ElevatorState, len(messages))

	go RunStateReceive(myID, netRx, peerStateCh)

	for _, msg := range messages {
		netRx <- msg
	}

	time.Sleep(time.Duration(waitMs) * time.Millisecond)

	var received []config.ElevatorState
	for {
		select {
		case st := <-peerStateCh:
			received = append(received, st)
		default:
			return received
		}
	}
}

// --- Test 1: Duplikat med samme counter skal forkastes ---
func TestDuplicateCounterDropped(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 5, Floor: 1, Behaviour: config.BehIdle},
		{ID: "elev2", Counter: 5, Floor: 2, Behaviour: config.BehMoving}, // duplikat
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 1 {
		t.Errorf("Forventet 1 (duplikat skal kastes), fikk %d", len(got))
	}
	if got[0].Floor != 1 {
		t.Errorf("Forventet Floor=1, fikk Floor=%d", got[0].Floor)
	}
}

// --- Test 2: Gammel melding (lavere counter) skal forkastes ---
func TestOldCounterDropped(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 10, Floor: 3, Behaviour: config.BehMoving},
		{ID: "elev2", Counter: 7,  Floor: 0, Behaviour: config.BehIdle},  // gammel
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 1 {
		t.Errorf("Forventet 1 (gammel counter skal kastes), fikk %d", len(got))
	}
	if got[0].Floor != 3 {
		t.Errorf("Forventet Floor=3, fikk Floor=%d", got[0].Floor)
	}
}

// --- Test 3: Stigende counter skal slippe gjennom ---
func TestNewCountersAccepted(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 1, Floor: 0, Behaviour: config.BehIdle},
		{ID: "elev2", Counter: 2, Floor: 1, Behaviour: config.BehMoving},
		{ID: "elev2", Counter: 3, Floor: 2, Behaviour: config.BehMoving},
		{ID: "elev2", Counter: 4, Floor: 3, Behaviour: config.BehDoorOpen},
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 4 {
		t.Errorf("Forventet 4 meldinger, fikk %d", len(got))
	}
}

// --- Test 4: Egne meldinger (myID) skal alltid forkastes ---
func TestOwnMessagesDropped(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev1", Counter: 1, Floor: 2, Behaviour: config.BehMoving}, // eget ID
		{ID: "elev2", Counter: 1, Floor: 1, Behaviour: config.BehIdle},
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 1 {
		t.Errorf("Forventet 1 (egne meldinger kastes), fikk %d", len(got))
	}
	if got[0].ID != "elev2" {
		t.Errorf("Forventet ID=elev2, fikk ID=%s", got[0].ID)
	}
}

// --- Test 5: Tom ID skal forkastes ---
func TestEmptyIDDropped(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "",      Counter: 1, Floor: 0, Behaviour: config.BehIdle},   // tom ID
		{ID: "elev2", Counter: 1, Floor: 2, Behaviour: config.BehMoving},
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 1 {
		t.Errorf("Forventet 1 (tom ID kastes), fikk %d", len(got))
	}
	if got[0].ID != "elev2" {
		t.Errorf("Forventet ID=elev2, fikk ID=%s", got[0].ID)
	}
}

// --- Test 6: Cyclic wrap-around (255 -> 0 -> 1) ---
// uint8: 0 - 255 = 1, delta=1 < 128 → skal aksepteres
func TestCyclicWrapAround(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 254, Floor: 0, Behaviour: config.BehIdle},
		{ID: "elev2", Counter: 255, Floor: 1, Behaviour: config.BehMoving},
		{ID: "elev2", Counter: 0,   Floor: 2, Behaviour: config.BehMoving},  // wrap
		{ID: "elev2", Counter: 1,   Floor: 3, Behaviour: config.BehDoorOpen},
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 4 {
		t.Errorf("Forventet 4 meldinger gjennom wrap-around, fikk %d", len(got))
	}
	if got[3].Floor != 3 {
		t.Errorf("Forventet Floor=3 etter wrap, fikk Floor=%d", got[3].Floor)
	}
}

// --- Test 7: Delta >= 128 skal forkastes (half-wrapper grense) ---
// Counter hopper fra 10 til 200: delta = 190 → for stor → kastes
func TestHalfWrapperRejectsLargeDelta(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 10,  Floor: 1, Behaviour: config.BehIdle},
		{ID: "elev2", Counter: 200, Floor: 3, Behaviour: config.BehMoving}, // delta=190
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 1 {
		t.Errorf("Forventet 1 (stor delta kastes), fikk %d", len(got))
	}
	if got[0].Floor != 1 {
		t.Errorf("Forventet Floor=1, fikk Floor=%d", got[0].Floor)
	}
}

// --- Test 8: Meldinger fra flere peers er uavhengige ---
// elev2 counter=5 slipper gjennom, elev3 counter=5 slipper også gjennom
// duplikat fra elev2 counter=5 kastes
func TestMultiplePeersIndependent(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 5, Floor: 1, Behaviour: config.BehIdle},
		{ID: "elev3", Counter: 5, Floor: 2, Behaviour: config.BehMoving},
		{ID: "elev2", Counter: 5, Floor: 3, Behaviour: config.BehIdle},    // duplikat elev2
		{ID: "elev3", Counter: 6, Floor: 0, Behaviour: config.BehDoorOpen},
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 3 {
		t.Errorf("Forventet 3 meldinger, fikk %d", len(got))
	}
}

// --- Test 9: Første melding fra ukjent peer skal alltid aksepteres (known=false) ---
func TestFirstMessageFromNewPeerAccepted(t *testing.T) {
	msgs := []config.ElevatorState{
		{ID: "elev2", Counter: 200, Floor: 2, Behaviour: config.BehMoving}, // høy counter, men første
	}
	got := runReceiveTest("elev1", msgs, 50)
	if len(got) != 1 {
		t.Errorf("Forventet 1 (første melding alltid akseptert), fikk %d", len(got))
	}
}