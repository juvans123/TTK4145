package config

import (
	"heis/elevio"
)


type TravelDirection int

const (
	TD_Up   TravelDirection = 0
	TD_Down TravelDirection = 1
)

type OrderEvent struct{
	Floor int
	Button elevio.ButtonType
}

type ClearEvent struct{
	Floor int
	Dir TravelDirection
}

type OrdersSnapshot struct {
	Cab []bool
	Hall [4][2]bool
}