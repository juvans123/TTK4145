package timer

import "time"


 func Run(timerCmd <-chan bool, doorTimeOut chan<- bool) {

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	for {
		select {

		case <-timerCmd:
			timer.Reset(3 * time.Second)

		case <-timer.C:
			doorTimeOut <- false
		}
	}
} 

/* func Run(timerCmd <-chan bool, doorTimeOut chan<- bool) {
	const doorOpenTime = 3 * time.Second
	var timer *time.Timer
	var timerActive bool

	for {
		select {
		case cmd := <-timerCmd:
			if cmd { // start/restart timer
				if timerActive {
					// Stoppe gammel timer
					if !timer.Stop() {
						// Timer gikk akkurat ut, tøm kanalen
						select {
						case <-timer.C:
						default:
						}
					}
				} else {
					timerActive = true
				}
				// Start ny timer
				timer = time.NewTimer(doorOpenTime)
			}

		case <-func() <-chan time.Time {
			if timerActive && timer != nil {
				return timer.C
			}
			// Returner aldri mottatt kanal hvis timer ikke aktiv
			return make(chan time.Time) // blokkerer, gjør ingen ting
		}():
			doorTimeOut <- false
			timerActive = false
		}
	}
} */