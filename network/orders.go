package network

import (
	om "elevator/ordermanagement"
)

func ForwardOutgoingOrders(
	ordersFromOm <-chan om.OrderMsg,
	netTx chan<- om.OrderMsg,
) {
	for msg := range ordersFromOm {
		netTx <- msg
	}
}

func DeliverIncomingOrders(
	netRx <-chan om.OrderMsg,
	ordersToOm chan<- om.OrderMsg,
) {
	for msg := range netRx {
		ordersToOm <- msg 
	}
}