package ordermanagement

type HallButtonEvent struct {
    Floor int
    Dir   Dir
}

type CabButtonEvent struct {
    Floor int
}

type ClearHallEvent struct {
    Floor int
    Dir   Dir
}

type ClearCabEvent struct {
    Floor int
}
