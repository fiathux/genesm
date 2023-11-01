package genesm

import (
	"fmt"
	"time"
)

// example struct
type StructState struct {
	a int
	b string
}

// example interface
type IntfState interface {
	GetA() int
	GetB() string
}

func (s *StructState) GetA() int {
	return s.a
}

func (s *StructState) GetB() string {
	return s.b
}

func Example() {

	// example values
	var (
		stateA int         = 10
		stateB string      = "stateB"
		stateC int32       = 100
		stateD StructState = StructState{
			a: 20,
			b: "stateD",
		}
		stateE IntfState = &StructState{
			a: 30,
			b: "stateE",
		}
	)

	// state transition:
	//      + <----------------------+
	//      |                        |
	//   -> +-> A -> B +-> C -+-> E -+
	//          |      |      |
	//          +-> D -+      |
	//          |             |
	//          + <-----------+

	// create state machine
	sm := NewStateMachine("Owner")

	// regist state
	bndA := RegState(sm, stateA)
	bndB := RegState(sm, stateB)
	bndC := RegState(sm, stateC)
	bndD := RegState(sm, stateD)
	bndE := RegState(sm, stateE)

	// regist event
	eA2B := RegEvent(sm, bndA, bndB)
	eB2C := RegEvent(sm, bndB, bndC)
	eC2E := RegEvent(sm, bndC, bndE)
	eA2D := RegEvent(sm, bndA, bndD)
	eD2C := RegEvent(sm, bndD, bndC)
	eC2D := RegEvent(sm, bndC, bndD)
	eE2A := RegEvent(sm, bndE, bndA)

	obctr := NewObsController(ObsControlCfg{})
	// add observer
	bndA.AddObserver(CreateEventObserver(obctr, ObsEventFuncs(
		func(owner string, id StateID, val int) { // enter
			fmt.Println("StateA Enter, val:", val)
		},
		func(owner string, id StateID, val int) { // exit
			fmt.Println("StateA Exit, val:", val)
		}, nil, nil,
	), nil))
	bndB.AddObserver(CreateEventObserver(obctr, ObsEventFuncs(
		func(owner string, id StateID, val string) {
			fmt.Println("StateB Enter, val:", val)
		},
		func(owner string, id StateID, val string) {
			fmt.Println("StateB Exit, val:", val)
		}, nil, nil,
	), nil))
	bndC.AddObserver(CreateEventObserver(obctr, ObsEventFuncs(
		func(owner string, id StateID, val int32) {
			fmt.Println("StateC Enter, val:", val)
		},
		func(owner string, id StateID, val int32) {
			fmt.Println("StateC Exit, val:", val)
		}, nil, nil,
	), nil))
	bndD.AddObserver(CreateEventObserver(obctr, ObsEventFuncs(
		func(owner string, id StateID, val StructState) {
			fmt.Println("StateD Enter, val:", val)
		},
		func(owner string, id StateID, val StructState) {
			fmt.Println("StateD Exit, val:", val)
		},
		func(owner string, id StateID, val StructState) { // pick
			fmt.Println("StateD Pick, val:", val)
		},
		func(owner string, id StateID, val StructState) { // update
			fmt.Println("StateD Update, val:", val)
		},
	), nil))
	bndE.AddObserver(CreateEventObserver(obctr, ObsEventFuncs(
		func(owner string, id StateID, val IntfState) {
			fmt.Println("StateE Enter, val:", val)
		},
		func(owner string, id StateID, val IntfState) {
			fmt.Println("StateE Exit, val:", val)
		}, nil, nil,
	), nil))

	// do test
	fmt.Println("A -> B (Availabled):", eA2B.Trigger())
	fmt.Println("B -> C (Availabled):", eB2C.Trigger())
	fmt.Println("C -> E (Availabled):", eC2E.Trigger())
	fmt.Println("E -> A (Availabled):", eE2A.Trigger())
	fmt.Println("A -> D (Availabled):", eA2D.Trigger())
	fmt.Println("D -> C (Availabled):", eD2C.Trigger())
	fmt.Println("C -> D (Availabled):", eC2D.Trigger())
	fmt.Println("C -> D (Repeated):", eC2D.Trigger())
	fmt.Println("A -> B (Invalid):", eA2B.Trigger())
	fmt.Println("B -> C (Invalid):", eB2C.Trigger())
	fmt.Println("C -> E (Invalid):", eC2E.Trigger())
	fmt.Println("E -> A (Invalid):", eE2A.Trigger())
	fmt.Println("A -> D (Invalid):", eA2D.Trigger())
	fmt.Println("E -> A (Invalid):", eE2A.Trigger())

	sm.PickState() // trigger Pick event to current state
	bndD.Set(StructState{
		a: 21,
		b: "stateD+",
	}) // update contained value

	time.Sleep(1 * time.Second) // wait for show async event in observer
}
