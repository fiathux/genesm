package genesm

import (
	"errors"
	"sync"
)

var (
	ErrEvAlreadyChanged  = errors.New("already on target state")
	ErrEvInvalidChange   = errors.New("invalid target state to change")
	ErrEvNothingTodo     = errors.New("nothing to change")
	ErrEvUnexpectedState = errors.New("unexpected current state")

	ErrNoState = errors.New("no status in state machine")
)

type StateID int

const STIDInvalid StateID = -1

type StateMachine[O any] struct {
	mux      sync.RWMutex
	owner    O
	stateTab []StateAgent[O]
	stateOn  StateID
}

func NewStateMachine[O any](owner O) *StateMachine[O] {
	return &StateMachine[O]{
		owner: owner,
	}
}

func (sm *StateMachine[O]) GetOwner() O {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	return sm.owner
}

func (sm *StateMachine[O]) SetOwner(o O) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.owner = o
}

func (sm *StateMachine[O]) StateID() StateID {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	if len(sm.stateTab) == 0 {
		return STIDInvalid
	}
	return sm.stateOn
}

func (sm *StateMachine[O]) PickState() error {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	if len(sm.stateTab) == 0 {
		return ErrNoState
	}
	sm.stateTab[sm.stateOn].onPick(sm.owner)
	return nil
}

func (sm *StateMachine[O]) regState(convert func(StateID) StateAgent[O]) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	s := convert(StateID(len(sm.stateTab)))
	sm.stateTab = append(sm.stateTab, s)
}

func (sm *StateMachine[O]) transform(trs func(StateID) StateID) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	next := trs(sm.stateOn)
	if next == sm.stateOn {
		return ErrEvNothingTodo
	} else if next < 0 {
		return nil
	} else if next >= StateID(len(sm.stateTab)) {
		return ErrEvInvalidChange
	}
	sm.stateTab[sm.stateOn].onExit(sm.owner)
	sm.stateOn = next
	sm.stateTab[next].onEnter(sm.owner)
}
