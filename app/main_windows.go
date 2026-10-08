//go:build windows

package main

import (
	"embed"
	"encoding/json"
	"image"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"
	"unsafe"
)

//go:embed sprites/*.png
var spriteFS embed.FS

const appName = "시몬스 데스크톱 컴패니언"

func init() { runtime.LockOSThread() }

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	pRegisterClassExW       = user32.NewProc("RegisterClassExW")
	pCreateWindowExW        = user32.NewProc("CreateWindowExW")
	pDefWindowProcW         = user32.NewProc("DefWindowProcW")
	pGetMessageW            = user32.NewProc("GetMessageW")
	pTranslateMessage       = user32.NewProc("TranslateMessage")
	pDispatchMessageW       = user32.NewProc("DispatchMessageW")
	pPostMessageW           = user32.NewProc("PostMessageW")
	pPostQuitMessage        = user32.NewProc("PostQuitMessage")
	pSetWindowsHookExW      = user32.NewProc("SetWindowsHookExW")
	pUnhookWindowsHookEx    = user32.NewProc("UnhookWindowsHookEx")
	pCallNextHookEx         = user32.NewProc("CallNextHookEx")
	pSetTimer               = user32.NewProc("SetTimer")
	pKillTimer              = user32.NewProc("KillTimer")
	pUpdateLayeredWindow    = user32.NewProc("UpdateLayeredWindow")
	pGetDC                  = user32.NewProc("GetDC")
	pReleaseDC              = user32.NewProc("ReleaseDC")
	pLoadImageW             = user32.NewProc("LoadImageW")
	pLoadCursorW            = user32.NewProc("LoadCursorW")
	pSetCursor              = user32.NewProc("SetCursor")
	pSetCapture             = user32.NewProc("SetCapture")
	pReleaseCapture         = user32.NewProc("ReleaseCapture")
	pCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	pAppendMenuW            = user32.NewProc("AppendMenuW")
	pTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	pDestroyMenu            = user32.NewProc("DestroyMenu")
	pSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	pGetCursorPos           = user32.NewProc("GetCursorPos")
	pGetWindowRect          = user32.NewProc("GetWindowRect")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pShowWindow             = user32.NewProc("ShowWindow")
	pDestroyWindow          = user32.NewProc("DestroyWindow")
	pMessageBoxW            = user32.NewProc("MessageBoxW")
	pSystemParametersInfoW  = user32.NewProc("SystemParametersInfoW")
	pGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	pRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pGdiFlush           = gdi32.NewProc("GdiFlush")
	pGetDeviceCaps      = gdi32.NewProc("GetDeviceCaps")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW     = kernel32.NewProc("CreateMutexW")

	pShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	pPlaySoundW       = winmm.NewProc("PlaySoundW")
)

const (
	wsPopup        = 0x80000000
	wsExLayered    = 0x00080000
	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080

	wmNull           = 0x0000
	wmDestroy        = 0x0002
	wmSetCursor      = 0x0020
	wmMouseActivate  = 0x0021
	wmKeyDown        = 0x0100
	wmKeyUp          = 0x0101
	wmSysKeyDown     = 0x0104
	wmSysKeyUp       = 0x0105
	wmTimer          = 0x0113
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmLButtonDblClk  = 0x0203
	wmRButtonDown    = 0x0204
	wmRButtonUp      = 0x0205
	wmCaptureChanged = 0x0215
	wmApp            = 0x8000
	wmAppKey         = wmApp + 1
	wmAppClick       = wmApp + 2
	wmAppTray        = wmApp + 3

	htClient       = 1
	maNoActivate   = 3
	whKeyboardLL   = 13
	whMouseLL      = 14
	ulwAlpha       = 2
	acSrcAlpha     = 1
	imageIcon      = 1
	nimAdd         = 0
	nimDelete      = 2
	nifMessage     = 1
	nifIcon        = 2
	nifTip         = 4
	mfString       = 0x0000
	mfSeparator    = 0x0800
	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100
	swpNoSize      = 0x0001
	swpNoZOrder    = 0x0004
	swpNoActivate  = 0x0010
	swShowNoAct    = 4
	smCxIcon       = 11
	smCyIcon       = 12
	smCxSmIcon     = 49
	smCySmIcon     = 50
	smXVirtual     = 76
	smYVirtual     = 77
	smCxVirtual    = 78
	smCyVirtual    = 79
	spiGetWorkArea = 0x0030
	logPixelsX     = 88
	idcArrow       = 32512
	idcSizeAll     = 32646
	idcHand        = 32649

	timerRevert = 1 // donut finished → back to the base pose
	timerSquash = 2
	timerDoze   = 3
	timerJelly  = 4
	timerFire   = 5

	sndAsync     = 0x0001
	sndNoDefault = 0x0002
	sndMemory    = 0x0004

	cmdResize = 1
	cmdQuit   = 2

	defaultWidth = 300.0 // logical px
	minWidth     = 140.0
	maxWidth     = 800.0
	dragSlop     = 5 // px of movement before a press becomes a drag
)

type dragKind int

const (
	dragNone    dragKind = iota
	dragPending          // button down, not moved yet → becomes a pat on release
	dragMove             // moving the companion around
	dragResize           // resize mode drag
)

type settings struct {
	X, Y   int
	HasPos bool
	Width  float64 // logical px (DPI independent)
}

type app struct {
	hwnd, hinst uintptr
	memDC       uintptr
	origBmp     uintptr
	srcW, srcH  int
	bmps        [spCount]uintptr
	pix         [spCount][]byte // DIB pixels of each pose at current size (GDI-owned memory)
	scratch     uintptr         // one reusable frame for squash/jelly effects
	scratchPix  []byte
	tempBmp     uintptr
	liveSrc     []byte // full-resolution pose used only while live-resizing
	jellyIdx    int
	crunch      []byte
	w, h        int
	cur         Sprite
	cfg         Config
	brain       *Brain
	st          settings
	dpi         float64
	kbHook      uintptr
	msHook      uintptr
	keyDown     [256]bool
	trayIcon    uintptr
	bigIcon     uintptr
	curHand     uintptr
	curSizeAll  uintptr
	taskbarMsg  uint32
	trayAdded   bool

	resizeMode bool
	menuOpen   bool
	drag       dragKind
	downPt     point
	downWin    rect
	centerX    float64
	centerY    float64
	d0         float64
	w0         int
	liveW      int
	liveH      int
}

var A app

var (
	cbWndProc = syscall.NewCallback(wndProc)
	cbKbProc  = syscall.NewCallback(kbProc)
	cbMsProc  = syscall.NewCallback(msProc)
)

func u16(s string) *uint16  { p, _ := syscall.UTF16PtrFromString(s); return p }
func ptr(p *uint16) uintptr { return uintptr(unsafe.Pointer(p)) }

func msgBox(text string) {
	pMessageBoxW.Call(0, ptr(u16(text)), ptr(u16(appName)), 0x40)
}

func cursorPos() point {
	var p point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

func (a *app) windowRect() rect {
	var r rect
	pGetWindowRect.Call(a.hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

// ---------- settings ----------

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "SimonsCompanion", "settings.json")
}

func loadSettings() settings {
	s := settings{Width: defaultWidth}
	if b, err := os.ReadFile(settingsPath()); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	if s.Width < minWidth || s.Width > maxWidth {
		s.Width = defaultWidth
	}
	return s
}

func (a *app) saveSettings() {
	p := settingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if b, err := json.MarshalIndent(a.st, "", "  "); err == nil {
		_ = os.WriteFile(p, b, 0o644)
	}
}

// ---------- graphics ----------

func getDPIScale() float64 {
	hdc, _, _ := pGetDC.Call(0)
	d, _, _ := pGetDeviceCaps.Call(hdc, logPixelsX)
	pReleaseDC.Call(0, hdc)
	if d == 0 {
		return 1
	}
	return float64(d) / 96
}

func makeDIB(pix []byte, w, h int) (uintptr, []byte) {
	var bi bitmapInfo
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	bi.Header.Width = int32(w)
	bi.Header.Height = -int32(h) // top-down
	bi.Header.Planes = 1
	bi.Header.BitCount = 32
	var bits unsafe.Pointer
	hbmp, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmp == 0 || bits == nil {
		return 0, nil
	}
	buf := unsafe.Slice((*byte)(bits), w*h*4)
	copy(buf, pix)
	return hbmp, buf
}

func decodeSprite(sp Sprite) *image.NRGBA {
	data, err := spriteFS.ReadFile("sprites/" + spriteFiles[sp] + ".png")
	if err != nil {
		return nil
	}
	img, err := decodeNRGBA(data)
	if err != nil {
		return nil
	}
	return img
}

func (a *app) heightFor(w int) int {
	return int(float64(w)*float64(a.srcH)/float64(a.srcW) + 0.5)
}

func (a *app) clampWidth(w int) int {
	lo, hi := int(minWidth*a.dpi), int(maxWidth*a.dpi)
	vw, _, _ := pGetSystemMetrics.Call(smCxVirtual)
	if v := int(int32(vw)); v > 0 && hi > v {
		hi = v
	}
	if w < lo {
		return lo
	}
	if w > hi {
		return hi
	}
	return w
}

func (a *app) freeBitmaps() {
	if a.origBmp != 0 {
		pSelectObject.Call(a.memDC, a.origBmp)
	}
	for i := range a.bmps {
		if a.bmps[i] != 0 {
			pDeleteObject.Call(a.bmps[i])
			a.bmps[i] = 0
		}
		a.pix[i] = nil
	}
	for _, b := range []*uintptr{&a.scratch, &a.tempBmp} {
		if *b != 0 {
			pDeleteObject.Call(*b)
			*b = 0
		}
	}
	a.scratchPix = nil
}

// buildBitmaps renders every pose once at the current size. Source images are decoded
// from the embedded PNGs on demand and dropped afterwards to keep memory low.
func (a *app) buildBitmaps() bool {
	a.freeBitmaps()
	a.w = a.clampWidth(int(a.st.Width*a.dpi + 0.5))
	a.h = a.heightFor(a.w)
	for i := Sprite(0); i < spCount; i++ {
		img := decodeSprite(i)
		if img == nil {
			return false
		}
		a.bmps[i], a.pix[i] = makeDIB(renderBGRA(img, a.w, a.h, 1), a.w, a.h)
		if a.bmps[i] == 0 {
			return false
		}
	}
	a.scratch, a.scratchPix = makeDIB(nil, a.w, a.h)
	debug.FreeOSMemory() // hand temporary render buffers back to Windows
	return a.scratch != 0
}

// presentWarped shows pose sp scaled around its bottom-center (squash / jelly).
func (a *app) presentWarped(sp Sprite, sx, sy float64) {
	if a.scratch == 0 || a.pix[sp] == nil {
		return
	}
	pGdiFlush.Call()
	warpInto(a.scratchPix, a.pix[sp], a.w, a.h, sx, sy)
	a.present(a.scratch, a.w, a.h, nil)
}

func (a *app) present(bmp uintptr, w, h int, pos *point) {
	old, _, _ := pSelectObject.Call(a.memDC, bmp)
	if a.origBmp == 0 {
		a.origBmp = old
	}
	sz := sizeT{int32(w), int32(h)}
	src := point{0, 0}
	blend := blendFunc{0, 0, 255, acSrcAlpha}
	pUpdateLayeredWindow.Call(a.hwnd, 0, uintptr(unsafe.Pointer(pos)), uintptr(unsafe.Pointer(&sz)), a.memDC,
		uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha)
}

func (a *app) show(sp Sprite, squash bool) {
	a.cur = sp
	if a.drag == dragResize {
		return // the live-resize preview owns the window right now
	}
	if squash {
		a.presentWarped(sp, 1, a.cfg.SquashScale)
		return
	}
	if bmp := a.bmps[sp]; bmp != 0 {
		a.present(bmp, a.w, a.h, nil)
	}
}

// onCharacter reports whether screen point (x,y) is on a visible pixel of the companion.
func (a *app) onCharacter(x, y int32) bool {
	r := a.windowRect()
	if x < r.L || x >= r.R || y < r.T || y >= r.B {
		return false
	}
	lx, ly := int(x-r.L), int(y-r.T)
	p := a.pix[a.cur]
	if lx >= a.w || ly >= a.h || len(p) < a.w*a.h*4 {
		return false
	}
	return p[(ly*a.w+lx)*4+3] > 24 // ignore the faint anti-aliased fringe
}

// ---------- hooks ----------

func kbProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) == 0 {
		k := (*kbdllHook)(unsafe.Pointer(lParam))
		vk := k.VkCode & 0xFF
		switch wParam {
		case wmKeyDown, wmSysKeyDown:
			if !A.keyDown[vk] { // ignore auto-repeat while held
				A.keyDown[vk] = true
				if !isModifier(vk) { // Shift/Ctrl/Alt/Win/한영/한자 alone don't count as 타
					pPostMessageW.Call(A.hwnd, wmAppKey, 0, 0)
				}
			}
		case wmKeyUp, wmSysKeyUp:
			A.keyDown[vk] = false
		}
	}
	r, _, _ := pCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

func msProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) == 0 && (wParam == wmLButtonDown || wParam == wmRButtonDown) {
		m := (*msllHook)(unsafe.Pointer(lParam))
		packed := uintptr(uint32(m.Pt.X)) | uintptr(uint32(m.Pt.Y))<<32
		pPostMessageW.Call(A.hwnd, wmAppClick, 0, packed)
	}
	r, _, _ := pCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

func isModifier(vk uint32) bool {
	switch vk {
	case 0x10, 0x11, 0x12, 0x14, 0x15, 0x19, 0x5B, 0x5C, 0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5:
		return true
	}
	return false
}

func (a *app) hook() {
	a.kbHook, _, _ = pSetWindowsHookExW.Call(whKeyboardLL, cbKbProc, a.hinst, 0)
	a.msHook, _, _ = pSetWindowsHookExW.Call(whMouseLL, cbMsProc, a.hinst, 0)
}

func (a *app) unhook() {
	if a.kbHook != 0 {
		pUnhookWindowsHookEx.Call(a.kbHook)
		a.kbHook = 0
	}
	if a.msHook != 0 {
		pUnhookWindowsHookEx.Call(a.msHook)
		a.msHook = 0
	}
	a.keyDown = [256]bool{}
}

// ---------- behaviour ----------

func (a *app) setTimer(id uintptr, d time.Duration) {
	pSetTimer.Call(a.hwnd, id, uintptr(d/time.Millisecond), 0)
}

func (a *app) armDoze() { a.setTimer(timerDoze, ms(a.cfg.DozeAfterMs)) }

// wake leaves dozeoff when any input arrives.
func (a *app) wake() {
	a.armDoze()
	if a.cur == SpDozeOff && !a.brain.Eating(time.Now()) {
		a.show(a.brain.Base(), false)
	}
}

func (a *app) bounce(sp Sprite) {
	pKillTimer.Call(a.hwnd, timerSquash)
	a.show(sp, true)
	a.setTimer(timerSquash, ms(a.cfg.SquashMs))
}

func (a *app) onKey() {
	a.armDoze()
	sp, ok := a.brain.Key(time.Now())
	if a.brain.OnFire() {
		a.setTimer(timerFire, ms(a.cfg.FireCheckMs)) // keep checking until the pace drops
	} else {
		pKillTimer.Call(a.hwnd, timerFire)
	}
	if ok {
		a.bounce(sp)
	}
}

func playWAV(w []byte) {
	if len(w) > 0 {
		pPlaySoundW.Call(uintptr(unsafe.Pointer(&w[0])), 0, sndMemory|sndAsync|sndNoDefault)
	}
}

// donut: click on the character → donut pose + crunchy "와삭바삭" + pudding wobble.
func (a *app) donut() {
	a.armDoze()
	sp := a.brain.Donut(time.Now())
	pKillTimer.Call(a.hwnd, timerSquash)
	playWAV(a.crunch)
	a.cur = sp
	a.jellyIdx = 0
	if a.drag != dragResize {
		sx, sy := jellyScale(0)
		a.presentWarped(SpDonut, sx, sy)
	}
	a.setTimer(timerJelly, ms(jellyFrameMs))
	a.setTimer(timerRevert, ms(a.cfg.DonutHoldMs))
}

func (a *app) jellyTick() {
	a.jellyIdx++
	if a.jellyIdx >= jellyFrames || a.cur != SpDonut {
		pKillTimer.Call(a.hwnd, timerJelly)
		a.show(a.cur, false)
		return
	}
	if a.drag != dragResize {
		sx, sy := jellyScale(float64(a.jellyIdx*jellyFrameMs) / 1000)
		a.presentWarped(SpDonut, sx, sy)
	}
}

// onGlobalClick: a click somewhere else on the screen only counts as "not idle".
func (a *app) onGlobalClick(packed uintptr) {
	x, y := int32(uint32(packed)), int32(uint32(packed>>32))
	if a.drag != dragNone || a.menuOpen || a.onCharacter(x, y) {
		return // handled by the companion's own mouse messages
	}
	a.wake()
}

func (a *app) onTimer(id uintptr) {
	if id == timerJelly {
		a.jellyTick()
		return
	}
	if id == timerFire {
		sp, changed := a.brain.Tick(time.Now())
		if !a.brain.OnFire() {
			pKillTimer.Call(a.hwnd, timerFire)
		}
		if changed && a.cur != SpDozeOff {
			a.show(sp, false)
		}
		return
	}
	pKillTimer.Call(a.hwnd, id)
	switch id {
	case timerSquash:
		a.show(a.cur, false)
	case timerRevert: // donut is over
		pKillTimer.Call(a.hwnd, timerJelly)
		a.show(a.brain.Base(), false)
	case timerDoze:
		if a.brain.Doze(time.Now()) {
			pKillTimer.Call(a.hwnd, timerFire)
			a.show(SpDozeOff, false)
		}
	}
}

// ---------- mouse on the companion ----------

func (a *app) onLButtonDown() {
	pSetCapture.Call(a.hwnd)
	a.downPt = cursorPos()
	a.downWin = a.windowRect()
	if a.resizeMode {
		a.drag = dragResize
		a.centerX = float64(a.downWin.L+a.downWin.R) / 2
		a.centerY = float64(a.downWin.T+a.downWin.B) / 2
		a.d0 = math.Max(20, math.Hypot(float64(a.downPt.X)-a.centerX, float64(a.downPt.Y)-a.centerY))
		a.w0, a.liveW, a.liveH = a.w, a.w, a.h
		if img := decodeSprite(a.cur); img != nil {
			a.liveSrc = renderBGRA(img, a.srcW, a.srcH, 1)
		}
		pSetCursor.Call(a.curSizeAll)
		return
	}
	a.drag = dragPending
}

func (a *app) onMouseMove() {
	p := cursorPos()
	switch a.drag {
	case dragPending:
		if abs32(p.X-a.downPt.X) > dragSlop || abs32(p.Y-a.downPt.Y) > dragSlop {
			a.drag = dragMove
		}
		fallthrough
	case dragMove:
		if a.drag == dragMove {
			x := a.downWin.L + p.X - a.downPt.X
			y := a.downWin.T + p.Y - a.downPt.Y
			pSetWindowPos.Call(a.hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
		}
	case dragResize:
		pSetCursor.Call(a.curSizeAll)
		d := math.Hypot(float64(p.X)-a.centerX, float64(p.Y)-a.centerY)
		nw := a.clampWidth(int(float64(a.w0)*d/a.d0 + 0.5))
		if nw != a.liveW {
			a.livePreview(nw)
		}
	}
}

// livePreview shows a quick nearest-neighbour scaled frame while dragging.
func (a *app) livePreview(nw int) {
	if a.liveSrc == nil {
		return
	}
	nh := a.heightFor(nw)
	bmp, _ := makeDIB(scaleNearest(a.liveSrc, a.srcW, a.srcH, nw, nh), nw, nh)
	if bmp == 0 {
		return
	}
	pos := point{int32(a.centerX - float64(nw)/2), int32(a.centerY - float64(nh)/2)}
	a.present(bmp, nw, nh, &pos)
	if a.tempBmp != 0 {
		pDeleteObject.Call(a.tempBmp) // no longer selected
	}
	a.tempBmp = bmp
	a.liveW, a.liveH = nw, nh
}

func (a *app) finishResize() {
	a.drag = dragNone
	a.resizeMode = false
	a.unhook() // rebuilding takes a moment; keep the hooks from timing out
	a.st.Width = float64(a.liveW) / a.dpi
	a.liveSrc = nil
	a.buildBitmaps() // also frees the temporary preview bitmap
	x := int(a.centerX - float64(a.w)/2)
	y := int(a.centerY - float64(a.h)/2)
	pos := point{int32(x), int32(y)}
	a.present(a.bmps[a.cur], a.w, a.h, &pos)
	a.hook()
	a.savePos()
}

func (a *app) onLButtonUp() {
	kind := a.drag
	if kind == dragResize {
		a.finishResize()
	} else {
		a.drag = dragNone
	}
	pReleaseCapture.Call()
	switch kind {
	case dragPending: // a click without moving = donut time
		a.donut()
	case dragMove:
		a.savePos()
	}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// ---------- window placement ----------

func (a *app) initialPos() (int, int) {
	vx, _, _ := pGetSystemMetrics.Call(smXVirtual)
	vy, _, _ := pGetSystemMetrics.Call(smYVirtual)
	vw, _, _ := pGetSystemMetrics.Call(smCxVirtual)
	vh, _, _ := pGetSystemMetrics.Call(smCyVirtual)
	x0, y0 := int(int32(vx)), int(int32(vy))
	if a.st.HasPos {
		cx, cy := a.st.X+a.w/2, a.st.Y+a.h/2
		if cx >= x0 && cx < x0+int(int32(vw)) && cy >= y0 && cy < y0+int(int32(vh)) {
			return a.st.X, a.st.Y
		}
	}
	var wa rect
	pSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&wa)), 0)
	return int(wa.R) - a.w - 24, int(wa.B) - a.h
}

func (a *app) savePos() {
	r := a.windowRect()
	a.st.X, a.st.Y, a.st.HasPos = int(r.L), int(r.T), true
	a.saveSettings()
}

// ---------- tray & menu ----------

func (a *app) addTray() {
	var nid notifyIconData
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	nid.UFlags = nifMessage | nifIcon | nifTip
	nid.UCallbackMessage = wmAppTray
	nid.HIcon = a.trayIcon
	tip, _ := syscall.UTF16FromString(appName)
	copy(nid.SzTip[:len(nid.SzTip)-1], tip)
	r, _, _ := pShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	a.trayAdded = r != 0
}

func (a *app) removeTray() {
	if !a.trayAdded {
		return
	}
	var nid notifyIconData
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = 1
	pShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	a.trayAdded = false
}

func (a *app) showMenu() {
	m, _, _ := pCreatePopupMenu.Call()
	pAppendMenuW.Call(m, mfString, cmdResize, ptr(u16("크기 조절")))
	pAppendMenuW.Call(m, mfSeparator, 0, 0)
	pAppendMenuW.Call(m, mfString, cmdQuit, ptr(u16("종료")))

	pt := cursorPos()
	a.menuOpen = true
	pSetForegroundWindow.Call(a.hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, tpmReturnCmd|tpmRightButton|tpmNoNotify,
		uintptr(int(pt.X)), uintptr(int(pt.Y)), 0, a.hwnd, 0)
	pPostMessageW.Call(a.hwnd, wmNull, 0, 0)
	pDestroyMenu.Call(m)
	a.menuOpen = false

	switch cmd {
	case cmdResize:
		a.resizeMode = true
		if a.onCharacter(cursorPos().X, cursorPos().Y) {
			pSetCursor.Call(a.curSizeAll)
		}
	case cmdQuit:
		pDestroyWindow.Call(a.hwnd)
	}
}

// ---------- window proc ----------

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case wmMouseActivate:
		return maNoActivate // never steal focus from the document being typed
	case wmSetCursor:
		if uint16(lParam) == htClient {
			if A.resizeMode {
				pSetCursor.Call(A.curSizeAll)
			} else {
				pSetCursor.Call(A.curHand)
			}
			return 1
		}
	case wmLButtonDown, wmLButtonDblClk:
		A.onLButtonDown()
		return 0
	case wmMouseMove:
		if A.drag != dragNone {
			A.onMouseMove()
		}
		return 0
	case wmLButtonUp:
		A.onLButtonUp()
		return 0
	case wmCaptureChanged:
		if A.drag == dragResize {
			A.finishResize()
		} else if A.drag == dragMove {
			A.savePos()
		}
		A.drag = dragNone
		return 0
	case wmRButtonUp:
		if A.drag == dragNone {
			if A.resizeMode {
				A.resizeMode = false // right-click cancels resize mode
				pSetCursor.Call(A.curHand)
			} else {
				A.showMenu()
			}
		}
		return 0
	case wmAppKey:
		A.onKey()
		return 0
	case wmAppClick:
		A.onGlobalClick(lParam)
		return 0
	case wmAppTray:
		switch uint32(lParam) {
		case wmLButtonUp, wmRButtonUp:
			A.showMenu()
		}
		return 0
	case wmTimer:
		A.onTimer(wParam)
		return 0
	case wmDestroy:
		pPlaySoundW.Call(0, 0, 0)
		A.unhook()
		A.removeTray()
		A.saveSettings()
		pPostQuitMessage.Call(0)
		return 0
	}
	if A.taskbarMsg != 0 && uint32(msg) == A.taskbarMsg {
		A.addTray() // explorer restarted
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

// ---------- main ----------

func (a *app) run() {
	a.cfg = DefaultConfig()
	a.crunch = makeCrunchWAV()
	a.brain = NewBrain(a.cfg)
	a.st = loadSettings()
	a.hinst, _, _ = pGetModuleHandleW.Call(0)
	a.dpi = getDPIScale()

	first := decodeSprite(SpStart)
	if first == nil {
		msgBox("이미지를 불러오지 못했어요.")
		return
	}
	a.srcW, a.srcH = first.Rect.Dx(), first.Rect.Dy()
	first = nil

	hdc, _, _ := pGetDC.Call(0)
	a.memDC, _, _ = pCreateCompatibleDC.Call(hdc)
	pReleaseDC.Call(0, hdc)
	if !a.buildBitmaps() {
		msgBox("이미지를 준비하지 못했어요.")
		return
	}

	cx, _, _ := pGetSystemMetrics.Call(smCxIcon)
	cy, _, _ := pGetSystemMetrics.Call(smCyIcon)
	a.bigIcon, _, _ = pLoadImageW.Call(a.hinst, 1, imageIcon, cx, cy, 0)
	sx, _, _ := pGetSystemMetrics.Call(smCxSmIcon)
	sy, _, _ := pGetSystemMetrics.Call(smCySmIcon)
	a.trayIcon, _, _ = pLoadImageW.Call(a.hinst, 1, imageIcon, sx, sy, 0)
	arrow, _, _ := pLoadCursorW.Call(0, idcArrow)
	a.curHand, _, _ = pLoadCursorW.Call(0, idcHand)
	a.curSizeAll, _, _ = pLoadCursorW.Call(0, idcSizeAll)

	cls := u16("SimonsDesktopCompanion")
	wc := wndClassEx{
		WndProc: cbWndProc, Instance: a.hinst, Icon: a.bigIcon, IconSm: a.trayIcon,
		Cursor: arrow, ClassName: cls,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, _ := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		msgBox("창을 만들지 못했어요.")
		return
	}

	x, y := a.initialPos()
	a.hwnd, _, _ = pCreateWindowExW.Call(wsExLayered|wsExToolWindow|wsExTopmost, ptr(cls), ptr(u16(appName)), wsPopup,
		uintptr(x), uintptr(y), uintptr(a.w), uintptr(a.h), 0, 0, a.hinst, 0)
	if a.hwnd == 0 {
		msgBox("창을 만들지 못했어요.")
		return
	}
	tm, _, _ := pRegisterWindowMessageW.Call(ptr(u16("TaskbarCreated")))
	a.taskbarMsg = uint32(tm)

	a.show(SpStart, false)
	pShowWindow.Call(a.hwnd, swShowNoAct)
	a.addTray()
	a.hook()
	a.armDoze()

	var m msgT
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	a.freeBitmaps()
	pDeleteDC.Call(a.memDC)
}

func main() {
	h, _, err := pCreateMutexW.Call(0, 0, ptr(u16("Simons.DesktopCompanion.SingleInstance")))
	if h != 0 && err == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		msgBox("이미 실행 중이에요! 작업 표시줄 오른쪽 트레이 아이콘을 확인해 주세요.")
		return
	}
	A.run()
	runtime.KeepAlive(h)
}
