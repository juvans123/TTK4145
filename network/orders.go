package network

//Hensikt er å samle alle sendinger i en modul (network) per sendingstype
//Problemet er at denne ble sendt i om.Run, OM trenger ikke å kjenne til nettverkskanalen direkte


import (
	om "heis/ordermanagement"
)


func RunOrderBroadcast(
	in <- chan om.OrderMsg,
	netTx chan <- om.OrderMsg,
) {
	for msg := range in {
		select {
		case netTx <- msg:
		default:
		}
	}
}

func RunOrderReceive(
	myID string, 
	netRx <-chan om.OrderMsg,
	out chan<- om.OrderMsg,
) {
	for msg := range netRx {
		select{
		case out <- msg:
		default: 
		}
	}
}