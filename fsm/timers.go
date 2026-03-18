package fsm

import "time"

func newStoppedTimer(duration time.Duration) (*time.Timer, bool) {
	timer := time.NewTimer(duration)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	return timer, false
}

func NewStoppedDoorTimer() *time.Timer {
	timer, _ := newStoppedTimer(doorOpenDuration)
	return timer
}

func resetTimer(timer *time.Timer, duration time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(duration)
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func startTimer(timer *time.Timer, timerIsActive *bool, duration time.Duration) {
	resetTimer(timer, duration)
	*timerIsActive = true
}

func stopActiveTimer(timer *time.Timer, timerIsActive *bool) {
	if *timerIsActive {
		stopTimer(timer)
	}
	*timerIsActive = false
}

func startMotorTimer(timer *time.Timer, timerIsActive *bool) {
	startTimer(timer, timerIsActive, motorImmobileTimeout)
}

func stopMotorTimer(timer *time.Timer, timerIsActive *bool) {
	stopActiveTimer(timer, timerIsActive)
}

func startObstructionTimer(timer *time.Timer, timerIsActive *bool) {
	startTimer(timer, timerIsActive, obstructionImmobileTimeout)
}

func stopObstructionTimer(timer *time.Timer, timerIsActive *bool) {
	stopActiveTimer(timer, timerIsActive)
}
