package ordermanagement
import (
	"heis/types"
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
    Button types.ButtonType
    SeenBy map[string]bool
    Phase OrderPhase
}

type OrderKey struct{
    OwnerID string
    Floor int
    Button types.ButtonType
}


type OrderInfo struct{
    SeenBy map[string]bool
    Phase OrderPhase
}

type OrderRegister map[OrderKey]OrderInfo


type WorldState struct {
    ConfirmedHallOrders [types.N_FLOORS][2]bool
    ConfirmedCabOrders map[string][]bool
    States map[string]types.ElevatorState
    Alive map[string]bool
    //OrderTracker OrderTracker
    // LastSeen map[string]time.Time
}


type AssignerInput struct {
	HallRequests [][]bool                        `json:"hallRequests"`
	States       map[string]types.ElevatorState `json:"states"`
}

type AssignerOutput map[string][][]bool