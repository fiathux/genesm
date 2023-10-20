package main

import (
	"github.com/ungerik/go-cairo"
	"github.com/veandco/go-sdl2/sdl"
)

type Window struct {
	wd  *sdl.Window
	sur *sdl.Surface
	cs  *cairo.Surface
	w   int
	h   int
}

// createWindow create window with cairo
func createWindow(width, height int) (*Window, error) {
	wd, err := sdl.CreateWindow("gensm Bigtest", sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED, int32(width), int32(height), sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, err
	}
	sur, err := wd.GetSurface()
	if err != nil {
		wd.Destroy()
		return nil, err
	}
	cs := cairo.NewSurfaceFromData(sur.Data(),
		cairo.FORMAT_ARGB32, int(sur.W), int(sur.H),
		int(sur.Pitch))
	return &Window{
		wd:  wd,
		sur: sur,
		cs:  cs,
		w:   width,
		h:   height,
	}, nil
}

// Destroy destroy window
func (w *Window) Destroy() {
	w.wd.Destroy()
}

// Clean clean canvas
func (w *Window) Clean() {
	w.sur.FillRect(nil, 0)
}

// Draw do drawing for cairo as a transaction
func (w *Window) Draw(f func(cs *cairo.Surface)) {
	w.cs.Save()
	defer func() {
		w.cs.Restore()
		w.wd.UpdateSurface()
	}()
	f(w.cs)
}
