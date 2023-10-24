package genesm

import (
	"errors"
	"sync"
)

// state machine errors
var (
	ErrEvAlreadyChanged  = errors.New("already on target state")
	ErrEvInvalidChange   = errors.New("invalid target state to change")
	ErrEvNothingTodo     = errors.New("nothing to change")
	ErrEvUnexpectedState = errors.New("unexpected current state")

	ErrNoState = errors.New("no status in state machine")
)

// StateID is serial number to identify a registed state
type StateID int

const STIDInvalid StateID = -1

// A StateMachine is use for manage DFA State objects.
//
// A DFA State object should be global unique. which be register and managed by
// StateMachine.
//
// StateMachine will select a State object to current State. once some action
// occured it will try automatically change current State to next one and keep
// it select until next action be trigger.
//
// all methods which StateMachine exported are thread-safe.
type StateMachine[O any] struct {
	mux      sync.RWMutex
	owner    O
	stateTab []stateAgent[O]
	stateOn  StateID
}

// NewStateMachine create a new state machine instance
func NewStateMachine[O any](owner O) *StateMachine[O] {
	return &StateMachine[O]{
		owner: owner,
	}
}

// GetOwner get owner object for state machine
func (sm *StateMachine[O]) GetOwner() O {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	return sm.owner
}

// SetOwner set new owner to state matchine
func (sm *StateMachine[O]) SetOwner(o O) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	sm.owner = o
}

// StateID get state ID of seleted state
func (sm *StateMachine[O]) StateID() StateID {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	if len(sm.stateTab) == 0 {
		return STIDInvalid
	}
	return sm.stateOn
}

// PickState trigger a Pick action on current selected state
func (sm *StateMachine[O]) PickState() error {
	sm.mux.RLock()
	defer sm.mux.RUnlock()
	if len(sm.stateTab) == 0 {
		return ErrNoState
	}
	sm.stateTab[sm.stateOn].onPick(sm.owner)
	return nil
}

// regState regist a new state to state machine
//
// the convert is the constructor of state. state matchine will pass new state
// ID to create state binder. and a stateAgent interface need return from the
// constructor.
func (sm *StateMachine[O]) regState(convert func(StateID) stateAgent[O]) {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	s := convert(StateID(len(sm.stateTab)))
	sm.stateTab = append(sm.stateTab, s)
}

// transform do state transform
//
// pass the argument trs to do transform from current state ID to new state ID.
// if transform is succeed, it return new state ID, else it return a nagtive
// number to break.
func (sm *StateMachine[O]) transform(trs func(StateID) StateID) error {
	sm.mux.Lock()
	defer sm.mux.Unlock()
	next := trs(sm.stateOn)
	if next == sm.stateOn { // transform is done before
		return ErrEvNothingTodo
	} else if next < 0 { // break transform
		return nil
	} else if next >= StateID(len(sm.stateTab)) {
		return ErrEvInvalidChange
	}
	sm.stateTab[sm.stateOn].onExit(sm.owner)
	sm.stateOn = next
	sm.stateTab[next].onEnter(sm.owner)
	return nil
}
