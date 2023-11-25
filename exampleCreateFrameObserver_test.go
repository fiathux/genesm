package genesm

import (
	"fmt"
	"time"
)

// ExampleCreateFrameObserver shows how to use a frame observer
func ExampleCreateFrameObserver() {
	// create state machine
	sm := NewStateMachine("Owner")
	// create controller
	ctr := NewObsController(ObsControlCfg{})
	// create ticker
	ticker, err := CreateObsFrameTicker(5)
	if err != nil {
		panic(err)
	}

	obI64 := CreateFrameObserver(ctr, ticker, ObsFrameFunc(
		func(owner string, ev FrameEvent, stateID StateID, skipped int64, val int64) {
			fmt.Println("Frame:", ev, stateID, skipped, val)
		},
	), nil)

	obU32 := CreateFrameObserver(ctr, ticker, ObsFrameFunc(
		func(owner string, ev FrameEvent, stateID StateID, skipped int64, val uint32) {
			fmt.Println("Frame:", ev, stateID, skipped, val)
		},
	), nil)

	// new state
	stateI64 := RegState(sm, int64(64))
	stateU32 := RegState(sm, uint32(32))

	// associate to observer
	stateI64.AddObserver(obI64)
	stateU32.AddObserver(obU32)

	switcher := GroupEvent(
		RegEvent(sm, stateI64, stateU32),
		RegEvent(sm, stateU32, stateI64))

	// switch state between stateI64 and stateU32. will see the frame event change
	<-func() <-chan struct{} {
		ret := make(chan struct{})
		go func() {
			defer close(ret)
			for i := 0; i < 10; i++ {
				switcher.Trigger()
				time.Sleep(time.Second)
			}
		}()
		return ret
	}()
}
