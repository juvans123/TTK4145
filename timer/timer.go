package timer

import "time"

/* func Run(timerCmd <-chan bool, doorTimeOut chan<- bool) {

	for {
		start := <-timerCmd
		if start {
			timer := time.NewTimer(3 * time.Second)
			<-timer.C
			doorTimeOut <- false
		}
	}
}
 */
 
type DoorTimer struct {
	 timer   *time.Timer
	 timeout chan struct{}
 }
 
 func NewDoorTimer() *DoorTimer {
	 t := &DoorTimer{
		 timer:   time.NewTimer(time.Hour),      
		 timeout: make(chan struct{}, 1),
	 }
	 if !t.timer.Stop() {
		 select {
		 case <-t.timer.C:
		 default:
		 }
	 }
 
	 // ÉN goroutine som lever hele tiden:
	 go func() {
		 for range t.timer.C {
			 // non-blocking signal (slik at vi ikke hoper opp)
			 select {
			 case t.timeout <- struct{}{}:
			 default:
			 }
		 }
	 }()
 
	 return t
 }
 
 func (t *DoorTimer) Reset(d time.Duration) {
	 if !t.timer.Stop() {
		 select {
		 case <-t.timer.C:
		 default:
		 }
	 }
	 t.timer.Reset(d)
 }
 
 func (t *DoorTimer) Stop() {
	 if !t.timer.Stop() {
		 select {
		 case <-t.timer.C:
		 default:
		 }
	 }
 }
 
 func (t *DoorTimer) Timeout() <-chan struct{} {
	 return t.timeout
 }
 