package msgfreshness

const maxToleratedCounterJump = 128 //half of uint8 range

func IsSequentiallyNewer(incomingCounter, storedCounter uint8) bool {
	delta := incomingCounter - storedCounter
	return delta > 0 && delta < maxToleratedCounterJump
}

func MissedTicksBetween(currentCounter, lastSeenCounter uint8) int {
	return int(currentCounter - lastSeenCounter)
}