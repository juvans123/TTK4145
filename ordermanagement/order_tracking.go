package ordermanagement

import (
	"heis/config"
)

func setLocalOrderPhase(view OrderTracker, key OrderKey,newPhase OrderPhase, myID string,) OrderInfo{
	localOrder := view[key]
	localOrder.Phase = newPhase
	localOrder.SeenBy = map[string]bool{myID: true}
	view[key] = localOrder
	return localOrder
}

func ownerForButton(myID string, button config.ButtonType) string {
	if button == config.BT_Cab {
		return myID
	}
	return ""
}

func makeOrderKey(ownerID string, floor int, button config.ButtonType) OrderKey {
	return OrderKey{
		OwnerID: ownerID,
		Floor:   floor,
		Button:  button,
	}
}

func copySeenBy(src map[string]bool) map[string]bool {
	dst := make(map[string]bool)
	for k, v := range src {
		dst[k] = v
	}
	return dst
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