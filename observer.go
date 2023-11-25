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
// Even though observer is run on asynchronous mode. but event handler will be
// run serially. a event handler will be run after previous one is returned or
// previous handler timeout.
//
// To avoid memory leak, you can handle timeout warning on process each event.
// which help to check the issue.
type Observer[O any, T any] interface {
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
//
// A Observer is run on asynchronous mode for performance. on a
// event be handled, it won't run under mutex protected. if some struct which
// not supported async access (such as `map`), you shall add hook funtion on
// you create an observer. the hook is run under mutex protected. you can copy
// your data on that time.
//
// DO NOT run complex logic in hook function, it might be harm your performance.
type ObserveProtectedHook[O any, T any] struct {
	init   func(owner O, id StateID, val T) (newval T)
	enter  func(owner O, id StateID, val T) (newval T, skip bool)
	exit   func(owner O, id StateID, val T) (newval T, skip bool)
	pick   func(owner O, id StateID, val T) (newval T, skip bool)
	update func(owner O, id StateID, val T) (newval T, skip bool)
}

// NewObserveProtectedHook create a new ObserveProtectedHook
func NewObserveProtectedHook[O any, T any](
	init func(owner O, id StateID, val T) (newval T),
	enter func(owner O, id StateID, val T) (newval T, skip bool),
	exit func(owner O, id StateID, val T) (newval T, skip bool),
	pick func(owner O, id StateID, val T) (newval T, skip bool),
	update func(owner O, id StateID, val T) (newval T, skip bool),
) *ObserveProtectedHook[O, T] {
	return &ObserveProtectedHook[O, T]{init, enter, exit, pick, update}
}

// ObsHandlerEvent represent handlers of event-base observer. developer need
// implement this interface to handle event of state transition
type ObsHandlerEvent[O any, T any] interface {
	Enter(owner O, id StateID, val T)
	Exit(owner O, id StateID, val T)
	Pick(owner O, id StateID, val T)
	Update(owner O, id StateID, val T)
}

// ObsHandlerFrames represent handler of time-based observer. developer need
// implement this interface to handle each frame
type ObsHandlerFrames[O any, T any] interface {
	Frame(owner O, evt FrameEvent, id StateID, skipped int64, val T)
}

// simpleEventOb is a simple ObsHandlerEvent implementation that create from
// ordinary function
type simpleEventOb[O any, T any] struct {
	enter  func(owner O, id StateID, val T)
	exit   func(owner O, id StateID, val T)
	pick   func(owner O, id StateID, val T)
	update func(owner O, id StateID, val T)
}

// ObsEventFuncs create ObsHandlerEvent from ordinary functions
func ObsEventFuncs[O any, T any](
	enter func(owner O, id StateID, val T),
	exit func(owner O, id StateID, val T),
	pick func(owner O, id StateID, val T),
	update func(owner O, id StateID, val T),
) ObsHandlerEvent[O, T] {
	return &simpleEventOb[O, T]{
		enter:  enter,
		exit:   exit,
		pick:   pick,
		update: update,
	}
}

// simpleFrameOb is a simple ObsHandlerFrames that create from pure function
type simpleFrameOb[O any, T any] func(
	owner O, ev FrameEvent, stateID StateID, skipped int64, val T)

// ObserverFrameFunc create ObsHandlerFrames from ordinary function
func ObsFrameFunc[O any, T any](
	frame func(owner O, ev FrameEvent, stateID StateID, skipped int64, val T),
) ObsHandlerFrames[O, T] {
	return simpleFrameOb[O, T](frame)
}

// ObsController represent a observer controller. it use for control execute of
// observers that handle event from different state, keep them run serially and
// safely.
type ObsController interface {
	// Warning return a channel that report warning of handler
	Warning() <-chan ObWarning

	run(func())
	packEvent(
		stateID StateID, wtimeout WarningType,
		f func(), runHook func(), retHook func(timeout bool),
	) func()
	warn(WarningType, StateID)
}

// ObsControlCfg is config of ObsController
//
// Timeout is a timeout for waiting a handler to execute. if a handler
// blocked more than this time, it will be report as a warning. handler will
// be continue blocked until it be executed, but next event will be handled.
//
// if Timeout is zero, handler will be blocked until it return.
//
// MaxBlock is a max count of handler that can be blocked. if count of blocked
// handler reach this value, it will be report as a warning. in this case nex
// event will not be handled, until a previous handler be released.
//
// Minimum value of maxBlock is 1. pass zero to maxBlock will be treat as 1.
//
// SizeEventQueue is size of event execute queue. if a handler of event is
// blocked, next event will waiting in the queue until previous handler return
// or timeout. if queue is full, whole event chain under state machine will be
// blocked. default value of SizeEventQueue is 5.
//
// SizeWarnChan is length of channel to report warning. default value is 3. if
// channel is full, the message of warning will be lost.
type ObsControlCfg struct {
	Timeout        time.Duration
	MaxBlock       uint32
	SizeEventQueue uint32
	SizeWarnChan   uint32
}

// obsControllerImpl is a implementation of ObsController
type obsControllerImpl struct {
	evtCh           chan func()
	evtRt           chan struct{}
	blockedCount    int32          // count of block handler.
	maxBlock        uint32         // max blocked handler.
	blockingTimeout time.Duration  // execute timeout for waiting a handler
	warnChan        chan ObWarning // channel for warning report
}

// NewObsController create a new ObsController
func NewObsController(cfg ObsControlCfg) ObsController {
	if cfg.MaxBlock == 0 {
		cfg.MaxBlock = 1
	}
	if cfg.SizeEventQueue == 0 {
		cfg.SizeEventQueue = 5
	}
	if cfg.SizeWarnChan == 0 {
		cfg.SizeWarnChan = 3
	}
	ret := &obsControllerImpl{
		evtCh:           make(chan func(), cfg.SizeEventQueue),
		evtRt:           make(chan struct{}, 1),
		blockingTimeout: cfg.Timeout,
		maxBlock:        cfg.MaxBlock,
		warnChan:        make(chan ObWarning, cfg.SizeWarnChan),
	}
	ret.init()
	return ret
}

// obsSyncControllerImpl is a synchonous ObsController implementation
type obsSyncControllerImpl struct {
	warnChan chan ObWarning // channel for warning report
}

// NewObsSyncController create a new synchonous ObsController.
//
// synchonous ObsController is directly handle event of observer. which have
// minimal delay, but will block thread of state machine.
//
// An exception is time based observer (frame observer). it have own thread to
// trigger frames. so if you don't care about tick timeout, synchonous
// ObsController will have higher performance.
func NewObsSyncController(sizeWarnChan uint32) ObsController {
	if sizeWarnChan == 0 {
		sizeWarnChan = 3
	}
	return &obsSyncControllerImpl{
		warnChan: make(chan ObWarning, sizeWarnChan),
	}
}

// eventObCollector is base struct of observer implementation
type eventObCollector struct {
	bindstat int32
	stateID  StateID
	ctr      ObsController
}

// eventObAgent implamented a event-based observer
type eventObAgent[O any, T any] struct {
	eventObCollector
	//state StateBinder[O, T]
	hook *ObserveProtectedHook[O, T]
	obIf ObsHandlerEvent[O, T]
}

// obTickable indecate a generic frameObAgent to adapt to common ticker
type obTickable interface {
	tick(runHook func(), retHook func())
	skipWarn()
}

// ObsFrameTicker join various time based observers and provide common time
// ticker to trigger them.
//
// ObsFrameTicker only send ticker to observer that State have been selected by
// StateMachine
type ObsFrameTicker interface {
	// Stop stop the ticker
	Stop()

	// Reset restart or update ticker. use framerate to set new frame rate. if
	// framerate be set to zero, previous config will be used.
	Reset(framerate float32) error

	// SkippedFrames get count of skipped frames that on current frame executing
	SkippedFrames() int64
	// TotalSkipped get count of all skipped frames
	TotalSkipped() int64
	// TotalFrames get count of executed frames
	TotalFrames() int64
	// TotalFrames get count of ticks after ticker been created
	TickCount() int64

	switchTo(ob obTickable, stateID StateID)
}

// obsFrameTicker is a ObsFrameTicker implementation
type obsFrameTicker struct {
	mux          sync.RWMutex
	ticker       *time.Ticker
	obs          map[uint32]obTickable
	d            time.Duration
	processing   int32 // atomic tag to mark  previous frame is inprogress
	skipped      int64 // current skipped frames
	totalskipped int64 // total skipped frames
	totalframe   int64 // total executed frames
	tickcount    int64 // count of ticks
}

// CreateObsFrameTicker create a new ObsFrameTicker.
//
// framerate must greater than 0.01 and less than 200.
func CreateObsFrameTicker(framerate float32) (ObsFrameTicker, error) {
	if framerate < 0.01 || framerate > float32(maxFrameRate) {
		return nil, ErrObInvalidFrameRate
	}
	return &obsFrameTicker{
		obs: make(map[uint32]obTickable),
		d:   time.Duration(1.0/framerate*1000.0) * time.Millisecond,
	}, nil
}

// frameObAgent implamented a time-based observer
type frameObAgent[O any, T any] struct {
	eventObCollector
	evmux  sync.Mutex
	hook   *ObserveProtectedHook[O, T]
	obIf   ObsHandlerFrames[O, T]
	owner  O
	val    T
	ticker ObsFrameTicker
	fev    FrameEvent // current frame events
}

// CreateEventObserver create a event based observer
func CreateEventObserver[O any, T any](
	ctrl ObsController, ob ObsHandlerEvent[O, T],
	hook *ObserveProtectedHook[O, T],
) Observer[O, T] {
	if ob == nil {
		panic("ob can not be nil")
	}
	if ctrl == nil {
		ctrl = NewObsController(ObsControlCfg{}) // use separate controller
	}
	if hook == nil {
		hook = &ObserveProtectedHook[O, T]{} // use default hook
	}
	return &eventObAgent[O, T]{
		eventObCollector: eventObCollector{
			ctr: ctrl,
		},
		hook: hook,
		obIf: ob,
	}
}

// CreateFrameObserver create a time based observer
func CreateFrameObserver[O any, T any](
	ctrl ObsController, ticker ObsFrameTicker, ob ObsHandlerFrames[O, T],
	hook *ObserveProtectedHook[O, T],
) Observer[O, T] {
	if ticker == nil {
		panic("ticker can not be nil")
	}
	if ob == nil {
		panic("ob can not be nil")
	}
	if ctrl == nil {
		ctrl = NewObsController(ObsControlCfg{}) // use separate controller
	}
	if hook == nil {
		hook = &ObserveProtectedHook[O, T]{} // use default hook
	}
	return &frameObAgent[O, T]{
		eventObCollector: eventObCollector{
			ctr: ctrl,
		},
		hook:   hook,
		obIf:   ob,
		ticker: ticker,
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

// --------------- ObsFrameTicker impelementation ---------------

// Stop stop ticker
func (tk *obsFrameTicker) Stop() {
	tk.mux.RLock()
	defer tk.mux.RUnlock()
	if tk.ticker != nil {
		tk.ticker.Stop()
	}
}

// Reset reset ticker
func (tk *obsFrameTicker) Reset(framerate float32) error {
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
func (tk *obsFrameTicker) SkippedFrames() int64 {
	return atomic.LoadInt64(&tk.skipped)
}

// TotalSkipped return total skipped frames
func (tk *obsFrameTicker) TotalSkipped() int64 {
	return atomic.LoadInt64(&tk.totalskipped)
}

// TotalFrames return total executed frames
func (tk *obsFrameTicker) TotalFrames() int64 {
	return atomic.LoadInt64(&tk.totalframe)
}

// TickCount return count of ticks
func (tk *obsFrameTicker) TickCount() int64 {
	return atomic.LoadInt64(&tk.totalframe)
}

// switchTo change current active observer
func (tk *obsFrameTicker) switchTo(ob obTickable, stateID StateID) {
	tk.mux.Lock()
	defer tk.mux.Unlock()
	tk.obs[stateID.SMSerial] = ob
	if tk.ticker == nil {
		// start ticker on first switching
		tk.ticker = time.NewTicker(tk.d)
		go func() {
			for {
				<-tk.ticker.C
				atomic.StoreInt64(&tk.tickcount, 0)
				func() {
					tk.mux.RLock()
					defer tk.mux.RUnlock()
					if len(tk.obs) == 0 {
						return
					}
					if atomic.CompareAndSwapInt32(&tk.processing, 0, int32(len(tk.obs))) {
						// check whether all observers are been trigged and clean frame
						// skiped counter
						resetSkip := func(countOb int32) func() {
							return func() {
								if atomic.AddInt32(&countOb, -1) == 0 {
									atomic.AddInt64(&tk.totalframe, 1)
									atomic.StoreInt64(&tk.skipped, 0)
								}
							}
						}(int32(len(tk.obs)))
						for _, ob := range tk.obs {
							ob.tick(resetSkip,
								func() { // reset processing flag on finish processing event
									atomic.AddInt32(&tk.processing, -1)
								})
						}
					} else { // skip frame
						atomic.AddInt64(&tk.skipped, 1)
						atomic.AddInt64(&tk.totalskipped, 1)
						for _, ob := range tk.obs {
							ob.skipWarn()
						}
					}
				}()
			}
		}()
	}
}

// --------------- ObsController implementation ---------------

// init initialize controller
func (ctrl *obsControllerImpl) init() {
	// start event thread
	go func() {
		for {
			exec := <-ctrl.evtCh
			<-ctrl.evtRt
			atomic.AddInt32(&ctrl.blockedCount, 1) //>>blockedCount
			go func(xc func()) {
				xc()
			}(exec)
		}
	}()
	ctrl.evtRt <- struct{}{}
}

// run run a function in observer thread
func (ctrl *obsControllerImpl) run(f func()) {
	ctrl.evtCh <- f
}

// warn send a warning
func (ctrl *obsControllerImpl) warn(w WarningType, stateID StateID) {
	select {
	case ctrl.warnChan <- ObWarning{
		Type:    w,
		Ts:      time.Now(),
		StateID: stateID,
	}:
	default:
	}
}

// packEvent pack a event with timeout watching
func (ctrl *obsControllerImpl) packEvent(
	stateID StateID, wtimeout WarningType, f func(),
	runHook func(), retHook func(timeout bool),
) func() {
	return func() {
		timeout := false
		defer func() {
			if retHook != nil {
				retHook(timeout)
			}
			ctrl.evtRt <- struct{}{}
		}()
		if runHook != nil {
			runHook()
		}
		if ctrl.blockingTimeout != 0 {
			retCh := make(chan struct{})
			go func() {
				f()
				close(retCh)
				atomic.AddInt32(&ctrl.blockedCount, -1) //<<blockedCount
			}()
			select {
			case <-retCh:
				return
			case <-time.After(ctrl.blockingTimeout):
				timeout = true
				ctrl.warn(wtimeout, stateID)
				if atomic.LoadInt32(&ctrl.blockedCount) >= int32(ctrl.maxBlock) {
					ctrl.warn(ObWMaxBlocking, stateID)
				} else {
					return
				}
				// on max blocking, continue to execute until return
				<-retCh
			}
		} else {
			f()
			atomic.AddInt32(&ctrl.blockedCount, -1) //<<blockedCount
		}
	}
}

// Warning retrieve a channel to receive observer warning
func (ctrl *obsControllerImpl) Warning() <-chan ObWarning {
	return ctrl.warnChan
}

// --------------- ObsController implementation ---------------

// init initialize controller
func (sctrl *obsSyncControllerImpl) init() {
}

// run run a function directly
func (sctrl *obsSyncControllerImpl) run(f func()) {
	f()
}

// warn send a warning
func (sctrl *obsSyncControllerImpl) warn(w WarningType, stateID StateID) {
	select {
	case sctrl.warnChan <- ObWarning{
		Type:    w,
		Ts:      time.Now(),
		StateID: stateID,
	}:
	default:
	}
}

// packEvent pack a event function
func (sctrl *obsSyncControllerImpl) packEvent(
	stateID StateID, wtimeout WarningType, f func(),
	runHook func(), retHook func(timeout bool),
) func() {
	return func() {
		defer func() {
			if retHook != nil {
				retHook(false)
			}
		}()
		if runHook != nil {
			runHook()
		}
		f()
	}
}

// Warning retrieve a channel to receive observer warning
func (sctrl *obsSyncControllerImpl) Warning() <-chan ObWarning {
	return sctrl.warnChan
}

// --------------- eventObCollector methods ---------------

// initOb init event processor
func (eoc *eventObCollector) initOb(stateID StateID) error {
	if !atomic.CompareAndSwapInt32(&eoc.bindstat, 0, 1) {
		return ErrObBeenBound
	}
	eoc.stateID = stateID
	return nil
}

// --------------- Event based Observer implementation ---------------

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
	eoa.ctr.run(eoa.ctr.packEvent(eoa.stateID, ObWEnterTimeout, func() {
		eoa.obIf.Enter(owner, id, newval)
	}, nil, nil))
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
	eoa.ctr.run(eoa.ctr.packEvent(eoa.stateID, ObWExitTimeout, func() {
		eoa.obIf.Exit(owner, id, newval)
	}, nil, nil))
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
	eoa.ctr.run(eoa.ctr.packEvent(eoa.stateID, ObWPickTimeout, func() {
		eoa.obIf.Pick(owner, id, newval)
	}, nil, nil))
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
	eoa.ctr.run(eoa.ctr.packEvent(eoa.stateID, ObWUpdateTimeout, func() {
		eoa.obIf.Update(owner, id, newval)
	}, nil, nil))
}

// --------------- Time based Observer implementation ---------------

// skipWarn implement obTickable interface
func (foa *frameObAgent[O, T]) skipWarn() {
	foa.ctr.warn(ObWFrameSkip, foa.stateID)
}

// tick implement obTickable interface
func (foa *frameObAgent[O, T]) tick(runHook func(), retHook func()) {
	foa.ctr.run(foa.ctr.packEvent(foa.stateID, ObWFrameTimeout, func() {
		skipped := foa.ticker.SkippedFrames()
		ev := foa.resetEv()
		runHook()
		foa.obIf.Frame(foa.owner, ev, foa.stateID, skipped, foa.val)
	}, nil, func(timeout bool) {
		retHook()
	}))
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
		foa.ticker.switchTo(foa, id)
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
	foa.ticker.switchTo(foa, id)
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
