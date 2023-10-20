package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fiathux/genesm"
	"github.com/ungerik/go-cairo"
	"github.com/veandco/go-sdl2/sdl"
)

const frameRate = float32(60)

// event triggers
var (
	// scGoto1 switch scene from root scene to scene 1
	scGoto1 genesm.Event
	// scGoto2 switch scene from root scene to scene 2
	scGoto2 genesm.Event
	// sc1Return switch scene 1 to root scene
	sc1Return genesm.Event
	// sc2Return switch scene 2 to root scene
	sc2Return genesm.Event
)

var sceneMap = map[genesm.StateID]string{}

const (
	WWidth   = 800
	WHeight  = 600
	WXCenter = 400
	WYCenter = 300
)

// frameDrawA create a frame observer with sceneA update support
func frameDrawA(
	upd func(newsc sceneA),
) func(*Window, genesm.FrameEvent, genesm.StateID, int64, sceneA) {
	return func(
		wd *Window, ev genesm.FrameEvent, stateID genesm.StateID,
		skipped int64, sc sceneA,
	) {
		// draw scene
		wd.Clean()
		wd.Draw(func(cs *cairo.Surface) {
			sc.o.Draw(cs, WXCenter, WYCenter)
		})
		// update scene
		sc.o.Rotate(sc.trsDeg)
		upd(sc)
	}
}

// initMgr use state machine to initialize scene manager
func initMgr(ctx context.Context, wd *Window) <-chan struct{} {
	// create state machine
	sm := genesm.NewStateMachine(wd)

	// bind scene as a state
	scbind0 := genesm.RegState(sm, sc0)
	scbind1 := genesm.RegState(sm, sc1)
	scbind2 := genesm.RegState(sm, sc2)
	sceneMap[scbind0.ID()] = "root"
	sceneMap[scbind1.ID()] = "scene one"
	sceneMap[scbind2.ID()] = "scene two"

	// bind event
	scGoto1 = genesm.RegEvent(sm, scbind0, scbind1)
	scGoto2 = genesm.RegEvent(sm, scbind0, scbind2)
	sc1Return = genesm.RegEvent(sm, scbind1, scbind0)
	sc2Return = genesm.RegEvent(sm, scbind2, scbind0)

	// create event handler
	obHndA := genesm.EventObserverFuncs(
		func(wd *Window, id genesm.StateID, val sceneA) {
			fmt.Println("Type A Enter:", sceneMap[id])
		},
		func(wd *Window, id genesm.StateID, val sceneA) {
			fmt.Println("Type A Exit:", sceneMap[id])
		},
		func(wd *Window, id genesm.StateID, val sceneA) {
			fmt.Println("Type A SM Pick:", sceneMap[id])
		}, nil,
	)
	obHndB := genesm.EventObserverFuncs(
		func(wd *Window, id genesm.StateID, val sceneB) {
			fmt.Println("Type B Enter:", sceneMap[id])
		},
		func(wd *Window, id genesm.StateID, val sceneB) {
			fmt.Println("Type B Exit:", sceneMap[id])
		},
		func(wd *Window, id genesm.StateID, val sceneB) {
			fmt.Println("Type B SM Pick:", sceneMap[id])
		}, nil,
	)

	// bind event observer
	scbind0.AddObserver(genesm.CreateEventObserver(obHndB, 0, 0, nil))
	scbind1.AddObserver(genesm.CreateEventObserver(obHndA, 0, 0, nil))
	scbind2.AddObserver(genesm.CreateEventObserver(obHndA, 0, 0, nil))

	// create time-based observer
	// time-based observer will draw graphic and update status for a actived scene

	// create ticker
	ticker, _ := genesm.CreateFrameObTicker(frameRate)

	// create observer handler
	sc0fr := genesm.FrameObserverFunc(func(
		wd *Window, ev genesm.FrameEvent, stateID genesm.StateID,
		skipped int64, sc sceneB,
	) {
		wd.Clean()
		wd.Draw(func(cs *cairo.Surface) {
			sc.a.o.Draw(cs, WXCenter, WYCenter)
			sc.b.o.Draw(cs, WXCenter, WYCenter)
		})
		// update scene
		sc.a.o.Rotate(sc.a.trsDeg)
		sc.b.o.Rotate(sc.b.trsDeg)
		scbind0.Set(sc)
	})
	sc1fr := genesm.FrameObserverFunc(frameDrawA(func(s sceneA) {
		scbind1.Set(s)
	}))
	sc2fr := genesm.FrameObserverFunc(frameDrawA(func(s sceneA) {
		scbind2.Set(s)
	}))

	// bind time-based observer
	scbind0.AddObserver(genesm.CreateFrameObserver(ticker, sc0fr, 0, 0, nil))
	scbind1.AddObserver(genesm.CreateFrameObserver(ticker, sc1fr, 0, 0, nil))
	scbind2.AddObserver(genesm.CreateFrameObserver(ticker, sc2fr, 0, 0, nil))

	ret := make(chan struct{})

	go func() {
		<-ctx.Done()
		ticker.Stop()
		time.Sleep(100 * time.Millisecond)
		close(ret)
	}()
	return ret
}

func main() {
	// init canvas
	sdl.Init(sdl.INIT_EVERYTHING)
	wd, err := createWindow(WWidth, WHeight)
	if err != nil {
		panic(err)
	}
	defer wd.Destroy()

	ctx, cancel := context.WithCancel(context.Background())
	initMgr(ctx, wd)

	wdsig := make(chan struct{})

	go func() {
		for {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch ev := event.(type) {
				case *sdl.KeyboardEvent:
					if ev.State == sdl.PRESSED && ev.Repeat == 0 {
						switch ev.Keysym.Sym {
						case sdl.K_1:
							scGoto1.Trigger()
						case sdl.K_2:
							scGoto2.Trigger()
						case sdl.K_RETURN:
							sc1Return.Trigger()
							sc2Return.Trigger()
						}
					}
				case *sdl.QuitEvent:
					cancel()
					close(wdsig)
				}
			}
		}
	}()

	<-wdsig
}
