package config


type TravelDirection int

const (
	TD_Up   TravelDirection = 0
	TD_Down TravelDirection = 1
)

type OrderEvent struct{
	Floor int
	Button ButtonType
}

type ClearEvent struct{
	Floor int
	Dir TravelDirection
}
 
type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
)


type ButtonEvent struct {
	Floor  int
	Button ButtonType
}
