package main

import (
	"encoding/json"
	"fmt"
	"heis/config"
	//"heis/elevio"
	//"heis/fsm"
	om "heis/ordermanagement"
	//"heis/timer"
)

func testAssigner() {
	// Test input for the assigner
	in := om.AssignerInput{
		HallRequests: [][]bool{
			{false, false},
			{true, false},
			{false, true},
			{true, false},
		},
		States: map[string]config.ElevatorState{
			"id_1": {
				ID:          "id_1",
				Behaviour:   config.BehIdle,
				Floor:       0,
				Direction:   config.DirStop,
				CabRequests: []bool{false, false, false, false},
			},
			"id_2": {
				ID:          "id_2",
				Behaviour:   config.BehMoving,
				Floor:       2,
				Direction:   config.DirUp,
				CabRequests: []bool{false, true, false, false},
			},
			"id_3": {
				ID:          "id_3",
				Behaviour:   config.BehMoving,
				Floor:       3,
				Direction:   config.DirDown,
				CabRequests: []bool{false, false, false, true},
			},
		},
	}

	// Call the assigner
	out, err := om.CallAssigner("./hall_request_assigner/hall_request_assigner", in)
	if err != nil {
		fmt.Println("Assigner error:", err)
		return
	}

	// Print the result
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println("Assigner output:")
	fmt.Println(string(b))
}

func main() {

	//numFloors := 4
	//myID := "1"

	// Uncomment the line below to test the assigner
	testAssigner()
	// return
/*
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
	go fsm.Run(t, floorCh, ordersOutCh, obstructionCh, stopButtonCh, clearCh)
	//go elevio.Run(lightsOrdersCh)

	// Start elevio.Run som sender kommandoer til heisen

	select {}
	*/
}
