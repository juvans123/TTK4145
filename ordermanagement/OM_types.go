package ordermanagement
import (
	"heis/config"
)

type Orders struct {   
	Cab  []bool
	Hall [][]bool
}

type HallPhase int
const (
    HallNone HallPhase = iota
    HallUnconfirmed
    HallConfirmed
)

type HallOrderState struct {
    Phase HallPhase
    SeenBy map[string]uint8
}

type HallObservation struct {
	FromID string
	Floor int
	Button config.ButtonType
}

type WorldState struct {
    NumFloors int

    // Input til assigner:
    // hallRequests[f][0]=HallUp, hallRequests[f][1]=HallDown (eller motsatt, men vær konsekvent)
    HallRequests [][]HallOrderState
    States map[string]config.ElevatorState
    Alive map[string]bool

    // Valgfritt: siste gang vi fikk state fra hver peer
    // LastSeen map[string]time.Time
}
