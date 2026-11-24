package main

import "math"

func hash32(x, y int32, seed uint32) uint32 {
	h := uint32(x)*0x27d4eb2d ^ uint32(y)*0x165667b1 ^ seed*0x9e3779b9
	h ^= h >> 15
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

// rnd is a deterministic hash in [0,1).
func rnd(x, y int, seed uint32) float64 {
	return float64(hash32(int32(x), int32(y), seed)>>8) / float64(1<<24)
}

// vnoise is smooth value noise in [0,1).
func vnoise(x, y float64, seed uint32) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := x-ix, y-iy
	fx = fx * fx * (3 - 2*fx)
	fy = fy * fy * (3 - 2*fy)
	x0, y0 := int(ix), int(iy)
	a := rnd(x0, y0, seed)
	b := rnd(x0+1, y0, seed)
	c := rnd(x0, y0+1, seed)
	d := rnd(x0+1, y0+1, seed)
	return lerp(lerp(a, b, fx), lerp(c, d, fx), fy)
}

// fbm is fractal value noise normalised to roughly [0,1].
func fbm(x, y float64, oct int, seed uint32) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for i := 0; i < oct; i++ {
		sum += amp * vnoise(x, y, seed+uint32(i)*131)
		norm += amp
		amp *= 0.5
		x, y = x*2.03+17.1, y*2.03+9.7
	}
	return sum / norm
}

// noise1 is smooth one-dimensional value noise in [0,1).
func noise1(x float64, seed uint32) float64 {
	ix := math.Floor(x)
	f := x - ix
	f = f * f * (3 - 2*f)
	return lerp(rnd(int(ix), 0, seed), rnd(int(ix)+1, 0, seed), f)
}

func fbm1(x float64, oct int, seed uint32) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for i := 0; i < oct; i++ {
		sum += amp * noise1(x, seed+uint32(i)*77)
		norm += amp
		amp *= 0.5
		x *= 2.1
	}
	return sum / norm
}
