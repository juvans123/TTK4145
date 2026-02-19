package main

import (
	"heis/config"
	"heis/elevio"
	"heis/fsm"
	om "heis/ordermanagement"
	"heis/timer"
)

func main() {

	numFloors := 4

	elevio.Init("localhost:15657", numFloors)
	

	// Kanaler for kommunikasjon
	buttonCh := make(chan elevio.ButtonEvent)
	floorCh := make(chan int)
	obstructionCh := make(chan bool)
	stopButtonCh := make(chan bool)

	//doorTimeOutCh := make(chan bool)
	//ordersCmd := make(chan om.Orders)

	ordersCh := make(chan om.Orders, 10)
	clearCh := make(chan config.ClearEvent, 10)

	// Start polling goroutines
	go elevio.PollButtons(buttonCh)
	go elevio.PollFloorSensor(floorCh)
	go elevio.PollObstructionSwitch(obstructionCh)
	go elevio.PollStopButton(stopButtonCh)

	// Start timer
	//go timer.Run(timerCmdCh, doorTimeOutCh)

	go om.Run(buttonCh, clearCh, ordersCh)
	t := timer.NewDoorTimer()
	go fsm.Run(t,floorCh, ordersCh, obstructionCh, stopButtonCh, clearCh)

	// Start elevio.Run som sender kommandoer til heisen

	select{}
}
