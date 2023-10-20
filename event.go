package genesm

// Event represent a event to change state on state matchine
type Event interface {
	Trigger() error
}

// EventX represent a event. which allow you add hook ahead the event trigger
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

// SetHook set a hook function that allow developer check contain data of each
// State. if an error is be returned, event will be canceled as well.
func (eb *eventBind[O, A, B]) SetHook(
	hook func(O, A, B) error,
) {
	eb.hook = hook
}

// Trigger use to trigger event
func (eb *eventBind[O, A, B]) Trigger() (rerr error) {
	eb.sm.transform(func(curID StateID) StateID {
		if curID != eb.a.ID() {
			if curID == eb.b.ID() {
				rerr = ErrEvAlreadyChanged
			} else {
				rerr = ErrEvUnexpectedState
			}
			return STIDInvalid
		}
		if eb.hook != nil {
			rerr = eb.hook(eb.sm.owner, eb.a.Get(), eb.b.Get())
		}
		if rerr != nil {
			return STIDInvalid
		}
		return eb.b.ID()
	})
	return
}
