package ordermanagement

import(e "heis/elevio")


func Run(button_pressed <-chan e.ButtonEvent){

	for{
		/* select{
		case button := <-button_pressed: 


		} */
	
	
	}




}


















/* type Channels struct {
	HallPressed <-chan HallButtonEvent
	CabPressed <-chan CabButtonEvent
	HallCleared <-chan ClearHallEvent
	CabCleared <-chan ClearCabEvent

	//PeerUpdate <-chan PeerUpdateWrap
	RemoteState <-chan ElevatorState

	OrdersOut chan<- Orders
}

// type PeerUpdateWrap

func Run(myID string, ch Channels){
	ws := WorldState{
		HallRequests: // FIKS DETTE
		ElevStates: map[string]ElevatorState{},
		Alive: map[string]bool{},
	}
	ws.Alive[myID] = true
	// mye greier her med triggerReassign

	ch.OrdersOut <- Orders{Cab: append([]bool(nil), myCab...)}

}
 */