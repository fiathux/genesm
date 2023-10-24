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

// root scene
var scroot = sceneB{
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
			lineWidth: 3,
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
			color:     colorRGB{0.4, 0.8, 0.9},
			zdepth:    600,
		},
		trsDeg: vect{
			y: -0.004 * math.Pi,
			x: -0.003 * math.Pi,
		},
	},
}

// scene 1
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

// scene 2
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

// scene 2.1
var sc2x1 = sceneB{
	a: sceneA{
		o: object{
			p: []plan{
				plan{
					vect{0, 100, 0},
					vect{100, 42.26497308103743, 0},
					vect{100, -42.26497308103743, 0},
					vect{0, -100, 0},
					vect{-100, -42.26497308103743, 0},
					vect{-100, 42.26497308103743, 0},
				},
			},
			lineWidth: 2,
			color:     colorRGB{0.9, 0.9, 0.3},
			zdepth:    600,
		},
		trsDeg: vect{
			y: 0.002 * math.Pi,
		},
	},
	b: sceneA{
		o: object{
			p: []plan{
				plan{
					vect{0, 50, 0},
					vect{50, 21.132486540518716, 0},
					vect{50, -21.132486540518716, 0},
					vect{0, -50, 0},
					vect{-50, -21.132486540518716, 0},
					vect{-50, 21.132486540518716, 0},
				},
			},
			lineWidth: 3,
			color:     colorRGB{0.8, 0.8, 0.6},
			zdepth:    600,
		},
		trsDeg: vect{
			y: -0.006 * math.Pi,
			z: -0.003 * math.Pi,
		},
	},
}
