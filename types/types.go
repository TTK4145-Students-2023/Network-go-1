package types

import (
	"Network-go/elevio"
	"container/list"
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
	RequestId   int
	SenderId    string
}

var Id string
var MasterId = ""

var HallRequests []RequestData

type AcknowledgementsListStruct struct {
	AcknowledgementsList *list.List
	Mx                   sync.Mutex
}
type Acknowledgements struct {
	AcknowledgementId int
	SenderId          string
}

var AcknowledgementsLists AcknowledgementsListStruct
