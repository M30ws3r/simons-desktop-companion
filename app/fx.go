package main

import (
	"bytes"
	"encoding/binary"
	"math"
)

// ---------- "왁뿌볼" crunch sound ----------

// rng is a tiny deterministic generator so the crunch sounds the same every time.
type rng uint32

func (r *rng) f() float64 { // 0..1
	*r ^= *r << 13
	*r ^= *r >> 17
	*r ^= *r << 5
	return float64(*r) / 4294967296.0
}

// makeCrunchWAV synthesizes the crumble of a crispy puffed snack ball being bitten:
// a dull "bite" thump, a sharp crack, then a shower of tiny crackles that thin out,
// each crackle a burst of band-passed noise at a random pitch.
func makeCrunchWAV() []byte {
	const rate = 44100
	const dur = 0.55
	n := int(rate * dur)
	out := make([]float64, n)
	r := rng(0x5eed1234)

	// grain: band-passed noise burst (biquad band-pass, Q≈1.4) with a fast-attack / exp-decay envelope
	grain := func(at, length, freq, amp float64) {
		w0 := 2 * math.Pi * freq / rate
		alpha := math.Sin(w0) / (2 * 1.4)
		a0 := 1 + alpha
		b0, b2 := alpha/a0, -alpha/a0
		a1, a2 := -2*math.Cos(w0)/a0, (1-alpha)/a0
		var x1, x2, y1, y2 float64
		i0, m := int(at*rate), int(length*rate)
		for k := 0; k < m && i0+k < n; k++ {
			x := r.f()*2 - 1
			y := b0*x + b2*x2 - a1*y1 - a2*y2
			x2, x1 = x1, x
			y2, y1 = y1, y
			t := float64(k) / rate
			env := math.Min(1, t/0.0006) * math.Exp(-t/(length*0.35))
			out[i0+k] += y * env * amp
		}
	}

	// 1) bite: soft low thump
	for i := 0; i < int(0.05*rate); i++ {
		t := float64(i) / rate
		out[i] += 0.55 * math.Sin(2*math.Pi*(150-900*t)*t) * math.Exp(-t/0.014) * math.Min(1, t/0.002)
	}
	// 2) the shell cracks: two bright snaps
	grain(0.004, 0.030, 2600, 2.6)
	grain(0.020, 0.024, 3400, 2.0)
	// 3) crumbs: dense at first, thinning out
	t := 0.035
	for t < dur-0.05 {
		p := t / dur
		grain(t, 0.006+r.f()*0.014, 1400+r.f()*4200, (0.5+r.f()*1.1)*math.Exp(-p*3.2)*2.2)
		t += 0.004 + r.f()*0.018*(1+p*4)
	}
	// 4) a couple of late crumbs dropping
	grain(0.36, 0.012, 3900, 0.5)
	grain(0.44, 0.010, 2800, 0.35)

	peak := 0.0
	for _, v := range out {
		peak = math.Max(peak, math.Abs(v))
	}
	samples := make([]int16, n)
	for i, v := range out {
		v = v / peak * 0.8
		if tt := float64(i) / rate; tt > dur-0.02 {
			v *= (dur - tt) / 0.02
		}
		samples[i] = int16(math.Tanh(v*1.3) / math.Tanh(1.3) * 0.8 * 32767)
	}
	return pcmWAV(samples, rate)
}

func pcmWAV(samples []int16, rate int) []byte {
	var buf bytes.Buffer
	dataLen := uint32(len(samples) * 2)
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16)} {
		binary.Write(&buf, binary.LittleEndian, v)
	}
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataLen)
	binary.Write(&buf, binary.LittleEndian, samples)
	return buf.Bytes()
}

// ---------- jelly wobble ----------

const (
	jellyFrames  = 24
	jellyFrameMs = 30
)

// jellyScale returns (scaleX, scaleY) at time t seconds: a damped squash-and-stretch,
// starting squashed (like a pudding being pressed), roughly area-preserving.
func jellyScale(t float64) (float64, float64) {
	amp := 0.11 * math.Exp(-t/0.2)
	sy := 1 - amp*math.Cos(2*math.Pi*4.2*t)
	sx := 1 + (1-sy)*0.7
	return sx, sy
}

// warpInto draws src (premultiplied BGRA, w×h) scaled by (sx, sy) around the bottom-center
// into dst (same size) with bilinear filtering. Used for the squash and jelly effects so
// no extra frames have to be kept in memory.
func warpInto(dst, src []byte, w, h int, sx, sy float64) {
	cx, fh := float64(w)/2, float64(h)
	isx, isy := 1/sx, 1/sy
	// per-column source positions (fixed 8-bit fractions), computed once per frame
	xs := make([]int32, w)
	xf := make([]int32, w)
	for x := 0; x < w; x++ {
		f := (float64(x)+0.5-cx)*isx + cx - 0.5
		x0 := math.Floor(f)
		xs[x] = int32(x0)
		xf[x] = int32((f - x0) * 256)
	}
	stride := w * 4
	px := func(x, y int32) (int32, int32, int32, int32) {
		if x < 0 || y < 0 || int(x) >= w || int(y) >= h {
			return 0, 0, 0, 0
		}
		o := int(y)*stride + int(x)*4
		return int32(src[o]), int32(src[o+1]), int32(src[o+2]), int32(src[o+3])
	}
	for y := 0; y < h; y++ {
		f := (float64(y)+0.5-fh)*isy + fh - 0.5
		fy0 := math.Floor(f)
		y0 := int32(fy0)
		ty := int32((f - fy0) * 256)
		row := dst[y*stride : (y+1)*stride]
		if y0 < -1 || int(y0) >= h {
			clear(row)
			continue
		}
		interiorY := y0 >= 0 && int(y0)+1 < h
		for x := 0; x < w; x++ {
			x0, tx := xs[x], xf[x]
			d := x * 4
			if x0 < -1 || int(x0) >= w {
				row[d], row[d+1], row[d+2], row[d+3] = 0, 0, 0, 0
				continue
			}
			var a0, a1, a2, a3, b0, b1, b2, b3, c0, c1, c2, c3, e0, e1, e2, e3 int32
			if interiorY && x0 >= 0 && int(x0)+1 < w {
				o := int(y0)*stride + int(x0)*4
				p := src[o : o+8 : o+8]
				q := src[o+stride : o+stride+8 : o+stride+8]
				a0, a1, a2, a3 = int32(p[0]), int32(p[1]), int32(p[2]), int32(p[3])
				b0, b1, b2, b3 = int32(p[4]), int32(p[5]), int32(p[6]), int32(p[7])
				c0, c1, c2, c3 = int32(q[0]), int32(q[1]), int32(q[2]), int32(q[3])
				e0, e1, e2, e3 = int32(q[4]), int32(q[5]), int32(q[6]), int32(q[7])
			} else {
				a0, a1, a2, a3 = px(x0, y0)
				b0, b1, b2, b3 = px(x0+1, y0)
				c0, c1, c2, c3 = px(x0, y0+1)
				e0, e1, e2, e3 = px(x0+1, y0+1)
			}
			itx, ity := 256-tx, 256-ty
			row[d] = byte(((a0*itx+b0*tx)*ity + (c0*itx+e0*tx)*ty + 32768) >> 16)
			row[d+1] = byte(((a1*itx+b1*tx)*ity + (c1*itx+e1*tx)*ty + 32768) >> 16)
			row[d+2] = byte(((a2*itx+b2*tx)*ity + (c2*itx+e2*tx)*ty + 32768) >> 16)
			row[d+3] = byte(((a3*itx+b3*tx)*ity + (c3*itx+e3*tx)*ty + 32768) >> 16)
		}
	}
}
