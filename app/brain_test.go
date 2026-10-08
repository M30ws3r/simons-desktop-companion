package main

import (
	"image"
	"image/color"
	"testing"
	"time"
	"unsafe"
)

var t0 = time.Unix(1000, 0)

func at(msv int) time.Time { return t0.Add(time.Duration(msv) * time.Millisecond) }

// type n keys evenly spaced, starting at startMs; returns the poses shown and the end time
func typeKeys(b *Brain, startMs, n, gapMs int) ([]Sprite, int) {
	var got []Sprite
	t := startMs
	for i := 0; i < n; i++ {
		sp, _ := b.Key(at(t))
		got = append(got, sp)
		t += gapMs
	}
	return got, t
}

func TestSlowTypingCyclesPoses(t *testing.T) {
	b := NewBrain(DefaultConfig())
	if b.Base() != SpStart {
		t.Fatal("should start with the start pose")
	}
	// 200 타/분 (300ms apart): first key changes the pose at once, then every 700ms+
	got, _ := typeKeys(b, 0, 20, 300)
	var seq []Sprite
	for i, sp := range got {
		if i == 0 || sp != got[i-1] {
			seq = append(seq, sp)
		}
	}
	want := []Sprite{SpOnAir, SpYawning, SpTutor, SpOnFire, SpOnAir, SpYawning, SpTutor}
	for i := range want {
		if i >= len(seq) || seq[i] != want[i] {
			t.Fatalf("pose order %v, want prefix %v", seq, want)
		}
	}
	if b.OnFire() {
		t.Fatal("200 타/분 must not lock on fire")
	}
}

func TestFireLocksWhileFastAndReleasesWhenSlow(t *testing.T) {
	b := NewBrain(DefaultConfig())
	// 600 타/분 = one key every 100ms, for 6 seconds
	got, end := typeKeys(b, 0, 60, 100)
	// lock must kick in once 3s of keys at 600/min ≥ 500 (≈ 25 keys)
	for i := 30; i < 60; i++ {
		if got[i] != SpOnFire {
			t.Fatalf("key %d at 600타/분 showed %v, want onfire (all %v)", i, got[i], got)
		}
	}
	if !b.OnFire() {
		t.Fatal("should be on fire")
	}
	// keep going at 400 타/분 (150ms): the 3s window drains below 500 and fire ends
	got2, _ := typeKeys(b, end, 40, 150)
	if b.OnFire() || got2[len(got2)-1] == SpOnFire && got2[len(got2)-2] == SpOnFire {
		t.Fatalf("fire should end at 400 타/분: %v", got2)
	}
}

func TestFireEndsWhenTypingStops(t *testing.T) {
	b := NewBrain(DefaultConfig())
	_, end := typeKeys(b, 0, 50, 100)
	if !b.OnFire() {
		t.Fatal("should be on fire")
	}
	if _, changed := b.Tick(at(end)); changed {
		t.Fatal("still fast right after the last key")
	}
	sp, changed := b.Tick(at(end + 800))
	if !changed || sp == SpOnFire || b.OnFire() {
		t.Fatalf("tick after stopping should drop fire: %v %v", sp, changed)
	}
}

func TestShortBurstIsNotFire(t *testing.T) {
	b := NewBrain(DefaultConfig())
	// a 1-second burst at 900 타/분 then nothing: average over 3s is only ~300
	typeKeys(b, 0, 15, 66)
	if b.OnFire() {
		t.Fatal("a 1s burst should not count as sustained 500타")
	}
}

func TestDonutAndDoze(t *testing.T) {
	b := NewBrain(DefaultConfig())
	typeKeys(b, 0, 3, 300)
	if sp := b.Donut(at(1000)); sp != SpDonut {
		t.Fatal("click should show the donut")
	}
	if _, show := b.Key(at(1500)); show {
		t.Fatal("typing during the donut keeps the donut on screen")
	}
	if b.Doze(at(1600)) {
		t.Fatal("no dozing while eating")
	}
	if _, show := b.Key(at(3000)); !show {
		t.Fatal("typing after the donut should work")
	}
	if !b.Doze(at(20000)) {
		t.Fatal("should doze off")
	}
}

func TestStructSizes(t *testing.T) {
	checks := map[string][2]uintptr{
		"MSG": {unsafe.Sizeof(msgT{}), 48}, "WNDCLASSEXW": {unsafe.Sizeof(wndClassEx{}), 80},
		"NOTIFYICONDATAW": {unsafe.Sizeof(notifyIconData{}), 976}, "KBDLLHOOKSTRUCT": {unsafe.Sizeof(kbdllHook{}), 24},
		"MSLLHOOKSTRUCT": {unsafe.Sizeof(msllHook{}), 32}, "BITMAPINFOHEADER": {unsafe.Sizeof(bitmapInfoHeader{}), 40},
		"BLENDFUNCTION": {unsafe.Sizeof(blendFunc{}), 4},
	}
	for n, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s size %d want %d", n, c[0], c[1])
		}
	}
}

func TestRender(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			src.SetNRGBA(x, y, color.NRGBA{200, 100, 50, 128})
		}
	}
	out := renderBGRA(src, 40, 30, 0.97)
	// bottom row fully covered, premultiplied: R = 200*128/255 ≈ 100
	d := (29*40 + 20) * 4
	if out[d+3] < 126 || out[d+3] > 130 || out[d+2] < 98 || out[d+2] > 102 || out[d] < 23 || out[d] > 27 {
		t.Fatalf("bad pixel %v", out[d:d+4])
	}
	if out[3] != 0 { // top row empty because of squash
		t.Fatalf("top row should be transparent, alpha=%d", out[3])
	}
}

func TestFX(t *testing.T) {
	for _, w := range [][]byte{makeCrunchWAV()} {
		if string(w[:4]) != "RIFF" || string(w[8:12]) != "WAVE" || len(w) < 10000 {
			t.Fatal("bad wav")
		}
	}
	sx, sy := jellyScale(0)
	if sy >= 1 || sx <= 1 {
		t.Fatalf("jelly should start squashed: %v %v", sx, sy)
	}
	if _, sy := jellyScale(float64(jellyFrames*jellyFrameMs) / 1000); sy < 0.97 || sy > 1.03 {
		t.Fatalf("jelly should settle, sy=%v", sy)
	}
	src := make([]byte, 25*20*4)
	for i := range src {
		src[i] = 255
	}
	dst := make([]byte, len(src))
	warpInto(dst, src, 25, 20, 1.1, 1.1)
	if dst[(10*25+12)*4+3] != 255 {
		t.Fatal("stretched frame should fill the box")
	}
	warpInto(dst, src, 25, 20, 1, 0.8)
	if dst[3] != 0 || dst[(19*25+12)*4+3] != 255 {
		t.Fatal("squashed frame: top should be empty, bottom filled")
	}
}

func BenchmarkWarpLarge(b *testing.B) {
	w, h := 1200, 893
	src := make([]byte, w*h*4)
	dst := make([]byte, w*h*4)
	for i := 0; i < b.N; i++ {
		warpInto(dst, src, w, h, 1.05, 0.92)
	}
}

func BenchmarkWarpDefault(b *testing.B) {
	w, h := 450, 335
	src := make([]byte, w*h*4)
	dst := make([]byte, w*h*4)
	for i := 0; i < b.N; i++ {
		warpInto(dst, src, w, h, 1.05, 0.92)
	}
}
