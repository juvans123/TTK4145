package fsm

/* import (
	c "heis/config") */

var Obstruction bool = false
var currentFloor int = -1 //husk å initialiser


type State int

const (
	EB_Idle     State = 0
	EB_Moving   State = 1
	EB_DoorOpen State = 2
)



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