package network

//Hensikt er å samle alle sendinger i en modul (network) per sendingstype
//Problemet er at denne ble sendt i om.Run, OM trenger ikke å kjenne til nettverkskanalen direkte

import (
	om "heis/ordermanagement"
	"time"
)

const (
	orderRetransmitInterval = 100 * time.Millisecond
	orderRetransmitCount    = 5 // Antall resendiger etter første send
)

func RunOrderBroadcast(
	in <-chan om.OrderMsg,
	netTx chan<- om.OrderMsg,
) {
	type pending struct {
		msg       om.OrderMsg
		remaining int
	}
	ticker := time.NewTicker(orderRetransmitInterval)
	defer ticker.Stop()

	var queue []pending

	for {
		select {

		case msg := <-in:
			//SJEKK OM SAMME ORDRE LIGGER I KØEN

			replaced := false
			for i, p := range queue {
				if p.msg.OwnerID == msg.OwnerID &&
					p.msg.Floor == msg.Floor &&
					p.msg.Button == msg.Button {
					queue[i] = pending{msg: msg, remaining: orderRetransmitCount}
					replaced = true
					break
				}
			}
			if !replaced {
				queue = append(queue, pending{msg: msg, remaining: orderRetransmitCount})
			}

			// NYTT: SEND ALLTID UMIDDELBART
			select {
			case netTx <- msg:
			default:
			}
		case <-ticker.C:
			//Resend alle ventede meldinger i køen, dekrementer remaining
			next := queue[:0]
			for _, p := range queue {
				select {
				case netTx <- p.msg:
				default:
				}
				p.remaining--
				if p.remaining > 0 {
					next = append(next, p)
				}
			}
			queue = next
		}
	}
}

func RunOrderReceive(
	netRx <-chan om.OrderMsg,
	out chan<- om.OrderMsg,
) {
	for msg := range netRx {
		out <- msg // blocking: vent til OM er klar
	}
}