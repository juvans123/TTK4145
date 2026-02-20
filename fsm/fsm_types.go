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


/* 
type TravelDirection int

const (
	TD_Up   TravelDirection = 0
	TD_Down TravelDirection = 1
) */
/* 
type Orders struct {   
	Cab  []bool
	Hall [4][2]bool
}

var CurrentOrders Orders = Orders{
	Cab:  make([]bool, 4),
	Hall: [4][2]bool{},
}
 */