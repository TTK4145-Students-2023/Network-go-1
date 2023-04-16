package main

import (
	"Network-go/elevio"
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
	flag.StringVar(&singleElev.Port_name, "id", "", "id of this peer")
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
	AcknowledgementsIn := make(chan types.Acknowledgements, 5)
	AcknowledgementsOut := make(chan types.Acknowledgements, 5)

	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(31616, TransmitsOut)
	go bcast.Receiver(31616, Requests)
	go bcast.Transmitter(31615, AcknowledgementsOut)
	go bcast.Receiver(31615, AcknowledgementsIn)

	go func() {
		types.StateIns.Mx.Lock()
		types.StateIns.State = types.Master
		types.StateIns.Mx.Unlock()
		for {
			p := <-peerUpdateCh
			types.StateIns.Mx.Lock()
			if p.New != "" {
				fmt.Println("Found new connection", p.New)
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

	go singleElev.SingleElevatorRun(Requests, RequestsToSingleElev, TransmitsInt)

	go func() {
		for {
			request := <-Requests

			types.StateIns.Mx.Lock()
			curState := types.StateIns.State
			types.StateIns.Mx.Unlock()

			if request.Valid && request.SenderId != types.Id {
				fmt.Println("Gonna process one request as ", curState)
				if curState == types.Master {
					if !request.Served {

						if request.ButtonEvent.Button != elevio.BT_Cab && !CheckHallRequests(request) {
							fmt.Println("New hall request append")
							types.HallRequests = append(types.HallRequests, request)
						}

						if request.ReceiverId != types.Id {
							fmt.Println("Received a request gonna acknowledge")
							AcknowledgementsOut <- types.Acknowledgements{AcknowledgementId: request.RequestId, SenderId: types.Id}
							request.ServerId = request.ReceiverId
							request.RequestId = rand.Intn(1000000000)
							TransmitsInt <- request
						} else {
							fmt.Println("Its mine")
							request.ServerId = request.ReceiverId
							RequestsToSingleElev <- request
						}
					} else {

					}

				} else {
					if !request.Served {
						if request.ServerId == types.Id {
							fmt.Println("Its from the master gonna acknowledge")
							AcknowledgementsOut <- types.Acknowledgements{AcknowledgementId: request.RequestId, SenderId: types.Id}
							RequestsToSingleElev <- request
						} else if request.ServerId == "" {
							fmt.Println("Its new should try myself first")
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
		types.AcknowledgementsLists.Mx.Lock()
		types.AcknowledgementsLists.AcknowledgementsList = list.New()
		types.AcknowledgementsLists.Mx.Unlock()
		for {
			acknowledgementsIn := <-AcknowledgementsIn
			if acknowledgementsIn.SenderId != types.Id {
				fmt.Println("Received an acknowledgement")
				types.AcknowledgementsLists.Mx.Lock()
				for item := types.AcknowledgementsLists.AcknowledgementsList.Front(); item != nil; item = item.Next() {
					//fmt.Println("item: ", item.Value, "")
					if item.Value == acknowledgementsIn.AcknowledgementId {
						fmt.Println("Found the acknowledgement in the list")
						item.Value = -1
					}
				}
				types.AcknowledgementsLists.Mx.Unlock()
			}
		}
	}()

	go func() {
		for {
			transmitRequest := <-TransmitsInt
			types.AcknowledgementsLists.Mx.Lock()
			ackRef := types.AcknowledgementsLists.AcknowledgementsList.PushBack(transmitRequest.RequestId)
			types.AcknowledgementsLists.Mx.Unlock()

			count := 0
			for {
				transmitRequest.SenderId = types.Id
				TransmitsOut <- transmitRequest

				var i int
				var succesfull bool
				for end := time.Now().Add(time.Millisecond * 100); ; {
					if i&0xffff == 0 { // Check in every 0xffff + 1 iteration
						types.AcknowledgementsLists.Mx.Lock()
						succesfull = (ackRef.Value == -1)

						if succesfull {
							fmt.Println("The request sent is acknowledged")
							types.AcknowledgementsLists.AcknowledgementsList.Remove(ackRef)
							types.AcknowledgementsLists.Mx.Unlock()
							break
						} else {
							types.AcknowledgementsLists.Mx.Unlock()
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
					types.AcknowledgementsLists.Mx.Lock()
					types.AcknowledgementsLists.AcknowledgementsList.Remove(ackRef)
					types.AcknowledgementsLists.Mx.Unlock()
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

	select {}
}

func CheckHallRequests(request types.RequestData) bool {
	for _, listReq := range types.HallRequests {
		if listReq.ButtonEvent == request.ButtonEvent {
			return true
		}
	}
	return false
}
