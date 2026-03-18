package ordermanagement

import (
	types "heis/types"
)

func setLocalOrderPhase(localOrderview OrderRegister, key OrderKey, newPhase OrderPhase, myID string) OrderInfo {
	localOrder := localOrderview[key]
	localOrder.Phase = newPhase
	localOrder.SeenBy = map[string]bool{myID: true}
	localOrderview[key] = localOrder
	return localOrder
}

// setLocalOrderPhaseKeepSeenBy sets a new phase without resetting SeenBy.
// Use this when transitioning to Served so allAliveHaveSeen works correctly.
func setLocalOrderPhaseKeepSeenBy(localOrderView OrderRegister, key OrderKey, newPhase OrderPhase, myID string) OrderInfo {
	localOrder := localOrderView[key]
	localOrder.Phase = newPhase
	if localOrder.SeenBy == nil {
		localOrder.SeenBy = make(map[string]bool)
	}
	localOrder.SeenBy[myID] = true
	localOrderView[key] = localOrder
	return localOrder
}

func ownerForButton(myID string, button types.ButtonType) string {
	if button == types.BT_Cab {
		return myID
	}
	return ""
}

func makeOrderKey(ownerID string, floor int, button types.ButtonType) OrderKey {
	return OrderKey{
		OwnerID: ownerID,
		Floor:   floor,
		Button:  button,
	}
}

func copySeenByMap(seenBy map[string]bool) map[string]bool {
	copiedSeenBy := make(map[string]bool, len(seenBy))
	for peerID, hasSeen := range seenBy {
		copiedSeenBy[peerID] = hasSeen
	}
	return copiedSeenBy
}

func allAliveHaveSeen(seenBy map[string]bool, alive map[string]bool) bool {
	for id, isAlive := range alive {
		if !isAlive {
			continue
		}
		if !seenBy[id] {
			return false
		}
	}
	return true
}
