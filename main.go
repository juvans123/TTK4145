package main

import (
	"context"
	"flag"
	"heis/config"
	"heis/elevio"
	"heis/fsm"
	"heis/network"
	om "heis/ordermanagement"
	"heis/supervisor"
)

const (
	stateChannelBuffer   = 16
	orderChannelBuffer   = 16
	networkReceiveBuffer = 64
	clearChannelBuffer   = 10
	ordersToFsmBuffer    = 10
	buttonLightsBuffer   = 16
	peerAlivenessBuffer  = 64
)

func broadcastLocalState(in <-chan config.ElevatorState, outs ...chan<- config.ElevatorState) {
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
	netCfg := network.DefaultNetworkPorts()
	doorTimer := fsm.NewStoppedDoorTimer()

	elevio.Init(*addrFlag, *floorsFlag)

	//hardware input channels
	buttonPressedCh := make(chan config.ButtonEvent)
	floorCh := make(chan int)
	obstructionCh := make(chan bool)
	stopButtonCh := make(chan bool)

	// FSM <-> OM channels
	ordersToFsmCh := make(chan om.Orders, ordersToFsmBuffer)
	clearCh := make(chan config.ClearEvent, clearChannelBuffer)
	buttonLightsCh := make(chan config.LightState, buttonLightsBuffer)

	// Elevator state channels
	fsmStateCh := make(chan config.ElevatorState, stateChannelBuffer)
	localStateCh := make(chan config.ElevatorState, stateChannelBuffer)
	netLocalStateCh := make(chan config.ElevatorState, stateChannelBuffer)

	go broadcastLocalState(fsmStateCh, localStateCh, netLocalStateCh)
	startHardwarePolling(buttonPressedCh, floorCh, obstructionCh, stopButtonCh)

	// Elevator state network
	stateTx := make(chan config.ElevatorState, stateChannelBuffer)
	stateRx := make(chan config.ElevatorState, networkReceiveBuffer)
	peerStateCh := make(chan config.ElevatorState, networkReceiveBuffer)

	go network.Transmitter(netCfg.StatePort, stateTx)
	go network.Receiver(netCfg.StatePort, stateRx)
	go network.BroadcastLocalState(myID, netLocalStateCh, stateTx)
	go network.DeliverIncomingPeerStates(myID, stateRx, peerStateCh)

	// Order network
	ordersBroadcastCh := make(chan om.OrderMsg, 16)   //OM -> network
	orderNetTx := make(chan om.OrderMsg, 16)          // network -> bcast TX
	orderNetRx := make(chan om.OrderMsg, 64)          // bcast RX -> network
	ordersFromNetworkCh := make(chan om.OrderMsg, 64) // network -> OM

	go network.Transmitter(netCfg.OrderPort, orderNetTx)
	go network.Receiver(netCfg.OrderPort, orderNetRx)
	go network.ForwardOutgoingOrders(ordersBroadcastCh, orderNetTx)
	go network.DeliverIncomingOrders(orderNetRx, ordersFromNetworkCh)

	// Heartbeat communication
	heartbeatFromSupervisorCh := make(chan supervisor.Heartbeat, 16)
	heartbeatToBroadcastTxCh := make(chan supervisor.Heartbeat, 16)
	heartbeatFromBroadcastRxCh := make(chan supervisor.Heartbeat, 64)
	heartbeatToSupervisorCh := make(chan supervisor.Heartbeat, 64)

	go network.Transmitter(netCfg.HeartbeatPort, heartbeatToBroadcastTxCh)
	go network.Receiver(netCfg.HeartbeatPort, heartbeatFromBroadcastRxCh)
	go network.ForwardOutgoingHeartbeats(heartbeatFromSupervisorCh, heartbeatToBroadcastTxCh)
	go network.DeliverIncomingHeartbeats(myID, heartbeatFromBroadcastRxCh, heartbeatToSupervisorCh)

	// Supervisor: peer aliveness monitoring
	peerAlivenessCh := make(chan config.PeerAliveness, peerAlivenessBuffer)

	sup := supervisor.New(
		supervisor.NewSupervisorInit(myID),
		heartbeatFromSupervisorCh, // outgoing heartbeats
		heartbeatToSupervisorCh,   // incoming heartbeats from network
		peerAlivenessCh,           // detected peer liveness changes
	)

	go sup.MonitorPeerHealth(context.Background())

	//Ordermanager
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

	//fsm
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
