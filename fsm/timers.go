package fsm

import "time"


func startTimer(timer *time.Timer, timerIsActive *bool, duration time.Duration) {
	if *timerIsActive {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	timer.Reset(duration)
	*timerIsActive = true
}

func stopTimer(timer *time.Timer, timerIsActive *bool) {
	if *timerIsActive {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}

	*timerIsActive = false
}

func newStoppedTimer(timeout time.Duration) (*time.Timer, bool) {
	timer := time.NewTimer(timeout)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	return timer, false
}


func startMotorTimer(timer *time.Timer, timerIsActive *bool) {
	startTimer(timer, timerIsActive, motorImmobileTimeout)
}

func stopMotorTimer(timer *time.Timer, timerIsActive *bool) {
	stopTimer(timer, timerIsActive)
}

func startObstructionTimer(timer *time.Timer, timerIsActive *bool) {
	startTimer(timer, timerIsActive, obstructionImmobileTimeout)
}

func stopObstructionTimer(timer *time.Timer, timerIsActive *bool) {
	stopTimer(timer, timerIsActive)
}
