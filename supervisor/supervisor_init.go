package supervisor

import "time"

type SupervisorParams struct {
	TickInterval      time.Duration
	SuspectThreshold  int
	ConsensusRequired int
}

func defaultSupervisorParams() SupervisorParams {
	return SupervisorParams{
		TickInterval:      100 * time.Millisecond,
		SuspectThreshold:  15,
		ConsensusRequired: 2,
	}
}

type SupervisorInit struct {
	MyID   string
	Params SupervisorParams
}

func NewSupervisorInit(myID string) SupervisorInit {
	return SupervisorInit{
		MyID:   myID,
		Params: defaultSupervisorParams(),
	}
}

func (s SupervisorInit) tickInterval() time.Duration { return s.Params.TickInterval }
func (s SupervisorInit) suspectThreshold() int       { return s.Params.SuspectThreshold }
func (s SupervisorInit) consensusRequired() int      { return s.Params.ConsensusRequired }
