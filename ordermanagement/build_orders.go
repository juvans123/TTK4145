package ordermanagement

import (
	"bytes"
	"encoding/json"
	"fmt"
	"heis/types"
	"os/exec"
)

func NewOrders(numFloors int) Orders {
	orders := Orders{
		Cab:  make([]bool, numFloors),
		Hall: make([][]bool, numFloors),
	}

	for floor := 0; floor < numFloors; floor++ {
		orders.Hall[floor] = make([]bool, 2)
	}

	return orders
}

func CallAssigner(path string, in AssignerInput) (AssignerOutput, error) {
	inputJSON, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	inputJSON = append(inputJSON, '\n')

	cmd := exec.Command(path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	_, err = stdin.Write(inputJSON)
	_ = stdin.Close()
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("assigner failed: %v, stderr: %s", err, stderr.String())
	}

	var output AssignerOutput
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &output); err != nil {
		return nil, fmt.Errorf("bad JSON from assigner: %v\nraw: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return output, nil
}

func buildAssignerInput(myID string, ws *WorldState) AssignerInput {
	hallRequests := make([][]bool, types.N_FLOORS)
	for floor := 0; floor < types.N_FLOORS; floor++ {
		hallRequests[floor] = make([]bool, 2)
		hallRequests[floor][types.BT_HallUp] = ws.ConfirmedHallOrders[floor][types.BT_HallUp]
		hallRequests[floor][types.BT_HallDown] = ws.ConfirmedHallOrders[floor][types.BT_HallDown]
	}

	states := make(map[string]types.ElevatorState)
	for id, state := range ws.States {
		if !ws.Alive[id] {
			continue
		}

		if state.IsImmobile && id != myID {
			continue
		}

		confirmedCab, ok := ws.ConfirmedCabOrders[id]
		if ok && len(confirmedCab) == types.N_FLOORS {
			cabCopy := make([]bool, types.N_FLOORS)
			copy(cabCopy, confirmedCab)
			state.CabRequests = cabCopy
		}

		states[id] = state
	}

	return AssignerInput{
		HallRequests: hallRequests,
		States:       states,
	}
}

func buildMyLocalOrders(ws *WorldState, myID string) Orders {
	inputAssigner := buildAssignerInput(myID, ws)
	path := "./hall_request_assigner"

	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		fmt.Printf("Assigner error: %v\n", err)
		return buildCabOrdersOnly(ws, myID)
	}

	myAssignedHall, ok := assignments[myID]
	if !ok {
		return buildCabOrdersOnly(ws, myID)
	}

	myLocalOrders := NewOrders(types.N_FLOORS)

	confirmedCab, ok := ws.ConfirmedCabOrders[myID]
	if ok && len(confirmedCab) == types.N_FLOORS {
		for floor := 0; floor < types.N_FLOORS; floor++ {
			myLocalOrders.Cab[floor] = confirmedCab[floor]
		}
	}

	for floor := 0; floor < types.N_FLOORS; floor++ {
		myLocalOrders.Hall[floor][types.BT_HallUp] = myAssignedHall[floor][types.BT_HallUp]
		myLocalOrders.Hall[floor][types.BT_HallDown] = myAssignedHall[floor][types.BT_HallDown]
	}

	return myLocalOrders
}

func buildCabOrdersOnly(ws *WorldState, myID string) Orders {
	cabOrders := NewOrders(types.N_FLOORS)

	for floor := 0; floor < types.N_FLOORS; floor++ {
		cabOrders.Cab[floor] = ws.ConfirmedCabOrders[myID][floor]
	}

	return cabOrders
}

func buildLightState(ws *WorldState, myID string) types.LightState {
	ls := types.LightState{
		Cab: make([]bool, types.N_FLOORS),
	}

	for floor := 0; floor < types.N_FLOORS; floor++ {
		ls.Hall[floor][types.BT_HallUp] = ws.ConfirmedHallOrders[floor][types.BT_HallUp]
		ls.Hall[floor][types.BT_HallDown] = ws.ConfirmedHallOrders[floor][types.BT_HallDown]
	}

	if cabOrders, ok := ws.ConfirmedCabOrders[myID]; ok && len(cabOrders) == types.N_FLOORS {
		copy(ls.Cab, cabOrders)
	}

	return ls
}
