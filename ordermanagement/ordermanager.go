package ordermanagement

import (
	"fmt"
	"heis/config"
	"Network-go/supervisor"
)

func Run(
	myID         string,
	buttonCh     <-chan config.ButtonEvent,
	clearCh      <-chan config.ClearEvent,
	localStateCh <-chan config.ElevatorState,
	peerStateCh  <-chan config.ElevatorState,
	peerEventCh  <-chan supervisor.PeerEvent,  // direkte fra supervisor
	ordersOutCh  chan<- Orders,
	hallOrderTxCh  chan<- config.ButtonEvent,
	hallOrderRxCh  <-chan config.ButtonEvent,
	clearEventTxCh chan<- config.ClearEvent,
	clearEventRxCh <-chan config.ClearEvent,
) {
	numFloors := config.NumFloors
	ws := NewWorldState(numFloors)
	ws.Alive[myID] = true
	ws.States[myID] = config.ElevatorState{
		ID:          myID,
		Floor:       0,
		Behaviour:   config.BehIdle,
		Direction:   config.DirStop,
		CabRequests: make([]bool, numFloors),
	}

	for _, id := range []string{"elev1", "elev2", "elev3"} {
		if id != myID {
			ws.States[id] = config.ElevatorState{
				ID:          id,
				Floor:       0,
				Behaviour:   config.BehIdle,
				Direction:   config.DirStop,
				CabRequests: make([]bool, numFloors),
			}
			ws.Alive[id] = true
		}
	}

	ordersOutCh <- buildOrders(&ws, myID)

	for {
		changed := false
		select {
		case btn := <-buttonCh:
			if applyButton(&ws, myID, btn) {
				changed = true
			}
			if btn.Button == config.BT_HallUp || btn.Button == config.BT_HallDown {
				select {
				case hallOrderTxCh <- btn:
				default:
				}
			}

		case cl := <-clearCh:
			if applyClear(&ws, myID, cl) {
				changed = true
			}
			if cl.ClearHallUp || cl.ClearHallDown {
				select {
				case clearEventTxCh <- cl:
				default:
				}
			}

		case st := <-localStateCh:
			ws.States[st.ID] = st

		case pst := <-peerStateCh:
			ws.States[pst.ID] = pst

		case event := <-peerEventCh:
			currentAlive, exists := ws.Alive[event.PeerID]
			if !exists || currentAlive != event.Alive {
				ws.Alive[event.PeerID] = event.Alive
				changed = true
			}

		case peerHallBtn := <-hallOrderRxCh:
			if peerHallBtn.Button == config.BT_HallUp || peerHallBtn.Button == config.BT_HallDown {
				if applyButton(&ws, myID, peerHallBtn) {
					changed = true
					fmt.Printf("[%s] Received hall order from peer: Floor %d, Button %d\n",
						myID, peerHallBtn.Floor, peerHallBtn.Button)
				}
			}

		case peerClear := <-clearEventRxCh:
			if peerClear.ClearHallUp || peerClear.ClearHallDown {
				if applyClear(&ws, myID, peerClear) {
					changed = true
					fmt.Printf("[%s] Received clear from peer: Floor %d (HallUp=%v, HallDown=%v)\n",
						myID, peerClear.Floor, peerClear.ClearHallUp, peerClear.ClearHallDown)
				}
			}
		}

		if changed {
			ordersOutCh <- buildOrders(&ws, myID)
		}
	}
}