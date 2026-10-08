package main

import "time"

// Sprite identifies one character pose.
type Sprite int

const (
	SpStart Sprite = iota
	SpOnAir
	SpYawning
	SpTutor
	SpOnFire
	SpDonut
	SpDozeOff
	spCount
)

var spriteFiles = [spCount]string{"start", "onair", "yawning", "tutor", "onfire", "donut", "dozeoff"}

// typingCycle is the order poses take turns in while typing.
var typingCycle = []Sprite{SpOnAir, SpYawning, SpTutor, SpOnFire}

// Config holds every timing value (same meaning as assets/simons/config.json).
type Config struct {
	PoseMinMs    int     // a typing pose stays at least this long before the next one may appear
	FireCPM      int     // 타/분 at or above which "onfire" locks in
	FireWindowMs int     // speed is measured over this sliding window (= how long the pace must be kept up)
	FireCheckMs  int     // how often the speed is re-checked while on fire
	DozeAfterMs  int     // no input for this long → dozeoff
	DonutHoldMs  int     // donut pose duration after a click on the character
	SquashMs     int     // per-keystroke bounce
	SquashScale  float64 // per-keystroke bounce height
}

func DefaultConfig() Config {
	return Config{
		PoseMinMs: 700, FireCPM: 500, FireWindowMs: 3000, FireCheckMs: 200,
		DozeAfterMs: 15000, DonutHoldMs: 1600, SquashMs: 60, SquashScale: 0.97,
	}
}

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

// Brain is the platform-independent state machine.
type Brain struct {
	cfg        Config
	presses    []time.Time
	cycleIdx   int // index into typingCycle of the pose currently shown (−1 = none yet)
	poseSince  time.Time
	onFire     bool
	donutUntil time.Time
	base       Sprite // what to go back to after the donut / when waking up
}

func NewBrain(c Config) *Brain { return &Brain{cfg: c, cycleIdx: -1, base: SpStart} }

// Base is the pose shown when nothing special is happening.
func (b *Brain) Base() Sprite { return b.base }

// OnFire reports whether the fast-typing lock is active.
func (b *Brain) OnFire() bool { return b.onFire }

// Eating reports whether the donut pose is still on screen.
func (b *Brain) Eating(now time.Time) bool { return now.Before(b.donutUntil) }

// CPM returns the current typing speed in 타/분 over the sliding window.
func (b *Brain) CPM(now time.Time) int {
	cut := now.Add(-ms(b.cfg.FireWindowMs))
	kept := b.presses[:0]
	for _, t := range b.presses {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	b.presses = kept
	return len(kept) * 60000 / b.cfg.FireWindowMs
}

func (b *Brain) next(now time.Time) Sprite {
	b.cycleIdx = (b.cycleIdx + 1) % len(typingCycle)
	b.poseSince = now
	b.base = typingCycle[b.cycleIdx]
	return b.base
}

// Key handles one key press. It always returns the pose to show (with a little bounce);
// show=false means "keep the donut on screen" (the press still counts towards speed).
func (b *Brain) Key(now time.Time) (sp Sprite, show bool) {
	b.presses = append(b.presses, now)
	fast := b.CPM(now) >= b.cfg.FireCPM

	switch {
	case fast:
		if !b.onFire {
			b.onFire = true
			b.poseSince = now
		}
		b.base = SpOnFire
	case b.onFire: // speed just dropped below the line
		b.onFire = false
		b.cycleIdx = -1
		b.next(now)
	case b.cycleIdx < 0 || b.base == SpStart || now.Sub(b.poseSince) >= ms(b.cfg.PoseMinMs):
		b.next(now)
	}
	if b.Eating(now) {
		return b.base, false
	}
	return b.base, true
}

// Tick re-checks the speed while on fire (typing may simply have stopped).
// changed=true means the fire ended and sp should now be shown.
func (b *Brain) Tick(now time.Time) (sp Sprite, changed bool) {
	if !b.onFire || b.CPM(now) >= b.cfg.FireCPM {
		return b.base, false
	}
	b.onFire = false
	b.cycleIdx = -1
	sp = b.next(now)
	return sp, !b.Eating(now)
}

// Donut handles a left click on the character itself.
func (b *Brain) Donut(now time.Time) Sprite {
	b.donutUntil = now.Add(ms(b.cfg.DonutHoldMs))
	return SpDonut
}

// Doze is called after DozeAfterMs without input. It reports whether to show dozeoff.
func (b *Brain) Doze(now time.Time) bool {
	if b.Eating(now) {
		return false
	}
	b.onFire = false
	b.presses = b.presses[:0]
	return true
}
