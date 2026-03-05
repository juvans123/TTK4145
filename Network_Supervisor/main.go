package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/supervisor"
	"context"
	"flag"
	"fmt"
	"os"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//
//	will be received as zero-values.

const hbPort = 15647 // Port for heartbeat broadcasts

func main() {
	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}
	fmt.Printf("Starting elevator node: %s\n", id)

	hbTx := make(chan supervisor.Heartbeat)
	hbRx := make(chan supervisor.Heartbeat)
	go bcast.Transmitter(hbPort, hbTx) //Starter begge på samme port
	go bcast.Receiver(hbPort, hbRx)    //Starter begge på samme port

	/*---------Til Ordermanager-------*/
	peerEventch := make(chan supervisor.PeerEvent)

	cfg := supervisor.DefaultConfig(id)
	sup := supervisor.New(cfg, hbTx, hbRx, peerEventch)

	if err := sup.MonitorPeerHealth(context.Background()); err != nil {
		fmt.Println("Supervisor error:", err)
	}
}
