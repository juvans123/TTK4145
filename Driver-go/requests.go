package elevator

import "Driver-go/elevio"

var requests [elevio.N_FLOORS][elevio.N_BUTTONS]bool

// AddRequest adds a request to the matrix

func request_addRequest(event elevio.ButtonEvent) {
	if event.Floor >= 0 && event.Floor < elevio.N_FLOORS && int(event.Button) < elevio.N_BUTTONS {
		requests[event.Floor][event.Button] = true
	}
}

// RemoveRequest removes a request from the matrix
func RemoveRequest(floor int, button elevio.ButtonType) {
	if floor >= 0 && floor < elevio.N_FLOORS && int(button) < elevio.N_BUTTONS {
		requests[floor][button] = false
	}
}

// GetRequest returns the status of a request
func GetRequest(floor int, button elevio.ButtonType) bool {
	if floor >= 0 && floor < elevio.N_FLOORS && int(button) < elevio.N_BUTTONS {
		return requests[floor][button]
	}
	return false
}
