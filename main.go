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
	buttonCh := make(chan config.ButtonEvent)
	floorCh := make(chan int)
	obstructionCh := make(chan bool)
	stopButtonCh := make(chan bool)

	//doorTimeOutCh := make(chan bool)
	//ordersCmd := make(chan om.Orders)

	omOrdersCh := make(chan om.Orders, 10)
	//lightsOrdersCh = make(chan om.Orders, 10)
	clearCh := make(chan config.ClearEvent, 10)

	// Start polling goroutines
	go elevio.PollButtons(buttonCh)
	go elevio.PollFloorSensor(floorCh)
	go elevio.PollObstructionSwitch(obstructionCh)
	go elevio.PollStopButton(stopButtonCh)

	// Start timer
	//go timer.Run(timerCmdCh, doorTimeOutCh)

	go om.Run(buttonCh, clearCh, omOrdersCh)
	t := timer.NewDoorTimer()
	go fsm.Run(t,floorCh, omOrdersCh, obstructionCh, stopButtonCh, clearCh)
	//go elevio.Run(lightsOrdersCh)

	// Start elevio.Run som sender kommandoer til heisen

	select{}
}
