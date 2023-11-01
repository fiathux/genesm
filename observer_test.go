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

		ctr := NewObsController(ObsControlCfg{
			Timeout: 100 * time.Millisecond, MaxBlock: 2,
		})

		ob := CreateEventObserver(
			ctr,
			ObsEventFuncs(
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
			),
			NewObserveProtectedHook(
				func(owner string, id StateID, val int) int {
					t.Logf("Hooked Init on ID %d owner=%s, val=%d", id, owner, val)
					hookRet[0] = false
					return val * 10
				},
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
		So(ob, ShouldNotBeNil)

		owner := "XEventOwner"
		ob.startOb(owner, StateID{1, 1}, 0, true)
		So(ob.(*eventObAgent[string, int]).stateID, ShouldEqual, StateID{1, 1})
		ob.enter(owner, StateID{1, 1}, 1)
		So(hookRet[0], ShouldBeFalse)
		ob.exit(owner, StateID{1, 1}, 2)
		So(hookRet[1], ShouldBeFalse)
		ob.pick(owner, StateID{1, 1}, 3)
		So(hookRet[2], ShouldBeFalse)
		ob.update(owner, StateID{1, 1}, 4)
		So(hookRet[3], ShouldBeFalse)
		time.Sleep(100 * time.Millisecond)
		t.Log("check eventRet:", eventRet)
		So(eventRet[0], ShouldBeFalse)
		So(eventRet[1], ShouldBeFalse)
		So(eventRet[2], ShouldBeFalse)
		So(eventRet[3], ShouldBeFalse)

		Convey("Event observer timeout test", func() {
			wrCount := 0
			go func() {
				for {
					w := <-ctr.Warning()
					t.Log("Got warning from observer:", w)
					wrCount++
				}
			}()
			shiftF = func() {
				time.Sleep(500 * time.Millisecond)
			}
			ob.enter(owner, StateID{1, 1}, 4)
			ob.enter(owner, StateID{1, 1}, 5)
			ob.enter(owner, StateID{1, 1}, 6)
			ob.enter(owner, StateID{1, 1}, 7)
			time.Sleep(2 * time.Second)
			// Timeout x 4 + max Blocking x 2
			So(wrCount == 6, ShouldBeTrue)
		})
	})

	// without hook
	Convey("Event observer without hook test", t, func() {
		eventRet := []bool{true, true, true, true}
		ob := CreateEventObserver(
			nil,
			ObsEventFuncs(
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
			), nil)
		So(ob, ShouldNotBeNil)
		owner := "XEventOwner-NoHook"
		ob.startOb(owner, StateID{1, 2}, 0, true)
		ob.enter(owner, StateID{1, 2}, 1)
		ob.exit(owner, StateID{1, 2}, 2)
		ob.pick(owner, StateID{1, 2}, 3)
		ob.update(owner, StateID{1, 2}, 4)
		runtime.Gosched()
		time.Sleep(100 * time.Millisecond)
		t.Log("check eventRet:", eventRet)
		So(eventRet[0], ShouldBeFalse)
		So(eventRet[1], ShouldBeFalse)
		So(eventRet[2], ShouldBeFalse)
		So(eventRet[3], ShouldBeFalse)
	})
}

func TestFrameObserver(t *testing.T) {
	Convey("Test frame observer", t, func() {
		hookRet := []bool{true, true, true, true}
		frameRet := []int{0, 0, 0}
		shiftF := func() {}

		tk, err := CreateObsFrameTicker(10)
		So(err, ShouldBeNil)
		So(tk, ShouldNotBeNil)

		ctr := NewObsController(ObsControlCfg{
			Timeout: 100 * time.Millisecond, MaxBlock: 2,
		})

		ob0 := CreateFrameObserver(
			ctr, tk,
			ObsFrameFunc(func(
				owner string, ev FrameEvent, stateID StateID, skipped int64, val int,
			) {
				t.Logf("Observed frame event %s on ID %d owner=%s, skipped=%d, val=%d",
					ev.String(), stateID, owner, skipped, val)
				frameRet[0]++
			}),
			NewObserveProtectedHook(
				func(owner string, id StateID, val int) int {
					t.Logf("Hooked Init on ID %d owner=%s, val=%d", id, owner, val)
					hookRet[0] = false
					return val * 10
				},
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
		So(ob0, ShouldNotBeNil)
		ob1 := CreateFrameObserver(
			ctr, tk,
			ObsFrameFunc(func(
				owner string, ev FrameEvent, stateID StateID, skipped int64, val int,
			) {
				t.Logf("Observed frame event %s on ID %d owner=%s, skipped=%d, val=%d",
					ev.String(), stateID, owner, skipped, val)
				frameRet[1]++
			}), nil)
		So(ob1, ShouldNotBeNil)
		ob2 := CreateFrameObserver(
			ctr, tk,
			ObsFrameFunc(func(
				owner string, ev FrameEvent, stateID StateID, skipped int64, val int,
			) {
				shiftF()
				t.Logf("Observed frame event %s on ID %d owner=%s, skipped=%d, val=%d",
					ev.String(), stateID, owner, skipped, val)
				frameRet[2]++
			}), nil)
		So(ob2, ShouldNotBeNil)

		owner := "XFrameOwner"

		ob0.startOb(owner, StateID{1, 1}, 0, true)
		ob1.startOb(owner, StateID{1, 2}, 100, false)
		ob2.startOb(owner, StateID{1, 3}, 200, false)

		time.Sleep(200 * time.Millisecond)
		ob0.enter(owner, StateID{1, 1}, 1)
		time.Sleep(200 * time.Millisecond)
		ob0.pick(owner, StateID{1, 1}, 2)
		time.Sleep(200 * time.Millisecond)
		ob0.update(owner, StateID{1, 1}, 3)
		time.Sleep(200 * time.Millisecond)
		ob0.exit(owner, StateID{1, 1}, 4)
		ob1.enter(owner, StateID{1, 2}, 101)
		time.Sleep(200 * time.Millisecond)
		ob1.pick(owner, StateID{1, 2}, 102)
		time.Sleep(200 * time.Millisecond)
		ob1.update(owner, StateID{1, 2}, 103)
		time.Sleep(200 * time.Millisecond)
		ob1.exit(owner, StateID{1, 2}, 104)
		ob2.enter(owner, StateID{1, 3}, 201)
		time.Sleep(200 * time.Millisecond)

		So(hookRet[0], ShouldBeFalse)
		So(hookRet[1], ShouldBeFalse)
		So(hookRet[2], ShouldBeFalse)
		So(hookRet[3], ShouldBeFalse)
		So(frameRet[0] > 0, ShouldBeTrue)
		So(frameRet[1] > 0, ShouldBeTrue)
		So(frameRet[2] > 0, ShouldBeTrue)
		So(frameRet[0]+frameRet[1]+frameRet[2], ShouldEqual, tk.TotalFrames())

		// Test frame skip
		warnBlock := false
		warnSkip := false
		warnTimeout := false
		warnUnknown := false
		go func() {
			for {
				w := <-ctr.Warning()
				t.Logf("Frame Ob2 warning %v", w)
				switch w.Type {
				case ObWFrameSkip:
					warnSkip = true
				case ObWFrameTimeout:
					warnTimeout = true
				case ObWMaxBlocking:
					warnBlock = true
				default:
					warnUnknown = true
				}
			}
		}()
		shiftF = func() {
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(2100 * time.Millisecond)
		So(tk.SkippedFrames(), ShouldBeGreaterThan, 0)
		shiftF = func() {}
		So(warnBlock, ShouldBeTrue)
		So(warnSkip, ShouldBeTrue)
		So(warnTimeout, ShouldBeTrue)
		So(warnUnknown, ShouldBeFalse)
		So(tk.TotalSkipped(), ShouldBeGreaterThan, 0)

		// Test ticker modify
		ob0.enter(owner, StateID{1, 1}, 9)
		time.Sleep(200 * time.Millisecond)
		tk.Stop()
		t.Log("STOPPED")
		fms := tk.TotalFrames()
		tcs := tk.TickCount()
		time.Sleep(1000 * time.Millisecond)
		So(tk.TotalFrames(), ShouldEqual, fms)
		t.Log("RESET")
		tk.Reset(20)
		time.Sleep(1010 * time.Millisecond)
		So(tk.TickCount()-tcs, ShouldEqual, 20)
		So(20-tk.TotalFrames()-fms <= 1, ShouldBeTrue)
	})
}
