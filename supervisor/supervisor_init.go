package supervisor

import "time"

type SupervisorConfig struct {
	TickInterval      time.Duration
	SuspectThreshold  int
	ConsensusRequired int
}


func DefaultSupervisorConfig() SupervisorConfig {
	return SupervisorConfig{
		TickInterval:      100 * time.Millisecond,
		SuspectThreshold:  15,
		ConsensusRequired: 2,
	}
}

type Config struct {
	MyID             string
	SupervisorConfig SupervisorConfig
}

func NewConfig(myID string) Config {
	return Config{
		MyID:             myID,
		SupervisorConfig: DefaultSupervisorConfig(),
	}
}

func (c Config) tickInterval() time.Duration { return c.SupervisorConfig.TickInterval }
func (c Config) suspectThreshold() int     { return c.SupervisorConfig.SuspectThreshold }
func (c Config) consensusRequired() int    { return c.SupervisorConfig.ConsensusRequired }