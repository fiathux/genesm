package genesm

import (
	"sync"
	"time"
)

type StateAgent[O any] interface {
	GetUpdTime() time.Time
	onEnter(owner O)
	onExit(owner O)
	onPick(owner O)
}

type StateBind[O any, T any] interface {
	StateAgent[O]
	ID() StateID
	Parent() *StateMachine[O]
	IsSelected() bool
	Get() T
	Set(val T) error
	Protect(handler func(owner O, v T, selected bool))
	AddObserver(obs StateObserver[O, T])
}

type stateBindImp[O any, T any] struct {
	// IMPORTANT: If both StateMachine and here mutex are required. StateMachine
	//            MUST be lock at first. DO NOT REVERSE this order
	mux        sync.RWMutex
	id         StateID
	parent     *StateMachine[O]
	sub        T
	subUpdTime time.Time
	selected   bool
	obs        []obListener
}

func RegState[O any, T any](sm *StateMachine[O], state T) StateBind[O, T] {
	ret := stateBind[O, T]{
		parent:     sm,
		sub:        state,
		subUpdTime: time.Now(),
	}
	sm.regState(func(id StateID) StateAgent[O] {
		ret.id = id
		if id == 0 { // add for first state into state machine
			selected = true
		}
		return &ret
	})
	return &ret
}

func (sb *stateBindImp[O, T]) onEnter(owner O) {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	sb.selected = true
	for _, obs := range sb.obs {
		ob.enter(owner, sb.sub)
	}
}
func (sb *stateBindImp[O, T]) onExit(owner O) {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	sb.selected = false
	for _, ob := range sb.obs {
		ob.exit(owner, sb.sub)
	}
}
func (sb *stateBindImp[O, T]) onPick(owner O) {
	sb.mux.RLock()
	defer sb.mux.RUnlock()
	for _, obs := range sb.obs {
		ob.pick(owner, sb.sub)
	}
}

func (sb *stateBindImp[O, T]) GetUpdTime() time.Time    { return sb.subUpdTime }
func (sb *stateBindImp[O, T]) ID() StateID              { return sb.id }
func (sb *stateBindImp[O, T]) Parent() *StateMachine[O] { return sb.parent }
func (sb *stateBindImp[O, T]) IsSelected() bool         { return sb.selected }
func (sb *stateBindImp[O, T]) Get() T                   { return sb.sub }

func (sb *stateBindImp[O, T]) Protect(
	handler func(owner O, v T, selected bool),
) {
	sb.parent.mux.RLock()
	sb.mux.RLock()
	defer func() {
		sb.mux.RUnlock()
		sb.parent.mux.RUnlock()
	}()
	return handler(sb.parent.owner, sb.sub, sb.selected)
}

func (sb *stateBindImp[O, T]) Set(val T) error {
	sb.mux.Lock()
	defer sb.mux.Unlock()
	sb.sub = val
	sb.subUpdTime = time.Now()
	for _, obs := range sb.obs {
		ob.update(owner, sb.sub)
	}
}
