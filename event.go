package genesm

import "errors"

// Event errors
var (
	ErrEvEmptyGroup   = errors.New("no member in event group")
	ErrEvGroupFailure = errors.New("all events are failure")
)

// Event represent a event to change state on state matchine
type Event interface {
	Trigger() error
}

// EventX represent a event that similar as Event. but which allow you add hook
// ahead the event trigger
type EventX[O any, A any, B any] interface {
	Event
	SetHook(hook func(O, A, B) error)
}

// eventBind implement a Event. it will regist to StateMachine. then provide
// methods to hook or trigger state change.
type eventBind[O any, A any, B any] struct {
	sm *StateMachine[O]
	a  StateBinder[O, A]
	b  StateBinder[O, B]

	hook func(O, A, B) error
}

// eventGroup group several Event objects
type eventGroup []Event

// RegEvent regist an event rule to state machine
//
// A event rule is path to change state from one (a) to next one (b).
//
// it return Event interface to let developer to trigger it.
func RegEvent[O any, A any, B any](
	sm *StateMachine[O], a StateBinder[O, A], b StateBinder[O, B],
) EventX[O, A, B] {
	if a.Parent() != sm {
		panic("state (a) is not be owned under specified StateMachine")
	}
	if b.Parent() != sm {
		panic("state (b) is not be owned under specified StateMachine")
	}
	return &eventBind[O, A, B]{
		sm: sm,
		a:  a,
		b:  b,
	}
}

// GroupEvent group several Event objects as a new Event. trigger this group is
// equal to try in-order trigger each event until got a succeed
func GroupEvent(evs ...Event) Event {
	return eventGroup(evs)
}

// SetHook set a hook function that allow developer check contain data of each
// State. if an error is be returned, event will be canceled as well.
func (eb *eventBind[O, A, B]) SetHook(
	hook func(O, A, B) error,
) {
	eb.hook = hook
}

// Trigger trigger the event
func (eb *eventBind[O, A, B]) Trigger() (rerr error) {
	eb.sm.transform(func(curID StateID) StateID {
		if curID != eb.a.ID() {
			if curID == eb.b.ID() {
				rerr = ErrEvAlreadyChanged
			} else {
				rerr = ErrEvUnexpectedState
			}
			return STIDInvalid()
		}
		if eb.hook != nil {
			rerr = eb.hook(eb.sm.owner, eb.a.Get(), eb.b.Get())
		}
		if rerr != nil {
			return STIDInvalid()
		}
		return eb.b.ID()
	})
	return
}

// Trigger try in-order trigger each event
func (eg eventGroup) Trigger() (rerr error) {
	if len(eg) == 0 {
		return ErrEvEmptyGroup
	}
	for _, ev := range eg {
		if err := ev.Trigger(); err == nil {
			return nil
		}
	}
	return ErrEvGroupFailure
}
