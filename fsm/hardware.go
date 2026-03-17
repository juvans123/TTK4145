package fsm

import (
	"heis/elevio"
	"heis/config"
)

func setMotor(dir elevio.MotorDirection) {
	elevio.SetMotorDirection(dir)
}

func stopMotor() {
	elevio.SetMotorDirection(elevio.MD_Stop)
}

func updateButtonLights(lightState config.LightState) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(config.BT_HallUp, floor, lightState.Hall[floor][config.BT_HallUp])
		elevio.SetButtonLamp(config.BT_HallDown, floor, lightState.Hall[floor][config.BT_HallDown])
		elevio.SetButtonLamp(config.BT_Cab, floor, lightState.Cab[floor])
	}
}

func clearAllButtonLamps() {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(config.BT_HallUp, floor, false)
		elevio.SetButtonLamp(config.BT_HallDown, floor, false)
		elevio.SetButtonLamp(config.BT_Cab, floor, false)
	}
}