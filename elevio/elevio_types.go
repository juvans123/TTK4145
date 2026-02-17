package elevio

import (
	"net"
	"sync"
	"time"
)


const _pollRate = 20 * time.Millisecond

var _initialized bool = false
var _numFloors int = 4
var _mtx sync.Mutex
var _conn net.Conn

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down MotorDirection = -1
	MD_Stop MotorDirection = 0
)


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

// Kommandoer
type ElevioCmdType int

const (
	SetMotorDirection ElevioCmdType = iota
	SetButtonLamp
	SetFloorIndicator
	SetDoorLamp
	SetStopLamp
)

type DriverCmd struct {
	Type ElevioCmdType

	MotorDir MotorDirection

	Button ButtonType
	Floor  int
	Value  bool

	IndicatorFloor int
}
