package genesm

type Event[O any, A any, B any] interface {
	SetHook(hook func(O, A, B) error)
	Trigger() error
}

type eventBind[O any, A any, B any] struct {
	sm *StateMachine[O]
	a  StateBind[O, A]
	b  StateBind[O, B]

	hook func(O, A, B) error
}

func RegEvent[O any, A any, B any](
	sm *StateMachine[O], a StateBind[O, A], b StateBind[O, B],
) Event[O, A, B] {
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

func (eb *eventBind[O, A, B]) SetHook(
	hook func(O, A, B) error,
) {
	eb.hook = hook
}

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
