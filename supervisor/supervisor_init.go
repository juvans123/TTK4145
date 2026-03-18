package supervisor

import "time"

type SupervisorInit struct {
	MyID              string
	TickInterval      time.Duration
	TicksBeforeSuspected  int
	ConsensusRequired int
}


func NewSupervisorInit(myID string) SupervisorInit {
	return SupervisorInit{
		MyID:              myID,
		TickInterval:      100 * time.Millisecond,
		TicksBeforeSuspected:  15, 
		ConsensusRequired: 2,
	}
}