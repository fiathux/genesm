package genesm

import (
	"runtime"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEventObserver(t *testing.T) {
	// with hook
	Convey("Event observer with hook test", t, func() {
		eventRet := []bool{true, true, true, true}
		hookRet := []bool{true, true, true, true}
		shiftF := func() {}
		ob, err := CreateEventObserver(
			EventObserverFuncs(
				func(owner string, id StateID, val int) {
					shiftF()
					t.Logf("Observed event Enter on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[0] = false
				},
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Exit on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[1] = false
				},
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Pick on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[2] = false
				},
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Update on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[3] = false
				},
			), 100*time.Millisecond, 2,
			NewObserveProtectedHook(
				func(owner string, id StateID, val int) (int, bool) {
					t.Logf("Hooked Enter on ID %d owner=%s, val=%d", id, owner, val)
					hookRet[0] = false
					return val * 10, false
				},
				func(owner string, id StateID, val int) (int, bool) {
					t.Logf("Hooked Exit on ID %d owner=%s, val=%d", id, owner, val)
					hookRet[1] = false
					return val * 10, false
				},
				func(owner string, id StateID, val int) (int, bool) {
					t.Logf("Hooked Pick on ID %d owner=%s, val=%d", id, owner, val)
					hookRet[2] = false
					return val * 10, false
				},
				func(owner string, id StateID, val int) (int, bool) {
					t.Logf("Hooked Update on ID %d owner=%s, val=%d", id, owner, val)
					hookRet[3] = false
					return val * 10, false
				},
			),
		)
		So(err, ShouldBeNil)
		So(ob, ShouldNotBeNil)

		owner := "XEventOwner"
		ob.startOb(owner, 1, 0, true)
		So(ob.(*eventObAgent[string, int]).stateID, ShouldEqual, 1)
		ob.enter(owner, 1, 1)
		So(hookRet[0], ShouldBeFalse)
		ob.exit(owner, 1, 2)
		So(hookRet[1], ShouldBeFalse)
		ob.pick(owner, 1, 3)
		So(hookRet[2], ShouldBeFalse)
		ob.update(owner, 1, 4)
		So(hookRet[3], ShouldBeFalse)
		runtime.Gosched()
		t.Log("check eventRet:", eventRet)
		So(eventRet[0], ShouldBeFalse)
		So(eventRet[1], ShouldBeFalse)
		So(eventRet[2], ShouldBeFalse)
		So(eventRet[3], ShouldBeFalse)

		Convey("Event observer timeout test", func() {
			wrCount := 0
			go func() {
				for {
					w := <-ob.Warning()
					t.Log("Got warning from observer:", w)
					wrCount++
				}
			}()
			shiftF = func() {
				time.Sleep(500 * time.Millisecond)
			}
			ob.enter(owner, 1, 4)
			ob.enter(owner, 1, 5)
			ob.enter(owner, 1, 6)
			ob.enter(owner, 1, 7)
			time.Sleep(2 * time.Second)
			// Timeout x 4 + max Blocking x 2
			So(wrCount == 6, ShouldBeTrue)
		})
	})

	// without hook
	Convey("Event observer without hook test", t, func() {
		eventRet := []bool{true, true, true, true}
		ob, err := CreateEventObserver(
			EventObserverFuncs(
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Enter on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[0] = false
				},
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Exit on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[1] = false
				},
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Pick on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[2] = false
				},
				func(owner string, id StateID, val int) {
					t.Logf("Observed event Update on ID %d owner=%s, val=%d", id, owner, val)
					eventRet[3] = false
				},
			), 0, 2, nil)
		So(err, ShouldBeNil)
		So(ob, ShouldNotBeNil)
		owner := "XEventOwner-NoHook"
		ob.startOb(owner, 2, 0, true)
		ob.enter(owner, 2, 1)
		ob.exit(owner, 2, 2)
		ob.pick(owner, 2, 3)
		ob.update(owner, 2, 4)
		runtime.Gosched()
		time.Sleep(100 * time.Millisecond)
		t.Log("check eventRet:", eventRet)
		So(eventRet[0], ShouldBeFalse)
		So(eventRet[1], ShouldBeFalse)
		So(eventRet[2], ShouldBeFalse)
		So(eventRet[3], ShouldBeFalse)
	})
}
