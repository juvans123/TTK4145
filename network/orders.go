package network

import (
	om "heis/ordermanagement"
)


func RunOrderBroadcast(
	in <- chan om.OrderMsg,
	netTx chan <- om.OrderMsg,
) {
	for msg := range in {
		netTx <- msg
	}
}

func RunOrderReceive(
    netRx <-chan om.OrderMsg,
    out chan<- om.OrderMsg,
) {
    for msg := range netRx {
        out <- msg  // blocking: vent til OM er klar
    }
}