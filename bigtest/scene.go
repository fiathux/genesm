package main

import "math"

type sceneA struct {
	o      object
	trsDeg vect
}

type sceneB struct {
	a sceneA
	b sceneA
}

var sc0 = sceneB{
	a: sceneA{
		o: object{
			p: []plan{
				plan{
					vect{-100, 100, 100},
					vect{100, 100, 100},
					vect{100, -100, 100},
					vect{-100, -100, 100},
				},
				plan{
					vect{100, 100, 100},
					vect{100, 0, -100},
					vect{100, -100, 100},
				},
				plan{
					vect{-100, 100, 100},
					vect{100, 100, 100},
					vect{100, 0, -100},
					vect{-100, 0, -100},
				},
				plan{
					vect{-100, 100, 100},
					vect{-100, 0, -100},
					vect{-100, -100, 100},
				},
			},
			lineWidth: 2,
			color:     colorRGB{0.2, 0.5, 0.9},
			zdepth:    600,
		},
		trsDeg: vect{
			y: 0.002 * math.Pi,
			x: 0.002 * math.Pi,
			z: 0.001 * math.Pi,
		},
	},
	b: sceneA{
		o: object{
			p: []plan{
				plan{
					vect{-50, 50, 50},
					vect{50, 50, 50},
					vect{50, -50, 50},
					vect{-50, -50, 50},
				},
				plan{
					vect{50, 50, 50},
					vect{50, 0, -50},
					vect{50, -50, 50},
				},
				plan{
					vect{-50, 50, 50},
					vect{50, 50, 50},
					vect{50, 0, -50},
					vect{-50, 0, -50},
				},
				plan{
					vect{-50, 50, 50},
					vect{-50, 0, -50},
					vect{-50, -50, 50},
				},
			},
			lineWidth: 2,
			color:     colorRGB{0.2, 0.5, 0.9},
			zdepth:    600,
		},
		trsDeg: vect{
			y: -0.004 * math.Pi,
			x: -0.003 * math.Pi,
		},
	},
}

var sc1 = sceneA{
	o: object{
		p: []plan{
			plan{
				vect{-100, 100, 100},
				vect{100, 100, 100},
				vect{-100, -100, 100},
			},
			plan{
				vect{100, 100, 100},
				vect{100, -100, 100},
				vect{-100, -100, 100},
			},
			plan{
				vect{100, 100, 100},
				vect{100, 100, -100},
				vect{100, -100, -100},
				vect{100, -100, 100},
			},
			plan{
				vect{-100, 100, -100},
				vect{100, 100, -100},
				vect{100, -100, -100},
				vect{-100, -100, -100},
			},
			plan{
				vect{-100, 100, 100},
				vect{-100, 100, -100},
				vect{-100, -100, -100},
				vect{-100, -100, 100},
			},
		},
		lineWidth: 2,
		color:     colorRGB{0.7, 0.1, 0.8},
		zdepth:    600,
	},
	trsDeg: vect{
		y: 0.002 * math.Pi,
		x: 0.001 * math.Pi,
	},
}

var sc2 = sceneA{
	o: object{
		p: []plan{
			plan{
				vect{-100, 100, 100},
				vect{100, 100, 100},
				vect{0, -100, 100},
			},
			plan{
				vect{-100, 100, 100},
				vect{100, 100, 100},
				vect{0, 0, -100},
			},
			plan{
				vect{100, 100, 100},
				vect{0, -100, 100},
				vect{0, 0, -100},
			},
		},
		lineWidth: 2,
		color:     colorRGB{0.2, 0.7, 0.3},
		zdepth:    600,
	},
	trsDeg: vect{
		y: -0.002 * math.Pi,
		z: 0.001 * math.Pi,
	},
}
