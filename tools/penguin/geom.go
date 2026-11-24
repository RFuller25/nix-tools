package main

import "math"

type vec3 struct{ x, y, z float64 }

func (a vec3) add(b vec3) vec3      { return vec3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a vec3) sub(b vec3) vec3      { return vec3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec3) scale(k float64) vec3 { return vec3{a.x * k, a.y * k, a.z * k} }
func (a vec3) dot(b vec3) float64   { return a.x*b.x + a.y*b.y + a.z*b.z }
func (a vec3) norm() vec3 {
	l := math.Sqrt(a.dot(a))
	if l == 0 {
		return a
	}
	return a.scale(1 / l)
}

// mat3 is a row-major 3x3 matrix.
type mat3 [9]float64

func ident() mat3 { return mat3{1, 0, 0, 0, 1, 0, 0, 0, 1} }

func rotX(a float64) mat3 {
	c, s := math.Cos(a), math.Sin(a)
	return mat3{1, 0, 0, 0, c, -s, 0, s, c}
}

func rotY(a float64) mat3 {
	c, s := math.Cos(a), math.Sin(a)
	return mat3{c, 0, s, 0, 1, 0, -s, 0, c}
}

func rotZ(a float64) mat3 {
	c, s := math.Cos(a), math.Sin(a)
	return mat3{c, -s, 0, s, c, 0, 0, 0, 1}
}

func (m mat3) mul(n mat3) mat3 {
	var o mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			o[i*3+j] = m[i*3]*n[j] + m[i*3+1]*n[3+j] + m[i*3+2]*n[6+j]
		}
	}
	return o
}

func (m mat3) vec(v vec3) vec3 {
	return vec3{
		m[0]*v.x + m[1]*v.y + m[2]*v.z,
		m[3]*v.x + m[4]*v.y + m[5]*v.z,
		m[6]*v.x + m[7]*v.y + m[8]*v.z,
	}
}

func (m mat3) t() mat3 {
	return mat3{m[0], m[3], m[6], m[1], m[4], m[7], m[2], m[5], m[8]}
}

// matID selects how a part is coloured.
type matID int

const (
	matBody matID = iota
	matNeck
	matHead
	matBeak
	matFlipper
	matFoot
	matTail
	matChickBody
	matChickHead
	matChickBeak
)

// part is one ellipsoid of a model, already placed in camera-aligned space
// (x right, y up, z toward the viewer) with the model's feet at the origin.
type part struct {
	c   vec3
	rot mat3
	r   vec3
	inv mat3 // world offset -> unit-sphere space
	a   float64
	mat matID
	cb  vec3 // centre in the body frame, for height-based colouring
}

func newPart(c vec3, rot mat3, r vec3, m matID, cb vec3) part {
	rt := rot.t()
	var inv mat3
	rr := [3]float64{r.x, r.y, r.z}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			inv[i*3+j] = rt[i*3+j] / rr[i]
		}
	}
	dz := vec3{-inv[2], -inv[5], -inv[8]}
	return part{c: c, rot: rot, r: r, inv: inv, a: dz.dot(dz), mat: m, cb: cb}
}

const camZ = 6.0

// hit tests the orthographic ray through (ox, oy) travelling toward -z. It
// returns the ray parameter and the unit-sphere hit point.
func (p *part) hit(ox, oy float64) (float64, vec3, bool) {
	off := vec3{ox - p.c.x, oy - p.c.y, camZ - p.c.z}
	i := &p.inv
	o := vec3{
		i[0]*off.x + i[1]*off.y + i[2]*off.z,
		i[3]*off.x + i[4]*off.y + i[5]*off.z,
		i[6]*off.x + i[7]*off.y + i[8]*off.z,
	}
	d := vec3{-i[2], -i[5], -i[8]}
	b := o.dot(d)
	c := o.dot(o) - 1
	disc := b*b - p.a*c
	if disc < 0 {
		return 0, vec3{}, false
	}
	t := (-b - math.Sqrt(disc)) / p.a
	return t, o.add(d.scale(t)), true
}
