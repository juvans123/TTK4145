package config
import "time"

const N_FLOORS = 4
type TravelDirection int

const (
	TD_Up   TravelDirection = iota
	TD_Down 
)

type OrderEvent struct {
	Floor  int
	Button ButtonType
}

type ClearEvent struct {
	Floor         int
	ClearCab      bool
	ClearHallUp   bool
	ClearHallDown bool
}

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type Behaviour string

const (
	BehIdle     Behaviour = "idle"
	BehMoving   Behaviour = "moving"
	BehDoorOpen Behaviour = "doorOpen"
)

type Direction string

const (
	DirUp   Direction = "up"
	DirDown Direction = "down"
	DirStop Direction = "stop"
)

type ElevatorState struct {
	ID          string
	Behaviour   Behaviour `json:"behaviour"`
	Floor       int       `json:"floor"`
	Direction   Direction `json:"direction"`
	CabRequests []bool    `json:"cabRequests"`
}

type PeerUpdate struct {
	ID    string
	Alive bool
}

type SupervisorConfig struct {
	TickInterval time.Duration
    SuspectThreshold int
	ConsensusRequired int
    HeartbeatPort int
}

func DefaultSupervisorConfig() SupervisorConfig {
	return SupervisorConfig{
		TickInterval:     3 * time.Second,
		SuspectThreshold: 5,
		ConsensusRequired: 2,
        HeartbeatPort: 15647,
	}
}
