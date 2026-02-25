package config


type TravelDirection int

const (
	TD_Up   TravelDirection = iota
	TD_Down 
	TD_Stop 
)

type OrderEvent struct{
	Floor int
	Button ButtonType
}

type ClearEvent struct{
	Floor int
	ClearCab bool
    ClearHallUp bool
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

