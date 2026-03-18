package main

import (
	//"encoding/json"
	"context"
	"flag"
	"heis/config"
	"heis/elevio"
	"heis/fsm"
	om "heis/ordermanagement"
	"heis/network"
	"heis/supervisor"
	//"time"
)

const (
	stateChannelBuffer     = 16
	orderChannelBuffer     = 16
	networkReceiveBuffer   = 64
	clearChannelBuffer     = 10
	ordersToFsmBuffer      = 10
	buttonLightsBuffer     = 16
	peerAlivenessBuffer    = 16
)

func fanOutState(in <-chan config.ElevatorState, outs ...chan<- config.ElevatorState) {
	for state := range in {
		for _, out := range outs {
			out <- state
		}
	}
}
func main() {
	idFlag := flag.String("id", "elev1", "Elevator ID (elev1, elev2, elev3, ...)")
	addrFlag := flag.String("addr", "localhost:18111", "Elevator server address")
	floorsFlag := flag.Int("floors", config.N_FLOORS, "Number of floors")
	flag.Parse()

	myID := *idFlag
	netCfg := config.DefaultNetworkConfig()
	doorTimer := fsm.NewStoppedDoorTimer()

	elevio.Init(*addrFlag, *floorsFlag)

	buttonPressedCh := make(chan config.ButtonEvent)
	floorCh := make(chan int)
	obstructionCh := make(chan bool)
	stopButtonCh := make(chan bool)

	ordersToFsmCh := make(chan om.Orders, ordersToFsmBuffer)
	clearCh := make(chan config.ClearEvent, clearChannelBuffer)

	fsmStateCh := make(chan config.ElevatorState, stateChannelBuffer)   // fra FSM
	localStateCh := make(chan config.ElevatorState, stateChannelBuffer) // til OM
	netLocalStateCh := make(chan config.ElevatorState, stateChannelBuffer) // til network heartbeat

	buttonLightsCh := make(chan config.LightState, buttonLightsBuffer) // fra OM til FSM

	go fanOutState(fsmStateCh, localStateCh, netLocalStateCh)

	startHardwarePolling(buttonPressedCh, floorCh, obstructionCh, stopButtonCh)

	// Elevator state network
	stateTx := make(chan config.ElevatorState, stateChannelBuffer)
	stateRx := make(chan config.ElevatorState, networkReceiveBuffer)
	peerStateCh := make(chan config.ElevatorState, networkReceiveBuffer)

	go network.Transmitter(netCfg.StatePort, stateTx)
	go network.Receiver(netCfg.StatePort, stateRx)
	go network.RunStateBroadcast(myID, netLocalStateCh, stateTx)
	go network.RunStateReceive(myID, stateRx, peerStateCh)

	// Order network
	ordersBroadcastCh := make(chan om.OrderMsg, 16) //OM -> network
	orderNetTx := make(chan om.OrderMsg, 16) // network -> bcast TX
	orderNetRx := make(chan om.OrderMsg, 64) // bcast RX -> network
	ordersFromNetworkCh := make(chan om.OrderMsg, 64) // network -> OM

	go network.Transmitter(netCfg.HallOrderPort, orderNetTx)
	go network.Receiver(netCfg.HallOrderPort, orderNetRx)
	go network.RunOrderBroadcast(ordersBroadcastCh, orderNetTx)
	go network.RunOrderReceive(orderNetRx, ordersFromNetworkCh)

	// Heartbeat network
	hbInternal := make(chan supervisor.Heartbeat, 16) // supervisor → network
	hbNetTx    := make(chan supervisor.Heartbeat, 16) // network → bcast TX
	hbNetRx    := make(chan supervisor.Heartbeat, 64) // bcast RX → network
	hbIncoming := make(chan supervisor.Heartbeat, 64) // network → supervisor

	go network.Transmitter(netCfg.HeartbeatPort, hbNetTx)
	go network.Receiver(netCfg.HeartbeatPort, hbNetRx)
	go network.RunHeartbeatBroadcast(hbInternal, hbNetTx)
	go network.RunHeartbeatReceive(myID, hbNetRx, hbIncoming)

	// --- Supervisor: peer health monitoring ---
	
	peerAlivenessCh := make(chan config.PeerEvent, peerAlivenessBuffer) //Ny
	
	sup := supervisor.New(supervisor.NewConfig(myID), hbInternal, hbIncoming, peerAlivenessCh)
	go func() {
		sup.MonitorPeerHealth(context.Background())
	}()

	go om.Run(myID,
		buttonPressedCh,
		clearCh,
		localStateCh,
		peerStateCh,
		peerAlivenessCh,
		ordersToFsmCh,
		ordersBroadcastCh,
		ordersFromNetworkCh,
		buttonLightsCh,
	)

	go fsm.Run(myID,
		doorTimer,
		floorCh,
		ordersToFsmCh,
		obstructionCh,
		stopButtonCh,
		clearCh,
		fsmStateCh,
		buttonLightsCh,
	)

	select {}
}


func startHardwarePolling(
	buttonPressedCh chan<- config.ButtonEvent,
	floorCh chan<- int,
	obstructionCh chan<- bool,
	stopButtonCh chan<- bool,
) {
	go elevio.PollButtons(buttonPressedCh)
	go elevio.PollFloorSensor(floorCh)
	go elevio.PollObstructionSwitch(obstructionCh)
	go elevio.PollStopButton(stopButtonCh)
}