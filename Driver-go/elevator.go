package elevator

import "Driver-go/elevio"

func elevator_setMotorDirection(dir elevio.MotorDirection) {
	elevio.SetMotorDirection(dir)
}

func elevator_setButtonLamp(button elevio.ButtonType, floor int, value bool) {
	elevio.SetButtonLamp(button, floor, value)
}

func elevator_setFloorIndicator(floor int) {
	elevio.SetFloorIndicator(floor)
}

func elevator_setDoorOpenLamp(value bool) {
	elevio.SetDoorOpenLamp(value)
}

func elevator_setStopLamp(value bool) {
	elevio.SetStopLamp(value)
}

func elevator_getButton(button elevio.ButtonType, floor int) {
	elevio.GetButton(button, floor)
}

func elevator_getStop() bool {
	return elevio.GetStop()
}

func elevator_getObstruction() bool {
	return elevio.GetObstruction()
}

func elevator_getFloor() int {
	return elevio.GetFloor()
}

func Run() {
	buttonsChan := make(chan elevio.ButtonEvent)

	// Start polling buttons in background
	go elevio.PollButtons(buttonsChan)

	// Listen for button events
	for {
		select {
		case button := <-buttonsChan:
			AddRequest(button.Floor, button.Button)
			elevator_setButtonLamp(button.Button, button.Floor, true)
		}
	}
}

func Run1() {
    buttonsChan := make(chan elevio.ButtonEvent)
    stopChan := make(chan bool)
    obstrChan := make(chan bool)
    floorChan := make(chan int)

    // Start polling all sensors in background
    go elevio.PollButtons(buttonsChan)
    go elevio.PollStopButton(stopChan)
    go elevio.PollObstructionSwitch(obstrChan)
    go elevio.PollFloorSensor(floorChan)

    // Listen for ALL events
    for {
        select {
        case button := <-buttonsChan:
            AddRequest(button.Floor, button.Button)
            elevator_setButtonLamp(button.Button, button.Floor, true)

        case floor := <-floorChan:
            elevator_setFloorIndicator(floor)

        case stop := <-stopChan:
            if stop {
                elevator_setMotorDirection(elevio.MD_Stop)
            }

        case obstr := <-obstrChan:
            if obstr {
                elevator_setMotorDirection(elevio.MD_Stop)
            }
        }
    }
}