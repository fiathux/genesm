package genesm

import "testing"

func TestFrameFunc(t *testing.T) {
	ff0 := FrameObserverFunc(func(o string, e FrameEvent, s StateID, k int, v int) {
		t.Log("FrameFunc0:", o, e, s, v)
	})
	ff0.Frame("A", FEvIdle, 10, 0, 99901)
}
