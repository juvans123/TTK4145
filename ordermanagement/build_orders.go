package ordermanagement

import(
	"fmt"
	"heis/config"
	"bytes"
	"encoding/json"
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
	hallRequests := make([][]bool, config.N_FLOORS)
	for floor := 0; floor < config.N_FLOORS; floor++ {
		hallRequests[floor] = make([]bool, 2)
		hallRequests[floor][config.BT_HallUp] = ws.ConfirmedHallOrders[floor][config.BT_HallUp]
		hallRequests[floor][config.BT_HallDown] = ws.ConfirmedHallOrders[floor][config.BT_HallDown]
	}

	states := make(map[string]config.ElevatorState)
	for id, state := range ws.States {
		if !ws.Alive[id] {
			continue
		}

		if state.Immobile && id != myID {
			continue
		}

		confirmedCab, ok := ws.ConfirmedCabOrders[id]
		if ok && len(confirmedCab) == config.N_FLOORS {
			cabCopy := make([]bool, config.N_FLOORS)
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
	path := "./hall_request_assigner/hall_request_assigner"

	assignments, err := CallAssigner(path, inputAssigner)
	if err != nil {
		fmt.Printf("Assigner error: %v\n", err)
		return buildCabOnlyOrders(ws, myID)
	}

	myAssignedHall, ok := assignments[myID]
	if !ok {
		return buildCabOnlyOrders(ws, myID)
	}

	myLocalOrders := NewOrders(config.N_FLOORS)

	confirmedCab, ok := ws.ConfirmedCabOrders[myID]
	if ok && len(confirmedCab) == config.N_FLOORS {
		for floor := 0; floor < config.N_FLOORS; floor++ {
			myLocalOrders.Cab[floor] = confirmedCab[floor]
		}
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		myLocalOrders.Hall[floor][config.BT_HallUp] = myAssignedHall[floor][config.BT_HallUp]
		myLocalOrders.Hall[floor][config.BT_HallDown] = myAssignedHall[floor][config.BT_HallDown]
	}

	return myLocalOrders
}

func buildCabOnlyOrders(ws *WorldState, myID string) Orders {
	cabOnlyOrders := NewOrders(config.N_FLOORS)

	confirmedCab, ok := ws.ConfirmedCabOrders[myID]
	if !ok {
		return cabOnlyOrders
	}

	if len(confirmedCab) != config.N_FLOORS {
		return cabOnlyOrders
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		cabOnlyOrders.Cab[floor] = confirmedCab[floor]
	}

	return cabOnlyOrders
}

func buildLightState(ws *WorldState, myID string) config.LightState {
	ls := config.LightState{
		Cab: make([]bool, config.N_FLOORS),
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		ls.Hall[floor][config.BT_HallUp] = ws.ConfirmedHallOrders[floor][config.BT_HallUp]
		ls.Hall[floor][config.BT_HallDown] = ws.ConfirmedHallOrders[floor][config.BT_HallDown]
	}

	if cabOrders, ok := ws.ConfirmedCabOrders[myID]; ok && len(cabOrders) == config.N_FLOORS {
		copy(ls.Cab, cabOrders)
	}

	return ls
}

