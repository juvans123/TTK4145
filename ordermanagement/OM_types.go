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
    HallRequests [][]HallOrderState
    States map[string]config.ElevatorState
    Alive map[string]bool
    // LastSeen map[string]time.Time
}


type AssignerInput struct {
	HallRequests [][]bool                        `json:"hallRequests"`
	States       map[string]config.ElevatorState `json:"states"`
}

type AssignerOutput map[string][][]bool