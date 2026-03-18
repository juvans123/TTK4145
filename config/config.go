package config

import "time"

const N_FLOORS = 4

type TravelDirection int
const (
	TD_Up TravelDirection = iota
	TD_Down
)

type OrderEvent struct {
	Floor  int
	Button ButtonType
}

type ClearEvent struct {
	Floor         int
	Cab      bool
	HallUp   bool
	HallDown bool
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
	Counter     uint8
	Behaviour   Behaviour `json:"behaviour"`
	Floor       int       `json:"floor"`
	Direction   Direction `json:"direction"`
	CabRequests []bool    `json:"cabRequests"`
	Obstructed  bool
	Immobile    bool
}


type PeerAliveness struct {
	ID string
	IsAlive  bool
}

type LightState struct {
	Hall [N_FLOORS][2]bool
	Cab  []bool
}
type SupervisorConfig struct {
	TickInterval      time.Duration
	SuspectThreshold  int
	ConsensusRequired int
}

func DefaultSupervisorConfig() SupervisorConfig {
	return SupervisorConfig{
		TickInterval:      100 * time.Millisecond,
		SuspectThreshold:  15,
		ConsensusRequired: 2,
	}
}

type NetworkConfig struct {
	StatePort     int
	HallOrderPort int
	HeartbeatPort int
}

func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		StatePort:     16570,
		HallOrderPort: 16571,
		HeartbeatPort: 16647,
	}
}

const maxToleratedCounterJump = 128 

func IsSequentiallyNewer(incomingCounter, storedCounter uint8) bool {
	delta := incomingCounter - storedCounter
	return delta > 0 && delta < maxToleratedCounterJump
}

func MissedTicksBetween(currentCounter, lastSeenCounter uint8) int {
	return int(currentCounter - lastSeenCounter)
}