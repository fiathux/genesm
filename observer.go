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

// errors
var ErrObInvalidFrameRate = errors.New("invalid frame rate")

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

	enter(owner O, id StateID, val T)
	exit(owner O, id StateID, val T)
	pick(owner O, id StateID, val T)
	update(owner O, id StateID, val T)
	startOb(owner O, id StateID, val T, selected bool) error
}

// WarningType represent type of warning for handler of observer
type WarningType string

const (
	ObWEnterTimeout  WarningType = "enter_timeout"
	ObWExitTimeout   WarningType = "exit_timeout"
	ObWPickTimeout   WarningType = "pick_timeout"
	ObWUpdateTimeout WarningType = "update_timeout"
	ObWFrameSkip     WarningType = "frame_skiped"
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
	Enter  func(owner O, id StateID, val T) (newval T, skip bool)
	Exit   func(owner O, id StateID, val T) (newval T, skip bool)
	Pick   func(owner O, id StateID, val T) (newval T, skip bool)
	Update func(owner O, id StateID, val T) (newval T, skip bool)
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
	Frame(owner O, evt FrameEvent, id StateID, skiped int, val T)
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
	owner O, ev FrameEvent, stateID StateID, skiped int, val T)

// FrameObserverFunc create FramesObserver from ordinary function
func FrameObserverFunc[O any, T any](
	frame func(owner O, ev FrameEvent, stateID StateID, skip int, val T),
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
	tick(time.Time)
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
	Reset(framerate int) error

	switchTo(ob obTickable)
}

// frameObTicker is a FrameObTicker implementation
type frameObTicker struct {
	mux    sync.RWMutex
	ticker *time.Ticker
	ob     obTickable
	d      time.Duration
}

// CreateFrameObTicker create a new FrameObTicker.
//
// framerate must greater than 0 and less than 200.
func CreateFrameObTicker(framerate int) (FrameObTicker, error) {
	if framerate <= 0 || framerate > maxFrameRate {
		return nil, ErrObInvalidFrameRate
	}
	return &frameObTicker{
		d: time.Duration(1.0/float32(framerate)*1000.0) * time.Millisecond,
	}, nil
}

// frameObAgent implamented a time-based observer
type frameObAgent[O any, T any] struct {
	eventObCollector
	//state       StateBinder[O, T]
	hook        *ObserveProtectedHook[O, T]
	obIf        FramesObserver[O, T]
	val         T
	ticker      FrameObTicker
	fev         FrameEvent // current frame events
	processing  int32      // atomic tag to mark  previous frame is inprogress
	skiped      int        // current skiped frames
	totalskiped int        // total skiped frames
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
) (Observer[O, T], error) {
	if ob == nil {
		return nil, errors.New("ob can not be nil")
	}
	obAgt := &eventObAgent{
		eventObCollector: initObCollector(evTimeout, maxBlock),
		hook:             hook,
		obIf:             ob,
	}
	//return nil, nil
}

// CreateFrameObserver create a time based observer
func CreateFrameObserver[O any, T any](
	ticker FrameObTicker, ob FramesObserver[O, T], evTimeout time.Duration,
	maxBlock uint32, hook *ObserveProtectedHook[O, T],
) (Observer[O, T], error) {
	if ticker == nil {
		return nil, errors.New("ticker can not be nil")
	}
	obAgt := &frameObAgent{
		eventObCollector: initObCollector(evTimeout, maxBlock),
		hook:             hook,
		obIf:             ob,
		ticker:           ticker,
		skiped:           0,
		totalskiped:      0,
	}
	//return nil, nil
}

// --------------- simple observer implemention ---------------

func (sob *simpleEventOb[O, T]) Enter(owner O, id StateID, val T) {
	if sob.enter != nil {
		sob.enter(owner, id, val)
	}
}
func (sob *simpleEventOb[O, T]) Exit(owner O, id StateID, val T) {
	if sob.exit != nil {
		sob.enter(owner, id, val)
	}
}
func (sob *simpleEventOb[O, T]) Pick(owner O, id StateID, val T) {
	if sob.pick != nil {
		sob.enter(owner, id, val)
	}
}
func (sob *simpleEventOb[O, T]) Update(owner O, id StateID, val T) {
	if sob.update != nil {
		sob.enter(owner, id, val)
	}
}

func (sob simpleFrameOb[O, T]) Frame(
	owner O, evt FrameEvent, id StateID, skiped int, val T,
) {
	sob(owner, evt, id, skiped, val)
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
func (tk *frameObTicker) Reset(framerate int) error {
	if framerate < 0 || framerate > maxFrameRate {
		return ErrObInvalidFrameRate
	}
	tk.mux.RLock()
	defer tk.mux.RUnlock()
	if tk.ticker == nil {
		return errors.New("no observer bound")
	}
	if framerate != 0 {
		tk.d = time.Duration(1.0/float32(framerate)*1000.0) * time.Millisecond
	}
	tk.ticker.Reset(tk.d)
	return nil
}

// switchTo change current active observer
func (tk *frameObTicker) switchTo(ob obTickable) {
	tk.mux.Lock()
	defer tk.mux.Unlock()
	tk.ob = ob
	if tk.ticker == nil {
		tk.ticker = time.NewTicker(tk.d)
		go func() {
			for {
				<-tk.ticker.C
				func() {
					tk.mux.RLock()
					defer tk.mux.RUnlock()
					ob.tick(time.Now())
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

// startOb init event processor
func (eoc *eventObCollector) startOb(stateID StateID) error {
	if !atomic.CompareAndSwapInt32(&eoc.bindstat, 0, 1) {
		return errors.New("observer already bound to a state")
	}
	eoc.stateID = stateID
	// start event thread
	go func() {
		for {
			exec := <-eoc.evtCh
			<-eoc.evtRt
			if atomic.LoadInt32(&eoc.blockedCount) >= int32(eoc.maxBlock) {
				eoc.warnOut(ObWMaxBlocking)
				continue
			}
			go func() {
				atomic.AddInt32(&eoc.blockedCount, 1)
				defer atomic.AddInt32(&eoc.blockedCount, -1)
				exec()
			}()
		}
	}()
	return nil
}

// packEvent pack a event with timeout watching
func (eoc *eventObCollector) packEvent(wtimeout WarningType, f func()) func() {
	return func() {
		defer func() {
			eoc.evtRt <- struct{}{}
		}()
		if eoc.blockingTimeout != 0 {
			retCh := make(chan struct{})
			go func() {
				f()
				close(retCh)
			}()
			select {
			case <-retCh:
				return
			case <-time.After(eoc.blockingTimeout):
				eoc.warnOut(wtimeout)
			}
		} else {
			f()
		}
	}
}

func (eoc *eventObCollector) Warning() <-chan ObWarning {
	return eoc.warnChan
}

// --------------- EventObserver implementation ---------------

// --------------- FrameObserver implementation ---------------
