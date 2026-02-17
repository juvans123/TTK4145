package ordermanagement


type Orders struct {   
	Cab  []bool
	Hall [4][2]bool
}

/* var CurrentOrders Orders = Orders{
	Cab:  make([]bool, 4),
	Hall: [4][2]bool{},
}
 */

/* type Dir int
const (
	DirUp Dir = 0
	DirDown Dir = 1
)

type Behaviour int
const (
	EB_Idle Behaviour = 0
	EB_Moving Behaviour = 1
	EB_DoorOpen Behaviour = 2
)

// State som broadcastes (og får fra nettverket)
type ElevatorState struct {
	ID string
	Floor int
	Dir Dir
	Behaviour Behaviour
	CabReq []bool
	Seq uint64
	LastSeen time.Time
}

type Orders struct {
	Cab []bool
	Hall [config.N_FLOORS][2]bool
}
 */

