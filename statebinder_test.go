package genesm

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	stateA int         = 10
	stateB string      = "stateB"
	stateC int         = 100
	stateD StructState = StructState{
		a: 20,
		b: "stateD",
	}
	stateE IntfState = &StructState{
		a: 30,
		b: "stateE",
	}
)

func TestStateBinder(t *testing.T) {
	// state transition:
	//      + <----------------------+
	//      |                        |
	//   -> +-> A -> B +-> C -+-> E -+
	//          |      |      |
	//          +-> D -+      |
	//          |             |
	//          + <-----------+
	Convey("State binder test", t, func() {
		// create state machine
		sm := NewStateMachine("ownerXYZ")
		So(sm.StateID(), ShouldEqual, STIDInvalid())
		So(sm.PickState(), ShouldNotBeNil)

		// regist state
		bndA := RegState(sm, stateA)
		bndB := RegState(sm, stateB)
		bndC := RegState(sm, stateC)
		bndD := RegState(sm, stateD)
		bndE := RegState(sm, stateE)
		So(sm.StateID(), ShouldEqual, bndA.ID())

		// regist event
		eA2B := RegEvent(sm, bndA, bndB)
		eB2C := RegEvent(sm, bndB, bndC)
		eC2E := RegEvent(sm, bndC, bndE)
		eA2D := RegEvent(sm, bndA, bndD)
		eD2C := RegEvent(sm, bndD, bndC)
		eC2D := RegEvent(sm, bndC, bndD)
		eE2A := RegEvent(sm, bndE, bndA)
		eg1 := GroupEvent(eA2B, eA2D, eB2C, eE2A)
		eg2 := GroupEvent(eD2C, eC2E, eC2D)

		hooked := false
		eA2B.SetHook(func(owner string, a int, b string) error {
			if hooked {
				return nil
			}
			So(owner, ShouldEqual, "ownerXYZ")
			So(a, ShouldEqual, 10)
			So(b, ShouldEqual, "stateB")
			hooked = true
			return nil
		})
		So(eA2B.Trigger(), ShouldBeNil)
		So(hooked, ShouldBeTrue)
		So(sm.StateID(), ShouldEqual, bndB.ID())
		So(eA2B.Trigger(), ShouldEqual, ErrEvAlreadyChanged)
		So(eA2D.Trigger(), ShouldEqual, ErrEvUnexpectedState)
		So(eB2C.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndC.ID())
		So(eC2E.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndE.ID())
		So(eE2A.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndA.ID())
		So(eA2B.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndB.ID())
		So(eB2C.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndC.ID())
		So(eC2D.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndD.ID())
		So(eD2C.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndC.ID())
		So(eA2B.Trigger(), ShouldEqual, ErrEvUnexpectedState)
		So(eA2D.Trigger(), ShouldEqual, ErrEvUnexpectedState)
		So(eB2C.Trigger(), ShouldEqual, ErrEvAlreadyChanged)
		So(eE2A.Trigger(), ShouldEqual, ErrEvUnexpectedState)
		So(eD2C.Trigger(), ShouldEqual, ErrEvAlreadyChanged)
		So(eg1.Trigger(), ShouldEqual, ErrEvGroupFailure)
		So(eg2.Trigger(), ShouldBeNil)
		So(sm.StateID(), ShouldEqual, bndE.ID())

		sm.SetOwner("ownerABC")
		So(sm.GetOwner(), ShouldEqual, "ownerABC")

		bndA.Protect(func(owner string, v int, selected bool) {
			So(owner, ShouldEqual, "ownerABC")
			So(v, ShouldEqual, 10)
			So(selected, ShouldBeFalse)
		})
		bndE.Protect(func(owner string, v IntfState, selected bool) {
			So(v.GetA(), ShouldEqual, 30)
			So(v.GetB(), ShouldEqual, "stateE")
			So(selected, ShouldBeTrue)
		})
		bndE.Set(&StructState{
			a: 31,
			b: "stateE+",
		})
		bndE.Protect(func(owner string, v IntfState, selected bool) {
			So(v.GetA(), ShouldEqual, 31)
			So(v.GetB(), ShouldEqual, "stateE+")
			So(selected, ShouldBeTrue)
		})

		Convey("State binder with observer", func() {
			aEvt := [4]bool{}
			chEvt := make(chan bool)
			obA := ObsEventFuncs(
				func(owner string, id StateID, val int) {
					aEvt[0] = true
					chEvt <- true
				}, func(owner string, id StateID, val int) {
					aEvt[1] = true
					chEvt <- true
				}, func(owner string, id StateID, val int) {
					aEvt[2] = true
					chEvt <- true
				}, func(owner string, id StateID, val int) {
					aEvt[3] = true
					chEvt <- val == 11
				})
			obARef := CreateEventObserver(nil, obA, nil)
			So(bndA.AddObserver(obARef), ShouldBeNil)
			So(bndC.AddObserver(obARef), ShouldEqual, ErrObBeenBound)

			So(eE2A.Trigger(), ShouldBeNil)
			So(<-chEvt, ShouldBeTrue)
			So(aEvt, ShouldEqual, [4]bool{true, false, false, false})
			So(sm.PickState(), ShouldBeNil)
			So(<-chEvt, ShouldBeTrue)
			So(aEvt, ShouldEqual, [4]bool{true, false, true, false})
			bndA.Set(11)
			So(<-chEvt, ShouldBeTrue)
			So(aEvt, ShouldEqual, [4]bool{true, false, true, true})
			eA2B.Trigger()
			So(<-chEvt, ShouldBeTrue)
			So(aEvt, ShouldEqual, [4]bool{true, true, true, true})
		})
	})
}
