package types

import (
	"Network-go/elevio"
	"sync"
)

type State int

const (
	Master State = 0
	Slave  State = 1
)

type StateStruct struct {
	State State
	Mx    sync.Mutex
}

var StateIns StateStruct

type PeersStruct struct {
	Peers []string
	Mx    sync.Mutex
}

var PeersIns PeersStruct

type RequestType int

const (
	ToBeServed RequestType = 0
	Served     RequestType = 1
)

type RequestData struct {
	Valid       bool
	ReceiverId  string
	ServerId    string
	ButtonEvent elevio.ButtonEvent
	Served      bool
}

type Acknowledgement int

var Id string
var MasterId = ""

var HallRequests []RequestData

type AcknowledgementsStruct struct {
	Acknowledgements []Acknowledgement
	Mx               sync.Mutex
}

var Acknowledgements AcknowledgementsStruct
