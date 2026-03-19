package fsm

import (
	"elevator/elevio"
	"elevator/types"
)

func updateButtonLights(lightState types.LightState) {
	for floor := 0; floor < types.N_FLOORS; floor++ {
		elevio.SetButtonLamp(types.BT_HallUp, floor, lightState.Hall[floor][types.BT_HallUp])
		elevio.SetButtonLamp(types.BT_HallDown, floor, lightState.Hall[floor][types.BT_HallDown])
		elevio.SetButtonLamp(types.BT_Cab, floor, lightState.Cab[floor])
	}
}

func clearAllButtonLamps() {
	for floor := 0; floor < types.N_FLOORS; floor++ {
		elevio.SetButtonLamp(types.BT_HallUp, floor, false)
		elevio.SetButtonLamp(types.BT_HallDown, floor, false)
		elevio.SetButtonLamp(types.BT_Cab, floor, false)
	}
}