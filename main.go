package main

import (
	"context"
	"flag"
	"heis/config"
	"heis/elevio"
	"heis/fsm"
	"heis/network"
	"heis/network/bcast"
	om "heis/ordermanagement"
	"heis/supervisor/supervisor"
	"heis/timer"
)

func main() {
	idFlag := flag.String("id", "elev1", "Elevator ID (elev1, elev2, elev3, ...)")
	addrFlag := flag.String("addr", "localhost:12345", "Elevator server address")
	floorsFlag := flag.Int("floors", config.NumFloors, "Number of floors")
	flag.Parse()
	myID := *idFlag

	elevio.Init(*addrFlag, *floorsFlag)

	// --- Hardware polling ---
	buttonCh := make(chan config.ButtonEvent)
	floorCh := make(chan int)
	obstructionCh := make(chan bool)
	stopButtonCh := make(chan bool)

	go elevio.PollButtons(buttonCh)
	go elevio.PollFloorSensor(floorCh)
	go elevio.PollObstructionSwitch(obstructionCh)
	go elevio.PollStopButton(stopButtonCh)

	// --- FSM/OM kanaler ---
	ordersOutCh := make(chan om.Orders, 10)
	clearCh := make(chan config.ClearEvent, 10)
	localStateCh := make(chan config.ElevatorState)
	fsmStateCh := make(chan config.ElevatorState, 16)

	// --- Network: ElevatorState broadcast ---
	stateTx := make(chan config.ElevatorState, 16)
	stateRx := make(chan config.ElevatorState, 64)
	peerStateCh := make(chan config.ElevatorState, 64)

	const statePort = 16570
	go bcast.Transmitter(statePort, stateTx)
	go bcast.Receiver(statePort, stateRx)
	go network.RunStateBroadcast(myID, fsmStateCh, stateTx)
	go network.RunStateReceive(myID, stateRx, peerStateCh)

	// --- Network: HallOrders broadcast ---
	hallOrderTx := make(chan config.ButtonEvent, 16)
	hallOrderRx := make(chan config.ButtonEvent, 64)

	const hallOrderPort = 16571
	go bcast.Transmitter(hallOrderPort, hallOrderTx)
	go bcast.Receiver(hallOrderPort, hallOrderRx)

	// --- Network: ClearEvents broadcast ---
	clearEventTx := make(chan config.ClearEvent, 16)
	clearEventRx := make(chan config.ClearEvent, 64)

	const clearPort = 16572
	go bcast.Transmitter(clearPort, clearEventTx)
	go bcast.Receiver(clearPort, clearEventRx)

	// --- Supervisor: peer health monitoring ---
	supCfg := config.DefaultSupervisorConfig()

	hbTx := make(chan supervisor.Heartbeat)
	hbRx := make(chan supervisor.Heartbeat)
	go bcast.Transmitter(supCfg.HeartbeatPort, hbTx)
	go bcast.Receiver(supCfg.HeartbeatPort, hbRx)

	peerEventCh := make(chan supervisor.PeerEvent, 16)
	sup := supervisor.New(supervisor.NewConfig(myID), hbTx, hbRx, peerEventCh)
	go func() {
		sup.MonitorPeerHealth(context.Background())
	}()

	// --- Order manager ---
	go om.Run(
		myID,
		buttonCh, clearCh, localStateCh, peerStateCh,
		peerEventCh,
		ordersOutCh,
		hallOrderTx, hallOrderRx,
		clearEventTx, clearEventRx,
	)

	// --- FSM ---
	t := timer.NewDoorTimer()
	go fsm.Run(myID, t, floorCh, ordersOutCh, obstructionCh, stopButtonCh, clearCh, fsmStateCh)

	select {}
}