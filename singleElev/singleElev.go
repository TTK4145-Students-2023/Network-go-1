package singleElev

import (
	"Network-go/elevio"
	"Network-go/types"
	"fmt"
	"math/rand"
)

var Port_name string

type State int

const (
	Idle      State = 0
	Moving    State = 1
	Door_Open State = 2
)

func SingleElevatorRun(Requests chan types.RequestData, RequestsToSingleElev chan types.RequestData, TransmitsInt chan types.RequestData) {

	numFloors := 4
	current_floor := -1
	var current_state State = Idle
	var door_obstructed bool = false
	var indicated_direction elevio.ButtonType = elevio.BT_Cab

	elevio.Init(("localhost:" + Port_name), numFloors)

	var d elevio.MotorDirection = elevio.MD_Up

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)
	drv_timer_serv := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	var requests_up []elevio.ButtonEvent
	var requests_down []elevio.ButtonEvent
	var request_here elevio.RequestHere
	request_here.Valid = false

	// Initalize/single_elevator_go
	elevio.SetStopLamp(false)
	elevio.SetDoorOpenLamp(false)
	for floor := 0; floor < numFloors; floor++ {
		for button := elevio.ButtonType(0); button < 3; button++ {
			elevio.SetButtonLamp(button, floor, false)
		}
	}

	elevio.SetMotorDirection(d)
	for current_floor == -1 {
		a := <-drv_floors
		current_floor = a
		elevio.SetFloorIndicator(current_floor)
		d = elevio.MD_Stop
		elevio.SetMotorDirection(d)
		fmt.Printf("Initalized at floor %+v\n", a)
	}
	/////////////

	for {
		select {
		case a := <-drv_buttons:
			fmt.Println("Button press", a)
			//a.Floor += 1
			Requests <- types.RequestData{Valid: true, ButtonEvent: a, Served: false, ReceiverId: types.Id, ServerId: "", RequestId: rand.Intn(1000000000)}
		case b := <-RequestsToSingleElev:
			a := b.ButtonEvent
			//a.Floor -= 1
			fmt.Println("Button in order", a)
			var light bool = true

			types.StateIns.Mx.Lock()
			curState := types.StateIns.State
			types.StateIns.Mx.Unlock()

			if current_state == Moving {
				if curState == types.Slave && a.Button != elevio.BT_Cab && types.Id != b.ServerId {
					TransmitsInt <- b
				} else {
					if a.Floor < current_floor {
						requests_down = append(requests_down, a)
					} else if a.Floor > current_floor {
						requests_up = append(requests_up, a)
					} else {
						if d == elevio.MD_Up {
							requests_down = append(requests_down, a)
						} else {
							requests_up = append(requests_up, a)
						}
					}
				}

			} else if current_state == Idle {
				if a.Floor < current_floor {
					if curState == types.Slave && a.Button != elevio.BT_Cab && types.Id != b.ServerId {
						TransmitsInt <- b
					} else {
						fmt.Println("moving down")
						//fmt.Println(requests_down)
						requests_down = append(requests_down, a)
						//fmt.Println(requests_down)
						current_state = Moving
						d = elevio.MD_Down
						elevio.SetMotorDirection(d)
					}

				} else if a.Floor > current_floor {
					if curState == types.Slave && a.Button != elevio.BT_Cab && types.Id != b.ServerId {
						TransmitsInt <- b
					} else {
						fmt.Println("moving up")
						requests_up = append(requests_up, a)
						current_state = Moving
						d = elevio.MD_Up
						elevio.SetMotorDirection(d)
					}

				} else {
					fmt.Println("same floor")
					current_state = Door_Open
					indicated_direction = a.Button
					elevio.SetDoorOpenLamp(true)
					go elevio.Timer(drv_timer_serv)
					light = false
				}

			} else if current_state == Door_Open {
				if a.Floor < current_floor {
					if curState == types.Slave && a.Button != elevio.BT_Cab && types.Id != b.ServerId {
						TransmitsInt <- b
					} else {
						requests_down = append(requests_down, a)
					}
				} else if a.Floor > current_floor {
					if curState == types.Slave && a.Button != elevio.BT_Cab && types.Id != b.ServerId {
						TransmitsInt <- b
					} else {
						requests_up = append(requests_up, a)
					}
				} else {
					if a.Button == indicated_direction || a.Button == elevio.BT_Cab {
						go elevio.Timer(drv_timer_serv)
						light = false
					} else {
						request_here.Request = a
						request_here.Valid = true
						light = true
					}
				}
			}

			if light {
				elevio.SetButtonLamp(a.Button, a.Floor, true)
			}

		case a := <-drv_floors:
			fmt.Printf("Floor sensor : %+v\n", a)
			var stop bool = false

			var new_requests []elevio.ButtonEvent
			if d == elevio.MD_Up {
				for _, s := range requests_up {
					if a == s.Floor {
						if s.Button == elevio.BT_HallDown {
							request_here.Valid = true
							request_here.Request = s
						} else {
							elevio.SetButtonLamp(s.Button, s.Floor, false)
							stop = true
							indicated_direction = elevio.BT_HallUp
						}

					} else {
						new_requests = append(new_requests, s)
					}
				}

				if request_here.Valid && stop != true {
					further_up := false
					for _, j := range requests_up {
						if a < j.Floor {
							further_up = true
						}
					}
					if further_up == false {
						elevio.SetButtonLamp(request_here.Request.Button, request_here.Request.Floor, false)
						request_here.Valid = false
						stop = true
						indicated_direction = elevio.BT_HallDown
					} else {
						requests_down = append(requests_down, request_here.Request)
						request_here.Valid = false
					}
				}

				requests_up = new_requests

			} else {
				for _, s := range requests_down {
					if a == s.Floor {
						if s.Button == elevio.BT_HallUp {
							request_here.Valid = true
							request_here.Request = s
							//fmt.Println("This is running on floor ", a)
						} else {
							elevio.SetButtonLamp(s.Button, s.Floor, false)
							stop = true
							indicated_direction = elevio.BT_HallDown
						}
					} else {
						new_requests = append(new_requests, s)
					}
				}

				if request_here.Valid && stop != true {
					further_down := false
					for _, j := range requests_down {
						if a > j.Floor {
							further_down = true
						}
					}
					if further_down == false {
						elevio.SetButtonLamp(request_here.Request.Button, request_here.Request.Floor, false)
						request_here.Valid = false
						stop = true
						indicated_direction = elevio.BT_HallUp
					} else {
						requests_up = append(requests_up, request_here.Request)
						request_here.Valid = false
					}
				}

				requests_down = new_requests
			}

			if stop {
				elevio.SetDoorOpenLamp(true)
				current_state = Door_Open
				if len(new_requests) == 0 {
					d = elevio.MD_Stop
				}
				elevio.SetMotorDirection(elevio.MD_Stop)
				go elevio.Timer(drv_timer_serv)
			}

			current_floor = a
			elevio.SetFloorIndicator(current_floor)

		case a := <-drv_obstr:
			if current_state == Door_Open {
				if a {
					fmt.Printf("Door obstructed!\n")
					door_obstructed = true
				} else {
					door_obstructed = false
				}
			}

		case <-drv_timer_serv:
			if request_here.Valid &&
				((indicated_direction == elevio.BT_HallUp && len(requests_up) == 0) ||
					(indicated_direction == elevio.BT_HallDown && len(requests_down) == 0)) {
				indicated_direction = request_here.Request.Button
				elevio.SetButtonLamp(request_here.Request.Button, request_here.Request.Floor, false)
				request_here.Valid = false
				go elevio.Timer(drv_timer_serv)

			} else {
				if !door_obstructed {
					fmt.Printf("Door closing \n")
					elevio.SetDoorOpenLamp(false)

					types.StateIns.Mx.Lock()
					curState := types.StateIns.State
					types.StateIns.Mx.Unlock()

					if indicated_direction == elevio.BT_HallUp && len(requests_up) > 0 {
						d = elevio.MD_Up
						elevio.SetMotorDirection(d)
						current_state = Moving
						indicated_direction = elevio.BT_Cab
						if request_here.Valid {
							if curState == types.Slave && request_here.Request.Button != elevio.BT_Cab {
								TransmitsInt <- types.RequestData{Valid: true, ButtonEvent: request_here.Request, Served: false, ReceiverId: types.Id, ServerId: ""}
							} else {
								requests_down = append(requests_down, request_here.Request)
							}
							request_here.Valid = false
						}

					} else if indicated_direction == elevio.BT_HallDown && len(requests_down) > 0 {
						d = elevio.MD_Down
						elevio.SetMotorDirection(d)
						current_state = Moving
						indicated_direction = elevio.BT_Cab
						if request_here.Valid {
							if curState == types.Slave && request_here.Request.Button != elevio.BT_Cab {
								TransmitsInt <- types.RequestData{Valid: true, ButtonEvent: request_here.Request, Served: false, ReceiverId: types.Id, ServerId: ""}
							} else {
								requests_up = append(requests_up, request_here.Request)
							}
							request_here.Valid = false
						}

					} else if len(requests_up) > 0 {
						d = elevio.MD_Up
						elevio.SetMotorDirection(d)
						current_state = Moving

					} else if len(requests_down) > 0 {
						d = elevio.MD_Down
						elevio.SetMotorDirection(d)
						current_state = Moving

					} else {
						current_state = Idle
						indicated_direction = elevio.BT_Cab
					}

				} else {
					fmt.Printf("Door cannot close \n")
					go elevio.Timer(drv_timer_serv)
				}
			}
		}
	}
}
