package main



import (
	"heis/elevio"
	"heis/fsm"
	"heis/timer"
)

func main() {

	numFloors := 4

	elevio.Init("localhost:15657", numFloors)
	

	// Kanaler for kommunikasjon
	button_pressed := make(chan elevio.ButtonEvent)
	floor_arrived := make(chan int)
	obstruction_pressed := make(chan bool)
	stopButton_pressed := make(chan bool)
	elevioCmd := make(chan elevio.DriverCmd)
	timerCmd := make(chan bool)
	doorTimeOut := make(chan bool)
	

	

	// Start polling goroutines
	go elevio.PollButtons(button_pressed)
	go elevio.PollFloorSensor(floor_arrived)
	go elevio.PollObstructionSwitch(obstruction_pressed)
	go elevio.PollStopButton(stopButton_pressed)

	// Start timer
	go timer.Run(timerCmd, doorTimeOut)

	// Start FSM som håndterer logikk
	go fsm.Run(button_pressed, doorTimeOut, floor_arrived, obstruction_pressed, stopButton_pressed, elevioCmd, timerCmd)

	// Start elevio.Run som sender kommandoer til heisen
	elevio.ExecuteElevioCmds(elevioCmd)
}
