package genesm

import (
	"errors"
	"sync"
	"time"
)

// A stateAgent provide several inner methods to use for interact with
// StateMachine
type stateAgent[O any] interface {
	onEnter(owner O)
	onExit(owner O)
	onPick(owner O)
}

// StateBinder is a management interface. which represent a DFA State that
// managed by StateMachine.
//
// generally StateBinder is global unique object. it have a State ID that
// assigned by StateMachine.
//
// Contained value can be any type. use Get/Set method to retrieve or update it.
// this operation could be watching by observer
type StateBinder[O any, T any] interface {
	ID() StateID
	Parent() *StateMachine[O]
	IsSelected() bool
	Get() T
	Set(val T) error
	GetUpdTime() time.Time
	Protect(handler func(owner O, v T, selected bool))
	AddObserver(obs Observer[O, T]) error
}

// stateBindImp is a inner object that implement StateBinder
type stateBindImp[O any, T any] struct {
	// IMPORTANT: If both StateMachine and here mutex are required. StateMachine
	//            MUST be lock at first. DO NOT REVERSE this order
	mux        sync.RWMutex
	id         StateID
	parent     *StateMachine[O]
	sub        T
	subUpdTime time.Time
	selected   bool
	obs        []Observer[O, T]
}

// RegState regist a state data to StateMachine and return StateBinder to do
// operations about state machine.
//
// State can be any thing. once it registed which will managed by state
// machine.
//
// As a DFA State, developer use StateMachine to manage it. contained data are
// not be care in this procedure.
//
// As a variable. StateBinder managed own values. which provide Update method
// to safely update value. also Update event will be trigger on observer
func RegState[O any, T any](sm *StateMachine[O], state T) StateBinder[O, T] {
	ret := &stateBindImp[O, T]{
		parent:     sm,
		sub:        state,
		subUpdTime: time.Now(),
	}
	sm.regState(func(id StateID) stateAgent[O] {
		ret.id = id
		if id == 0 { // add for first state into state machine
			ret.selected = true
		}
		return ret
	})
	return ret
}

// onEnter handle enter event
func (sb *stateBindImp[O, T]) onEnter(owner O) {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	sb.selected = true
	for _, ob := range sb.obs {
		ob.enter(owner, sb.id, sb.sub)
	}
}

// onExit handle exit event
func (sb *stateBindImp[O, T]) onExit(owner O) {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	sb.selected = false
	for _, ob := range sb.obs {
		ob.exit(owner, sb.id, sb.sub)
	}
}

// onPick handle pick event
func (sb *stateBindImp[O, T]) onPick(owner O) {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	for _, ob := range sb.obs {
		ob.pick(owner, sb.id, sb.sub)
	}
}

// methods to get properties

func (sb *stateBindImp[O, T]) GetUpdTime() time.Time    { return sb.subUpdTime }
func (sb *stateBindImp[O, T]) ID() StateID              { return sb.id }
func (sb *stateBindImp[O, T]) Parent() *StateMachine[O] { return sb.parent }
func (sb *stateBindImp[O, T]) IsSelected() bool         { return sb.selected }
func (sb *stateBindImp[O, T]) Get() T                   { return sb.sub }

// Protect run your function with state data and state machine under mutex
// protected
func (sb *stateBindImp[O, T]) Protect(
	handler func(owner O, v T, selected bool),
) {
	sb.parent.mux.RLock()
	sb.mux.RLock()
	defer func() {
		sb.mux.RUnlock()
		sb.parent.mux.RUnlock()
	}()
	handler(sb.parent.owner, sb.sub, sb.selected)
}

// Set use to update contain data for a State
func (sb *stateBindImp[O, T]) Set(val T) error {
	sb.mux.Lock()
	defer sb.mux.Unlock()
	sb.sub = val
	sb.subUpdTime = time.Now()
	for _, ob := range sb.obs {
		ob.update(sb.parent.owner, sb.id, sb.sub)
	}
	return nil
}

// addObserver is low-level method for append a state observer
func (sb *stateBindImp[O, T]) addObserver(obs Observer[O, T]) error {
	if err := obs.startOb(
		sb.parent.owner, sb.id, sb.sub, sb.selected); err != nil {
		return err
	}
	sb.obs = append(sb.obs, obs)
	return nil
}

// AddObserver append a state observer
func (sb *stateBindImp[O, T]) AddObserver(obs Observer[O, T]) error {
	sb.mux.Lock()
	defer sb.mux.Unlock()
	if obs == nil {
		return errors.New("observer can not be nil")
	}
	return sb.addObserver(obs)
}
