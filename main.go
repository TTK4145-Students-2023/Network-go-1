package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Network-go/singleElev"
	"Network-go/types"
	"container/list"
	"flag"
	"fmt"
	"math/rand"
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
	TransmitsInt := make(chan types.RequestData, 5)
	TransmitsOut := make(chan types.RequestData, 5)
	Requests := make(chan types.RequestData, 5)
	RequestsToSingleElev := make(chan types.RequestData, 5)
	AcknowledgementsIn := make(chan types.Acknowledgement, 5)
	AcknowledgementsOut := make(chan types.Acknowledgement, 5)
	AskMaster := make(chan types.RequestData, 5)

	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(31616, TransmitsOut, AcknowledgementsOut)
	go bcast.Receiver(31616, Requests, AcknowledgementsIn)

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

	go singleElev.SingleElevatorRun(Requests, RequestsToSingleElev, AskMaster)

	go func() {
		for {
			request := <-Requests

			types.StateIns.Mx.Lock()
			curState := types.StateIns.State
			types.StateIns.Mx.Unlock()

			if request.Valid {
				AcknowledgementsOut <- request.RequestId

				if curState == types.Master {
					if !request.Served {
						if request.ReceiverId != types.Id {
							request.ServerId = request.ReceiverId
							request.RequestId = types.Acknowledgement(rand.Intn(1000000000))
							TransmitsInt <- request
						} else {
							request.ServerId = request.ReceiverId
							RequestsToSingleElev <- request
						}
					} else {

					}

				} else {
					if !request.Served {
						if request.ServerId == types.Id {
							RequestsToSingleElev <- request
						} else {

						}
					} else {

					}
				}
			}
		}
	}()

	go func() {
		types.Acknowledgements.Mx.Lock()
		types.Acknowledgements.AcknowledgementsList = list.New()
		types.Acknowledgements.Mx.Unlock()
		for {
			acknowledgementsIn := <-AcknowledgementsIn
			types.Acknowledgements.Mx.Lock()
			for item := types.Acknowledgements.AcknowledgementsList.Front(); item != nil; item = item.Next() {
				if item.Value == acknowledgementsIn {
					item.Value = -1
				}
			}
			types.Acknowledgements.Mx.Unlock()
		}
	}()

	go func() {
		for {
			transmitRequest := <-TransmitsInt
			types.Acknowledgements.Mx.Lock()
			ackRef := types.Acknowledgements.AcknowledgementsList.PushBack(transmitRequest.RequestId)
			types.Acknowledgements.Mx.Unlock()

			count := 0
			for {
				TransmitsOut <- transmitRequest

				var i int
				var succesfull bool
				for end := time.Now().Add(time.Millisecond * 10); ; {
					if i&0xff == 0 { // Check in every 256th iteration
						types.Acknowledgements.Mx.Lock()
						succesfull = (ackRef.Value == types.Acknowledgement(-1))

						if succesfull {
							types.Acknowledgements.AcknowledgementsList.Remove(ackRef)
							types.Acknowledgements.Mx.Unlock()
							break
						} else {
							types.Acknowledgements.Mx.Unlock()
						}

						if time.Now().After(end) {
							succesfull = false
							break
						}
					}
					i++
				}

				if succesfull {
					fmt.Println("Succesfully sent one req")
					break
				} else {
					fmt.Println("Sent one req but no acknowledgement")
					count++
				}

				if count == 4 {
					fmt.Println("Was not able to send this req for 4 times")
					types.Acknowledgements.Mx.Lock()
					types.Acknowledgements.AcknowledgementsList.Remove(ackRef)
					types.Acknowledgements.Mx.Unlock()
					if !transmitRequest.Served {
						fmt.Println("Gonna redirect it to me")
						transmitRequest.ServerId = types.Id
						RequestsToSingleElev <- transmitRequest
					}
					break
				}
			}
		}
	}()

	go func() {

	}()
}
