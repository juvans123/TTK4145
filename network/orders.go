package network

import (
	om "heis/ordermanagement"
)

func ForwardOutgoingOrders(
	in <-chan om.OrderMsg,
	netTx chan<- om.OrderMsg,
) {
	for msg := range in {
		netTx <- msg
	}
}

func DeliverIncomingOrders(
	netRx <-chan om.OrderMsg,
	out chan<- om.OrderMsg,
) {
	for msg := range netRx {
		out <- msg // blocking: vent til OM er klar
	}
}