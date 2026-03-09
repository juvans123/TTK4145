package ordermanagement
import (
	"heis/config"
)

type Orders struct {   
	Cab  []bool
	Hall [][]bool
}



type OrderPhase int
const (
    NoOrder OrderPhase = iota
    Unconfirmed
    Confirmed
    Served
)

type OrderMsg struct {
    OwnerID string
    Floor int
    Button config.ButtonType
    SeenBy map[string]bool
    Phase OrderPhase
    //Counter int
}

type OrderKey struct{
    OwnerID string
    Floor int
    Button config.ButtonType
}


type OrderInfo struct{
    SeenBy map[string]bool
    Phase OrderPhase
    //counter
}

type OrderTracker map[OrderKey]OrderInfo


type WorldState struct {
    ConfirmedHallOrders [config.N_FLOORS][2]bool
    ConfirmedCabOrders map[string][]bool
    States map[string]config.ElevatorState
    Alive map[string]bool
    //OrderTracker OrderTracker
    // LastSeen map[string]time.Time
}


type AssignerInput struct {
	HallRequests [][]bool                        `json:"hallRequests"`
	States       map[string]config.ElevatorState `json:"states"`
}

type AssignerOutput map[string][][]bool