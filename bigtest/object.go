package main

import (
	"math"

	"github.com/ungerik/go-cairo"
)

type vect struct {
	x float64
	y float64
	z float64
}

type colorRGB struct {
	r float64
	g float64
	b float64
}

type plan []vect

// object represent a 3d object
type object struct {
	p         []plan
	lineWidth float64
	color     colorRGB
	zdepth    float64
}

// Rotate do rotate for a vector
func (v vect) Rotate(deg vect) vect {
	r := v
	if deg.x != 0 {
		r = vect{
			x: r.x,
			y: r.y*math.Cos(deg.x) + r.z*math.Sin(deg.x),
			z: r.z*math.Cos(deg.x) - r.y*math.Sin(deg.x),
		}
	}
	if deg.y != 0 {
		r = vect{
			x: r.x*math.Cos(deg.y) - r.z*math.Sin(deg.y),
			y: r.y,
			z: r.x*math.Sin(deg.y) + r.z*math.Cos(deg.y),
		}
	}
	if deg.z != 0 {
		r = vect{
			x: r.x*math.Cos(deg.z) + r.y*math.Sin(deg.z),
			y: r.y*math.Cos(deg.z) - r.x*math.Sin(deg.z),
			z: r.z,
		}
	}
	return r
}

func (p plan) Rotate(deg vect) {
	for i, v := range p {
		p[i] = v.Rotate(deg)
	}
}

func (o *object) Rotate(deg vect) {
	for _, p := range o.p {
		p.Rotate(deg)
	}
}

func (o *object) Draw(cs *cairo.Surface, cx float64, cy float64) {
	vecDraw := func(v vect, move bool) {
		zd := float64(1.0)
		if o.zdepth != 0 {
			zd = math.Pow(2, v.z/o.zdepth)
		}
		if move {
			cs.MoveTo(v.x*zd+cx, cy-v.y*zd)
		} else {
			cs.LineTo(v.x*zd+cx, cy-v.y*zd)
		}
	}
	for _, p := range o.p {
		lastv := p[0]
		for _, v := range p[1:] {
			cs.SetSourceRGB(o.color.r, o.color.g, o.color.b)
			cs.SetLineWidth(o.lineWidth)
			vecDraw(lastv, true)
			vecDraw(v, false)
			cs.Stroke()
			lastv = v
		}
		cs.SetSourceRGB(o.color.r, o.color.g, o.color.b)
		cs.SetLineWidth(o.lineWidth)
		vecDraw(lastv, true)
		vecDraw(p[0], false)
		cs.Stroke()
	}
}
