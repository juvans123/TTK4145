package main

import (
	"heis/elevio"
	"heis/fsm"
	"heis/ordermanagement"
	om "heis/ordermanagement"
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
	timerCmd := make(chan bool,1)
	doorTimeOut := make(chan bool, 1)
	ordersCmd := make(chan om.Orders)
	orders_Executed := make(chan om.ExecutedOrder)

	

	// Start polling goroutines
	go elevio.PollButtons(button_pressed)
	go elevio.PollFloorSensor(floor_arrived)
	go elevio.PollObstructionSwitch(obstruction_pressed)
	go elevio.PollStopButton(stopButton_pressed)

	// Start timer
	go timer.Run(timerCmd, doorTimeOut)

	// Start FSM som håndterer logikk
	go fsm.Run(orders_Executed, button_pressed, ordersCmd, doorTimeOut, floor_arrived, obstruction_pressed, stopButton_pressed, elevioCmd, timerCmd)

	go ordermanagement.Run(button_pressed, ordersCmd, orders_Executed)

	// Start elevio.Run som sender kommandoer til heisen
	elevio.ExecuteElevioCmds(elevioCmd)
}
