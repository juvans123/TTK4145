package main

import (
	//"encoding/json"
	"flag"
	"fmt"
	"heis/config"
	"heis/elevio"
	"heis/fsm"
	"heis/Network"
	om "heis/ordermanagement"
	"heis/timer"
	"time"
)

// simulateVirtualElevator sends periodic idle state updates for non-hardware elevators
func simulateVirtualElevator(myID string, numFloors int, localStateCh chan<- config.ElevatorState) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	state := config.ElevatorState{
		ID:          myID,
		Behaviour:   config.BehIdle,
		Floor:       0,
		Direction:   config.DirStop,
		CabRequests: make([]bool, numFloors),
	}

	for range ticker.C {
		localStateCh <- state
	}
}

/* 
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
 */
func main() {

	// Parse command-line flags
	idFlag := flag.String("id", "elev1", "Elevator ID (elev1, elev2, elev3, ...)")
	flag.Parse()

	numFloors := 4
	myID := *idFlag

	// Uncomment the line below to test the assigner
	//testAssigner()
	// return

	// Only connect to hardware if this is elev1
	if myID == "elev1" {
		elevio.Init("localhost:15657", numFloors)
	}

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

	// Start polling goroutines only for elev1 (hardware-connected)
	if myID == "elev1" {
		go elevio.PollButtons(buttonCh)
		go elevio.PollFloorSensor(floorCh)
		go elevio.PollObstructionSwitch(obstructionCh)
		go elevio.PollStopButton(stopButtonCh)
	}

	// Start order manager and network
	go om.Run(myID, buttonCh, clearCh, localStateCh, peerStateCh, peerUpdateCh, ordersOutCh)
	go network.Run(localStateCh, peerStateCh, peerUpdateCh, myID)

	// Start FSM only for hardware-connected elevator
	if myID == "elev1" {
		t := timer.NewDoorTimer()
		go fsm.Run(t, floorCh, ordersOutCh, obstructionCh, stopButtonCh, clearCh)
	} else {
		// Virtual elevator: send periodic idle state
		go simulateVirtualElevator(myID, numFloors, localStateCh)
	}

	fmt.Printf("Elevator %s started (hardware: %v)\n", myID, myID == "elev1")
	select {}
}
