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
	myID := "1"

	elevio.Init("localhost:15657", numFloors)
	

	// Kanaler for kommunikasjon
	buttonCh := make(chan config.ButtonEvent)
	floorCh := make(chan int)
	obstructionCh := make(chan bool)
	stopButtonCh := make(chan bool)

	ordersOutCh := make(chan om.Orders, 10)
	clearCh := make(chan config.ClearEvent, 10)
	localStateCh := make(chan config.ElevatorState)
	peerStateCh := make(chan config.ElevatorState)
	peerUpdateCh := make(chan config.PeerUpdate)

	// Start polling goroutines
	go elevio.PollButtons(buttonCh)
	go elevio.PollFloorSensor(floorCh)
	go elevio.PollObstructionSwitch(obstructionCh)
	go elevio.PollStopButton(stopButtonCh)

	// Start timer

	go om.Run(myID, buttonCh, clearCh, localStateCh, peerStateCh, peerUpdateCh, ordersOutCh)
	t := timer.NewDoorTimer()
	go fsm.Run(t,floorCh, ordersOutCh, obstructionCh, stopButtonCh, clearCh)
	//go elevio.Run(lightsOrdersCh)

	// Start elevio.Run som sender kommandoer til heisen

	select{}
}
