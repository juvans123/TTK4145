package fsm

import (
	"heis/elevio"
	om "heis/ordermanagement"
	"heis/types"
)

type Behavior int

const (
	EB_Idle Behavior = iota
	EB_Moving
	EB_DoorOpen
)

type Elevator struct {
	Floor        int
	Dir          elevio.MotorDirection
	TravelDir    types.TravelDirection
	Behavior     Behavior
	Orders       om.Orders
	IsObstructed bool
	IsImmobile   bool
}
