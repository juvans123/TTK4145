package supervisor

func IsNewer(newCounter, oldCounter uint8) bool{
	delta := newCounter - oldCounter
	return delta > 0 && delta < 128 //Half-wrapper, altså for stor forskjell tolereres ikke 
}
