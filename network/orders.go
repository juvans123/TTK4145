package network

//Hensikt er å samle alle sendinger i en modul (network) per sendingstype
//Problemet er at denne ble sendt i om.Run, OM trenger ikke å kjenne til nettverkskanalen direkte


import (
	om "heis/ordermanagement"
	"heis/supervisor"
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
	lastCounter := make(map[string]uint8)

	for msg := range netRx {
		if msg.SenderID == "" || msg.SenderID == myID {
			continue
		}
		last, known := lastCounter[msg.OwnerID]
		if known && !supervisor.IsNewer(msg.Counter, last){
			continue //Filtrer bort gammel og duplikat
		}
		lastCounter[msg.SenderID] = msg.Counter
		select{
		case out <- msg:
		default: 
		}
	}
}