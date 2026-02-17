package timer

import "time"

func Run(timerCmd <-chan bool, doorTimeOut chan<- bool) {

	for {
		start := <-timerCmd
		if start {
			timer := time.NewTimer(3 * time.Second)
			<-timer.C
			doorTimeOut <- false
		}
	}
}

