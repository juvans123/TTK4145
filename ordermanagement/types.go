package ordermanagement

import "time"

type Direction string
const (
	DirUp Direction = "up"
	DirDown Direction = "down"
	DirStop Direction = "stop"
)

type Behaviour string
const (
	BehIdle Behaviour = "idle"
	BehMoving Behaviour = "moving"
	BehDoorOpen Behaviour = "doorOpen"
)

// State som broadcastes (og får fra nettverket)
/* type ElevatorState struct {
	ID string
	Floor int
	Dir Dir
	Behaviour Behaviour
	CabReq []bool
	Seq uint64
	LastSeen time.Time
} */

type ElevatorState struct {
    Behaviour Behaviour `json:"behaviour"`
    Floor     int       `json:"floor"`
    Direction Direction `json:"direction"`
    CabRequests []bool  `json:"cabRequests"`
}

type Orders struct {
	CabRequests []bool
	Hall [config.N_FLOORS][2]bool
}

type AssignerInput struct {
	HallRequests [][]bool                 `json:"hallRequests"`
	States       map[string]ElevatorState `json:"states"`
}

type AssignerOutput map[string][][]bool
