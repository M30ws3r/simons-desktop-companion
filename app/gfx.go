package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"
)

func decodeNRGBA(data []byte) (*image.NRGBA, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if n, ok := img.(*image.NRGBA); ok {
		return n, nil
	}
	b := img.Bounds()
	n := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(n, n.Bounds(), img, b.Min, draw.Src)
	return n, nil
}

// plane holds premultiplied RGBA as float32, 4 values per pixel.
type plane struct {
	w, h int
	p    []float32
}

func toPremul(src *image.NRGBA) plane {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := plane{w, h, make([]float32, w*h*4)}
	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride : y*src.Stride+w*4]
		for x := 0; x < w; x++ {
			a := float32(row[x*4+3]) / 255
			o := (y*w + x) * 4
			out.p[o] = float32(row[x*4]) * a
			out.p[o+1] = float32(row[x*4+1]) * a
			out.p[o+2] = float32(row[x*4+2]) * a
			out.p[o+3] = float32(row[x*4+3])
		}
	}
	return out
}

type tap struct {
	i int
	w float32
}

// weights builds resampling taps: area-average when shrinking, linear when enlarging.
func weights(srcN, dstN int) [][]tap {
	res := make([][]tap, dstN)
	scale := float64(srcN) / float64(dstN)
	for i := 0; i < dstN; i++ {
		if scale >= 1 {
			a, b := float64(i)*scale, float64(i+1)*scale
			for j := int(math.Floor(a)); j < int(math.Ceil(b)) && j < srcN; j++ {
				ov := math.Min(b, float64(j+1)) - math.Max(a, float64(j))
				if ov > 0 {
					res[i] = append(res[i], tap{j, float32(ov / scale)})
				}
			}
		} else {
			c := (float64(i)+0.5)*scale - 0.5
			j0 := int(math.Floor(c))
			t := float32(c - float64(j0))
			clamp := func(j int) int {
				if j < 0 {
					return 0
				}
				if j >= srcN {
					return srcN - 1
				}
				return j
			}
			res[i] = []tap{{clamp(j0), 1 - t}, {clamp(j0 + 1), t}}
		}
	}
	return res
}

func resizeH(in plane, nw int) plane {
	out := plane{nw, in.h, make([]float32, nw*in.h*4)}
	ws := weights(in.w, nw)
	for y := 0; y < in.h; y++ {
		for x, taps := range ws {
			var r, g, b, a float32
			for _, t := range taps {
				o := (y*in.w + t.i) * 4
				r += in.p[o] * t.w
				g += in.p[o+1] * t.w
				b += in.p[o+2] * t.w
				a += in.p[o+3] * t.w
			}
			o := (y*nw + x) * 4
			out.p[o], out.p[o+1], out.p[o+2], out.p[o+3] = r, g, b, a
		}
	}
	return out
}

func resizeV(in plane, nh int) plane {
	out := plane{in.w, nh, make([]float32, in.w*nh*4)}
	ws := weights(in.h, nh)
	for y, taps := range ws {
		for x := 0; x < in.w; x++ {
			var r, g, b, a float32
			for _, t := range taps {
				o := (t.i*in.w + x) * 4
				r += in.p[o] * t.w
				g += in.p[o+1] * t.w
				b += in.p[o+2] * t.w
				a += in.p[o+3] * t.w
			}
			o := (y*in.w + x) * 4
			out.p[o], out.p[o+1], out.p[o+2], out.p[o+3] = r, g, b, a
		}
	}
	return out
}

func clamp8(v float32) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}

// renderBGRA scales src to w×round(h*scaleY), bottom-aligned inside a w×h canvas,
// and returns premultiplied BGRA bytes (top-down) ready for a 32-bit DIB.
func renderBGRA(src *image.NRGBA, w, h int, scaleY float64) []byte {
	sh := int(math.Round(float64(h) * scaleY))
	if sh < 1 {
		sh = 1
	}
	pl := resizeV(resizeH(toPremul(src), w), sh)
	out := make([]byte, w*h*4)
	off := h - sh
	for y := 0; y < sh; y++ {
		for x := 0; x < w; x++ {
			s := (y*w + x) * 4
			d := ((y+off)*w + x) * 4
			a := clamp8(pl.p[s+3])
			r, g, b := clamp8(pl.p[s]), clamp8(pl.p[s+1]), clamp8(pl.p[s+2])
			if r > a {
				r = a
			}
			if g > a {
				g = a
			}
			if b > a {
				b = a
			}
			out[d], out[d+1], out[d+2], out[d+3] = b, g, r, a
		}
	}
	return out
}

// scaleNearest is a fast preview scaler used while the user is live-resizing.
func scaleNearest(src []byte, sw, sh, dw, dh int) []byte {
	out := make([]byte, dw*dh*4)
	for y := 0; y < dh; y++ {
		sy := y * sh / dh
		for x := 0; x < dw; x++ {
			sx := x * sw / dw
			copy(out[(y*dw+x)*4:(y*dw+x)*4+4], src[(sy*sw+sx)*4:(sy*sw+sx)*4+4])
		}
	}
	return out
}
