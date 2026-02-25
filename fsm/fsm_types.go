package fsm

import (
	"heis/config"
	om "heis/ordermanagement"
)

var Obstruction bool = false


type Behavior int

const (
	EB_Idle Behavior = iota
	EB_Moving   
	EB_DoorOpen 
)

type Elevator struct {
	Floor int
	Dir config.TravelDirection
	Behavior Behavior
	Orders om.Orders
}

