package fsm

import (
	"heis/config"
	"heis/elevio"
	om "heis/ordermanagement"
)

type Behavior int
const (
	EB_Idle Behavior = iota
	EB_Moving   
	EB_DoorOpen 
)

type Elevator struct {
	Floor int
	Dir elevio.MotorDirection
	TravelDir config.TravelDirection
	Behavior Behavior
	Orders om.Orders
	Obstructed bool
	Immobile bool
}

