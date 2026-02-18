package fsm

/* import (
	c "heis/config") */

var Obstruction bool = false
var currentFloor = -1 //husk å initialiser


type State int

const (
	EB_Idle     State = 0
	EB_Moving   State = 1
	EB_DoorOpen State = 2
)


