package genesm

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// A FrameEvent represent time based observe event
type FrameEvent int

const (
	fEvFree   FrameEvent = iota // state is not inused (reserve by system)
	FEvIdle                     // state inused but no changes
	FEvEnter                    // state is just insused
	FEvUpdate                   // state content updated
)

// maxFrameRate is max frame rate limitation for all time based observer
const maxFrameRate = 200

// Obbserve errors
var (
	ErrObInvalidFrameRate = errors.New("invalid frame rate")
	ErrObNoBound          = errors.New("no observer bound")
	ErrObBeenBound        = errors.New("observer already bound to a state")
)

// Observer represent a observer of state machine.
//
// IMPORTANT: A Observer is run on asynchronous mode for performance. on a
// event be handled, it won't run under mutex protected. if some struct which
// not supported async access (such as `map`), you need add hook funtion on you
// create an observer. the hook is run under mutex protected. you can copy your
// data on that time. DO NOT run complex logic in hook function, it might be
// harm your performance.
//
// Even though observer is run on asynchronous mode. but event handler will be
// run serially. a event handler will be run after previous one is returned or
// previous handler timeout.
//
// To avoid memory leak, you can handle timeout warning on process each event.
// which help to check the issue.
type Observer[O any, T any] interface {
	// Warning is a channel to report warning from observer handler
	Warning() <-chan ObWarning

	startOb(owner O, id StateID, val T, selected bool) error
	enter(owner O, id StateID, val T)
	exit(owner O, id StateID, val T)
	pick(owner O, id StateID, val T)
	update(owner O, id StateID, val T)
}

// WarningType represent type of warning for handler of observer
type WarningType string

const (
	ObWEnterTimeout  WarningType = "enter_timeout"
	ObWExitTimeout   WarningType = "exit_timeout"
	ObWPickTimeout   WarningType = "pick_timeout"
	ObWUpdateTimeout WarningType = "update_timeout"
	ObWFrameTimeout  WarningType = "frame_timeout"
	ObWFrameSkip     WarningType = "frame_skipped"
	ObWMaxBlocking   WarningType = "max_hander_blocking"
)

// ObWarning is a notification to report failures on handler of observer
type ObWarning struct {
	Type    WarningType
	Ts      time.Time
	StateID StateID
}

// ObserveProtectedHook provide some hook function that run under mutex
// protected. it use for hook event to do data copy or cancel followed handler.
type ObserveProtectedHook[O any, T any] struct {
	init   func(owner O, id StateID, val T) (newval T)
	enter  func(owner O, id StateID, val T) (newval T, skip bool)
	exit   func(owner O, id StateID, val T) (newval T, skip bool)
	pick   func(owner O, id StateID, val T) (newval T, skip bool)
	update func(owner O, id StateID, val T) (newval T, skip bool)
}

func NewObserveProtectedHook[O any, T any](
	init func(owner O, id StateID, val T) (newval T),
	enter func(owner O, id StateID, val T) (newval T, skip bool),
	exit func(owner O, id StateID, val T) (newval T, skip bool),
	pick func(owner O, id StateID, val T) (newval T, skip bool),
	update func(owner O, id StateID, val T) (newval T, skip bool),
) *ObserveProtectedHook[O, T] {
	return &ObserveProtectedHook[O, T]{init, enter, exit, pick, update}
}

// EventObserver represent a event-base observer. when a event occured will
// trigger to corresponding method
type EventObserver[O any, T any] interface {
	Enter(owner O, id StateID, val T)
	Exit(owner O, id StateID, val T)
	Pick(owner O, id StateID, val T)
	Update(owner O, id StateID, val T)
}

// FramesObserver represent a time-based observer. it will trigger periodically.
type FramesObserver[O any, T any] interface {
	Frame(owner O, evt FrameEvent, id StateID, skipped int64, val T)
}

// simpleEventOb is a simple EventObserver that create from ordinary function
type simpleEventOb[O any, T any] struct {
	enter  func(owner O, id StateID, val T)
	exit   func(owner O, id StateID, val T)
	pick   func(owner O, id StateID, val T)
	update func(owner O, id StateID, val T)
}

// EventObserverFuncs create EventObserver from ordinary functions
func EventObserverFuncs[O any, T any](
	enter func(owner O, id StateID, val T),
	exit func(owner O, id StateID, val T),
	pick func(owner O, id StateID, val T),
	update func(owner O, id StateID, val T),
) EventObserver[O, T] {
	return &simpleEventOb[O, T]{
		enter:  enter,
		exit:   exit,
		pick:   pick,
		update: update,
	}
}

// simpleFrameOb is a simple FramesObserver that create from pure function
type simpleFrameOb[O any, T any] func(
	owner O, ev FrameEvent, stateID StateID, skipped int64, val T)

// FrameObserverFunc create FramesObserver from ordinary function
func FrameObserverFunc[O any, T any](
	frame func(owner O, ev FrameEvent, stateID StateID, skipped int64, val T),
) FramesObserver[O, T] {
	return simpleFrameOb[O, T](frame)
}

// eventObCollector is base struct of observer implementation
type eventObCollector struct {
	bindstat        int32
	stateID         StateID
	evtCh           chan func()
	evtRt           chan struct{}
	blockedCount    int32          // count of block handler.
	maxBlock        uint32         // max blocked handler.
	blockingTimeout time.Duration  // execute timeout for waiting a handler
	warnChan        chan ObWarning // channel for warning report
}

// eventObAgent implamented a event-based observer
type eventObAgent[O any, T any] struct {
	eventObCollector
	//state StateBinder[O, T]
	hook *ObserveProtectedHook[O, T]
	obIf EventObserver[O, T]
}

// obTickable indecate a generic frameObAgent to adapt to common ticker
type obTickable interface {
	tick(runHook func(), retHook func())
	skipWarn()
}

// FrameObTicker join various time based observers and provide common time
// ticker to trigger them.
//
// FrameObTicker only send ticker to observer that State have been selected by
// StateMachine
type FrameObTicker interface {
	// Stop stop the ticker
	Stop()

	// Reset restart or update ticker. use framerate to set new frame rate. if
	// framerate be set to zero, previous config will be used.
	Reset(framerate float32) error

	// SkippedFrame get count of skipped frames that on current frame executing
	SkippedFrame() int64
	// TotalSkipped get count of all skipped frames
	TotalSkipped() int64
	// TotalFrames get count of all frames
	TotalFrames() int64

	switchTo(ob obTickable)
}

// frameObTicker is a FrameObTicker implementation
type frameObTicker struct {
	mux          sync.RWMutex
	ticker       *time.Ticker
	ob           obTickable
	d            time.Duration
	processing   int32 // atomic tag to mark  previous frame is inprogress
	skipped      int64 // current skipped frames
	totalskipped int64 // total skipped frames
	totalframe   int64 // total frames
	skipWarnIf   func()
}

// CreateFrameObTicker create a new FrameObTicker.
//
// framerate must greater than 0.01 and less than 200.
func CreateFrameObTicker(framerate float32) (FrameObTicker, error) {
	if framerate < 0.01 || framerate > float32(maxFrameRate) {
		return nil, ErrObInvalidFrameRate
	}
	return &frameObTicker{
		d: time.Duration(1.0/framerate*1000.0) * time.Millisecond,
	}, nil
}

// frameObAgent implamented a time-based observer
type frameObAgent[O any, T any] struct {
	eventObCollector
	evmux  sync.Mutex
	hook   *ObserveProtectedHook[O, T]
	obIf   FramesObserver[O, T]
	owner  O
	val    T
	ticker FrameObTicker
	fev    FrameEvent // current frame events
}

// initObCollector initialize a eventObCollector
func initObCollector(evTimeout time.Duration, maxBlock uint32) eventObCollector {
	if maxBlock == 0 {
		maxBlock = 1
	}
	return eventObCollector{
		bindstat:        0,
		evtCh:           make(chan func(), 20),
		evtRt:           make(chan struct{}, 1),
		blockedCount:    0,
		maxBlock:        maxBlock,
		blockingTimeout: evTimeout,
		warnChan:        make(chan ObWarning, 5),
	}
}

// CreateEventObserver create a event based observer
func CreateEventObserver[O any, T any](
	ob EventObserver[O, T], evTimeout time.Duration, maxBlock uint32,
	hook *ObserveProtectedHook[O, T],
) Observer[O, T] {
	if ob == nil {
		panic("ob can not be nil")
	}
	if hook == nil {
		hook = &ObserveProtectedHook[O, T]{} // use default hook
	}
	return &eventObAgent[O, T]{
		eventObCollector: initObCollector(evTimeout, maxBlock),
		hook:             hook,
		obIf:             ob,
	}
}

// CreateFrameObserver create a time based observer
func CreateFrameObserver[O any, T any](
	ticker FrameObTicker, ob FramesObserver[O, T], evTimeout time.Duration,
	maxBlock uint32, hook *ObserveProtectedHook[O, T],
) Observer[O, T] {
	if ticker == nil {
		panic("ticker can not be nil")
	}
	if ob == nil {
		panic("ob can not be nil")
	}
	if hook == nil {
		hook = &ObserveProtectedHook[O, T]{} // use default hook
	}
	return &frameObAgent[O, T]{
		eventObCollector: initObCollector(evTimeout, maxBlock),
		hook:             hook,
		obIf:             ob,
		ticker:           ticker,
	}
}

// --------------- simple observer implemention ---------------

func (sob *simpleEventOb[O, T]) Enter(owner O, id StateID, val T) {
	if sob.enter != nil {
		sob.enter(owner, id, val)
	}
}
func (sob *simpleEventOb[O, T]) Exit(owner O, id StateID, val T) {
	if sob.exit != nil {
		sob.exit(owner, id, val)
	}
}
func (sob *simpleEventOb[O, T]) Pick(owner O, id StateID, val T) {
	if sob.pick != nil {
		sob.pick(owner, id, val)
	}
}
func (sob *simpleEventOb[O, T]) Update(owner O, id StateID, val T) {
	if sob.update != nil {
		sob.update(owner, id, val)
	}
}

func (sob simpleFrameOb[O, T]) Frame(
	owner O, evt FrameEvent, id StateID, skipped int64, val T,
) {
	sob(owner, evt, id, skipped, val)
}

// --------------- FrameObTicker impelementation ---------------

// Stop stop ticker
func (tk *frameObTicker) Stop() {
	tk.mux.RLock()
	defer tk.mux.RUnlock()
	if tk.ticker != nil {
		tk.ticker.Stop()
	}
}

// Reset reset ticker
func (tk *frameObTicker) Reset(framerate float32) error {
	if framerate < 0.01 || framerate > float32(maxFrameRate) {
		return ErrObInvalidFrameRate
	}
	tk.mux.RLock()
	defer tk.mux.RUnlock()
	if tk.ticker == nil {
		return ErrObNoBound
	}
	if framerate != 0 {
		tk.d = time.Duration(1.0/framerate*1000.0) * time.Millisecond
	}
	tk.ticker.Reset(tk.d)
	return nil
}

// SkippedFrame return current skipped frames
func (tk *frameObTicker) SkippedFrame() int64 {
	return atomic.LoadInt64(&tk.skipped)
}

// TotalSkipped return total skipped frames
func (tk *frameObTicker) TotalSkipped() int64 {
	return atomic.LoadInt64(&tk.totalskipped)
}

// TotalFrames return total frames
func (tk *frameObTicker) TotalFrames() int64 {
	return atomic.LoadInt64(&tk.totalframe)
}

// switchTo change current active observer
func (tk *frameObTicker) switchTo(ob obTickable) {
	tk.mux.Lock()
	defer tk.mux.Unlock()
	tk.ob = ob
	if tk.ticker == nil {
		// start ticker on first switching
		tk.ticker = time.NewTicker(tk.d)
		go func() {
			for {
				<-tk.ticker.C
				atomic.AddInt64(&tk.totalframe, 1)
				func() {
					tk.mux.RLock()
					defer tk.mux.RUnlock()
					if atomic.CompareAndSwapInt32(&tk.processing, 0, 1) {
						tk.skipWarnIf = tk.ob.skipWarn
						tk.ob.tick(func() { // reset skipped frame on start processing event
							atomic.StoreInt64(&tk.skipped, 0)
						}, func() { // reset processing flag on finish processing event
							atomic.StoreInt32(&tk.processing, 0)
						})
					} else if tk.skipWarnIf != nil { // skip frame
						atomic.AddInt64(&tk.skipped, 1)
						atomic.AddInt64(&tk.totalskipped, 1)
						tk.skipWarnIf()
					}
				}()
			}
		}()
	}
}

// --------------- eventObCollector methods ---------------

// warnOut send an observer warning
func (eoc *eventObCollector) warnOut(w WarningType) {
	select {
	case eoc.warnChan <- ObWarning{
		Type:    w,
		Ts:      time.Now(),
		StateID: eoc.stateID,
	}:
	default:
	}
}

// initOb init event processor
func (eoc *eventObCollector) initOb(stateID StateID) error {
	if !atomic.CompareAndSwapInt32(&eoc.bindstat, 0, 1) {
		return ErrObBeenBound
	}
	eoc.stateID = stateID
	// start event thread
	go func() {
		for {
			exec := <-eoc.evtCh
			<-eoc.evtRt
			atomic.AddInt32(&eoc.blockedCount, 1) //>>blockedCount
			go func(xc func()) {
				xc()
			}(exec)
		}
	}()
	eoc.evtRt <- struct{}{}
	return nil
}

// packEvent pack a event with timeout watching
func (eoc *eventObCollector) packEvent(
	wtimeout WarningType, f func(),
	runHook func(), retHook func(timeout bool),
) func() {
	return func() {
		timeout := false
		defer func() {
			if retHook != nil {
				retHook(timeout)
			}
			eoc.evtRt <- struct{}{}
		}()
		if runHook != nil {
			runHook()
		}
		if eoc.blockingTimeout != 0 {
			retCh := make(chan struct{})
			go func() {
				f()
				close(retCh)
				atomic.AddInt32(&eoc.blockedCount, -1) //<<blockedCount
			}()
			select {
			case <-retCh:
				return
			case <-time.After(eoc.blockingTimeout):
				timeout = true
				eoc.warnOut(wtimeout)
				if atomic.LoadInt32(&eoc.blockedCount) >= int32(eoc.maxBlock) {
					eoc.warnOut(ObWMaxBlocking)
				} else {
					return
				}
				// on max blocking, continue to execute until return
				<-retCh
			}
		} else {
			f()
			atomic.AddInt32(&eoc.blockedCount, -1) //<<blockedCount
		}
	}
}

// Warning retrieve a channel to receive observer warning
func (eoc *eventObCollector) Warning() <-chan ObWarning {
	return eoc.warnChan
}

// --------------- EventObserver implementation ---------------

func (eoa *eventObAgent[O, T]) startOb(
	owner O, id StateID, val T, selected bool,
) error {
	return eoa.initOb(id)
}

func (eoa *eventObAgent[O, T]) enter(owner O, id StateID, val T) {
	var newval T
	skip := false
	if eoa.hook != nil && eoa.hook.enter != nil {
		newval, skip = eoa.hook.enter(owner, id, val)
		if skip {
			return
		}
	} else {
		newval = val
	}
	fadv := eoa.packEvent(ObWEnterTimeout, func() {
		eoa.obIf.Enter(owner, id, newval)
	}, nil, nil)
	eoa.evtCh <- fadv
}

func (eoa *eventObAgent[O, T]) exit(owner O, id StateID, val T) {
	var newval T
	skip := false
	if eoa.hook != nil && eoa.hook.exit != nil {
		newval, skip = eoa.hook.exit(owner, id, val)
		if skip {
			return
		}
	} else {
		newval = val
	}
	fadv := eoa.packEvent(ObWExitTimeout, func() {
		eoa.obIf.Exit(owner, id, newval)
	}, nil, nil)
	eoa.evtCh <- fadv
}

func (eoa *eventObAgent[O, T]) pick(owner O, id StateID, val T) {
	var newval T
	skip := false
	if eoa.hook != nil && eoa.hook.pick != nil {
		newval, skip = eoa.hook.pick(owner, id, val)
		if skip {
			return
		}
	} else {
		newval = val
	}
	fadv := eoa.packEvent(ObWPickTimeout, func() {
		eoa.obIf.Pick(owner, id, newval)
	}, nil, nil)
	eoa.evtCh <- fadv
}

func (eoa *eventObAgent[O, T]) update(owner O, id StateID, val T) {
	var newval T
	skip := false
	if eoa.hook != nil && eoa.hook.update != nil {
		newval, skip = eoa.hook.update(owner, id, val)
		if skip {
			return
		}
	} else {
		newval = val
	}
	fadv := eoa.packEvent(ObWUpdateTimeout, func() {
		eoa.obIf.Update(owner, id, newval)
	}, nil, nil)
	eoa.evtCh <- fadv
}

// --------------- FrameObserver implementation ---------------

// skipWarn implement obTickable interface
func (foa *frameObAgent[O, T]) skipWarn() {
	foa.warnOut(ObWFrameSkip)
}

// tick implement obTickable interface
func (foa *frameObAgent[O, T]) tick(runHook func(), retHook func()) {
	fadv := foa.packEvent(ObWFrameTimeout, func() {
		skipped := foa.ticker.SkippedFrame()
		ev := foa.resetEv()
		runHook()
		foa.obIf.Frame(foa.owner, ev, foa.stateID, skipped, foa.val)
	}, nil, func(timeout bool) {
		retHook()
	})
	foa.evtCh <- fadv
}

// updateEv set a key-frame
func (foa *frameObAgent[O, T]) updateEv(ev FrameEvent) {
	foa.evmux.Lock()
	defer foa.evmux.Unlock()
	foa.fev = ev
}

// resetEv set a tansit frame
func (foa *frameObAgent[O, T]) resetEv() FrameEvent {
	foa.evmux.Lock()
	defer foa.evmux.Unlock()
	ev := foa.fev
	foa.fev = FEvIdle
	return ev
}

func (foa *frameObAgent[O, T]) startOb(
	owner O, id StateID, val T, selected bool,
) error {
	if err := foa.initOb(id); err != nil {
		return err
	}
	foa.owner = owner
	if selected {
		if foa.hook != nil && foa.hook.init != nil {
			foa.val = foa.hook.init(owner, id, val)
		} else {
			foa.val = val
		}
		foa.updateEv(FEvEnter)
		foa.ticker.switchTo(foa)
	}
	return nil
}

func (foa *frameObAgent[O, T]) enter(owner O, id StateID, val T) {
	if foa.hook != nil && foa.hook.enter != nil {
		val, skip := foa.hook.enter(owner, id, val)
		if !skip {
			foa.val = val
		} else {
			return
		}
	} else {
		foa.val = val
	}
	foa.owner = owner
	foa.updateEv(FEvEnter)
	foa.ticker.switchTo(foa)
}

func (foa *frameObAgent[O, T]) exit(owner O, id StateID, val T) {
	if foa.hook != nil && foa.hook.exit != nil {
		val, skip := foa.hook.exit(owner, id, val)
		if !skip {
			foa.val = val
		} else {
			return
		}
	} else {
		foa.val = val
	}
	foa.owner = owner
}

func (foa *frameObAgent[O, T]) pick(owner O, id StateID, val T) {
	if foa.hook != nil && foa.hook.pick != nil {
		val, skip := foa.hook.pick(owner, id, val)
		if !skip {
			foa.val = val
		} else {
			return
		}
	} else {
		foa.val = val
	}
	foa.owner = owner
}

func (foa *frameObAgent[O, T]) update(owner O, id StateID, val T) {
	if foa.hook != nil && foa.hook.update != nil {
		val, skip := foa.hook.update(owner, id, val)
		if !skip {
			foa.val = val
		} else {
			return
		}
	} else {
		foa.val = val
	}
	foa.owner = owner
	foa.updateEv(FEvUpdate)
}

// --------------- FrameEvent ---------------

// String return string representation of FrameEvent
func (fe FrameEvent) String() string {
	switch fe {
	case fEvFree:
		return "Free"
	case FEvIdle:
		return "Idle"
	case FEvEnter:
		return "Enter"
	case FEvUpdate:
		return "Update"
	default:
		return "Unknown"
	}
}
