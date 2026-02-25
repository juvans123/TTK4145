package supervisor

func IsNewer(newCounter, oldCounter uint8) bool {
	delta := newCounter - oldCounter
	return delta > 0 && delta < 128 //Half-wrapper, altså for stor forskjell tolereres ikke
}

func Delta(newCounter, oldCounter uint8) int { //Kan kalle for missedHeartbeats og ta -1 her
	return int(newCounter - oldCounter)
}