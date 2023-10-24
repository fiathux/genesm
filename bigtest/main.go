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
	// scGoto3 switch scene from scene 2 to scene 2x1
	scGoto3 genesm.Event
	// scReturn return to root scene
	scReturn genesm.Event
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

// frameDrawB create a frame observer with sceneB update support
func frameDrawB(
	upd func(newsc sceneB),
) func(*Window, genesm.FrameEvent, genesm.StateID, int64, sceneB) {
	return func(
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
		upd(sc)
	}
}

// initMgr use state machine to initialize scene manager
func initMgr(ctx context.Context, wd *Window) <-chan struct{} {
	// create state machine
	sm := genesm.NewStateMachine(wd)

	// bind scene as a state
	scbind0 := genesm.RegState(sm, scroot)
	scbind1 := genesm.RegState(sm, sc1)
	scbind2 := genesm.RegState(sm, sc2)
	scbind3 := genesm.RegState(sm, sc2x1)
	sceneMap[scbind0.ID()] = "root"
	sceneMap[scbind1.ID()] = "scene one"
	sceneMap[scbind2.ID()] = "scene two"
	sceneMap[scbind3.ID()] = "scene two/sub"

	// bind event
	//      + <<<<<<<<<<<<<<<<<<< +
	//      |                     |
	//   >> + >> 0 + >> 1 >>>>>>> +
	//             |              |
	//             + >> 2 >> 3 >> +
	scGoto1 = genesm.RegEvent(sm, scbind0, scbind1)
	scGoto2 = genesm.RegEvent(sm, scbind0, scbind2)
	scGoto3 = genesm.RegEvent(sm, scbind2, scbind3)
	sc1Return := genesm.RegEvent(sm, scbind1, scbind0)
	sc3Return := genesm.RegEvent(sm, scbind3, scbind0)
	scReturn = genesm.GroupEvent(sc1Return, sc3Return)

	// create event handler
	obHndA := genesm.ObsEventFuncs(
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
	obHndB := genesm.ObsEventFuncs(
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

	// create controller for event observer
	ctrEv := genesm.NewObsController(0, 0)

	// bind event observer
	scbind0.AddObserver(genesm.CreateEventObserver(ctrEv, obHndB, nil))
	scbind1.AddObserver(genesm.CreateEventObserver(ctrEv, obHndA, nil))
	scbind2.AddObserver(genesm.CreateEventObserver(ctrEv, obHndA, nil))
	scbind3.AddObserver(genesm.CreateEventObserver(ctrEv, obHndB, nil))

	// create time-based observer
	// time-based observer will draw graphic and update status for a actived scene

	// create controller for time-based observer
	ctrFm := genesm.NewObsController(0, 0)

	// create ticker
	ticker, _ := genesm.CreateObsFrameTicker(frameRate)

	// create frame handler
	sc0fr := genesm.ObsFrameFunc(frameDrawB(func(s sceneB) {
		scbind0.Set(s)
	}))
	sc1fr := genesm.ObsFrameFunc(frameDrawA(func(s sceneA) {
		scbind1.Set(s)
	}))
	sc2fr := genesm.ObsFrameFunc(frameDrawA(func(s sceneA) {
		scbind2.Set(s)
	}))
	sc3fr := genesm.ObsFrameFunc(frameDrawB(func(s sceneB) {
		scbind3.Set(s)
	}))

	// bind time-based observer
	scbind0.AddObserver(genesm.CreateFrameObserver(ctrFm, ticker, sc0fr, nil))
	scbind1.AddObserver(genesm.CreateFrameObserver(ctrFm, ticker, sc1fr, nil))
	scbind2.AddObserver(genesm.CreateFrameObserver(ctrFm, ticker, sc2fr, nil))
	scbind3.AddObserver(genesm.CreateFrameObserver(ctrFm, ticker, sc3fr, nil))

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
						case sdl.K_3:
							scGoto3.Trigger()
						case sdl.K_RETURN:
							scReturn.Trigger()
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
