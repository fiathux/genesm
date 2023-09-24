package genesm

import "context"

type FrameEvent int

const (
	FEvFree   FrameEvent = iota // state is not inused
	FEvIdle                     // state inused but no changes
	FEvEnter                    // state is just insused
	FEvUpdate                   // state content updated
)

type obListener[O any, T any] interface {
	enter(owner O, id StateID, val T)
	exit(owner O, id StateID, val T)
	pick(owner O, id StateID, val T)
	update(owner O, id StateID, val T)
}

type ObserveProtectedHook[O any, T any] struct {
	Enter  func(owner O, id StateID, val T) (newval T, skip bool)
	Exit   func(owner O, id StateID, val T) (newval T, skip bool)
	Pick   func(owner O, id StateID, val T) (newval T, skip bool)
	Update func(owner O, id StateID, val T) (newval T, skip bool)
}

type EventObserver[O any, T any] interface {
	Enter(owner O, id StateID, val T)
	Exit(owner O, id StateID, val T)
	Pick(owner O, id StateID, val T)
	Update(owner O, id StateID, val T)
}

type FramesObserver[O any, T any] interface {
	Frame(owner O, evt FrameEvent, id StateID, val T)
}

func NewObserverFuncs[O any, T any](
	enter func(owner O, id StateID, val T),
	exit func(owner O, id StateID, val T),
	pick func(owner O, id StateID, val T),
	update func(owner O, id StateID, val T),
) EventObserver[O, T] {
}

func NewFrameFunc[O any, T any](
	frames func(owner O, evt FrameEvent, id StateID, val T),
) FramesObserver[O, T] {
}

type eventObCollector struct {
	ctx   context.Context
	evtCh chan func()
	evtRt chan struct{}
}

type eventObAgent[O any, T any] struct {
	parent *eventObCollector
	state  *stateBindImp
	obIf   Observer[O, T]
}
