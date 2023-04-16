package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Network-go/singleElev"
	"Network-go/types"
	"flag"
	"fmt"
	"os"
	"time"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//
//	will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

func main() {
	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	flag.StringVar(&types.Id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if types.Id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		types.Id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)
	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(31661, types.Id, peerTxEnable)
	go peers.Receiver(31661, peerUpdateCh)

	// We make channels for sending and receiving our custom data types
	Transmits := make(chan RequestData)
	RequestsIn := make(chan RequestData)
	RequestsOut := make(chan RequestData)
	AcknowledgementsIn := make(chan types.Acknowledgement)
	AcknowledgementsOut := make(chan types.Acknowledgement)
	AskMaster := make(chan types.RequestData, 5)

	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(31616, Transmits, AcknowledgementsOut)
	go bcast.Receiver(31616, RequestsOut, AcknowledgementsIn)

	go func() {
		types.StateIns.Mx.Lock()
		types.StateIns.State = types.Master
		types.StateIns.Mx.Unlock()
		for {
			p := <-peerUpdateCh
			types.StateIns.Mx.Lock()
			if p.New != "" {
				if (types.StateIns.State == types.Master && types.Id < p.New) || (types.StateIns.State == types.Slave && types.MasterId < p.New) {

				} else if (types.StateIns.State == types.Master && types.Id > p.New) || (types.StateIns.State == types.Slave && types.MasterId > p.New) {
					types.MasterId = p.New
					types.StateIns.State = types.Slave
				}
			} else if types.StateIns.State == types.Slave && ((len(p.Lost) > 0 && p.Lost[0] == types.MasterId) || (len(p.Lost) > 1 && p.Lost[1] == types.MasterId)) {
				types.MasterId = ""
				types.StateIns.State = types.Master
			}
			types.StateIns.Mx.Unlock()
			types.PeersIns.Mx.Lock()
			copy(types.PeersIns.Peers, p.Peers)
			types.PeersIns.Mx.Unlock()
		}
	}()

	go singleElev.SingleElevatorRun(Requests, AskMaster)

	go func() {
		for {
			select{
				case acknowledgement <- AcknowledgementsIn:
					
				case request <- RequestsOut:
					
			}
		}
	}
	


	var i int
	var succesfull bool
	for end := time.Now().Add(time.Millisecond * 10); ; {
		if elevio.ListOfConnections.MessageSent[order.OrderId] {
			succesfull = true
			elevio.ListOfConnections.MessageSent[order.OrderId] = false
			break
		}
		// elevio.ListOfConnections.Mx.Unlock()

		if i&0xff == 0 { // Check in every 256th iteration
			if time.Now().After(end) {
				succesfull = false
				break
			}
		}
		i++
	}

	if succesfull {
		fmt.Println("Sent one order")
		break
	} else {
		fmt.Println("Thought we sent it but actually no")
		count++
	}
}
if count == 4 {
	fmt.Println("Connection to slave is lost")
	elevio.ListOfConnections.List[order.OrderId].Close()
	// elevio.ListOfConnections.List[order.OrderId] = nil
	break
}