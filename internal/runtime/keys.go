package runtime

import "github.com/g3n/engine/window"

// Blitz3D / DirectInput scan codes used by KeyDown / KeyHit.
const (
	KeyEscape    = 1
	Key1         = 2
	Key2         = 3
	Key3         = 4
	Key4         = 5
	Key5         = 6
	Key6         = 7
	Key7         = 8
	Key8         = 9
	Key9         = 10
	Key0         = 11
	KeyMinus     = 12
	KeyEquals    = 13
	KeyBackspace = 14
	KeyTab       = 15
	KeyQ         = 16
	KeyW         = 17
	KeyE         = 18
	KeyR         = 19
	KeyT         = 20
	KeyY         = 21
	KeyU         = 22
	KeyI         = 23
	KeyO         = 24
	KeyP         = 25
	KeyLBracket  = 26
	KeyRBracket  = 27
	KeyEnter     = 28
	KeyLControl  = 29
	KeyA         = 30
	KeyS         = 31
	KeyD         = 32
	KeyF         = 33
	KeyG         = 34
	KeyH         = 35
	KeyJ         = 36
	KeyK         = 37
	KeyL         = 38
	KeyLShift    = 42
	KeyZ         = 44
	KeyX         = 45
	KeyC         = 46
	KeyV         = 47
	KeyB         = 48
	KeyN         = 49
	KeyM         = 50
	KeySpace     = 57
	KeyF1        = 59
	KeyF2        = 60
	KeyF3        = 61
	KeyF4        = 62
	KeyF5        = 63
	KeyF6        = 64
	KeyF7        = 65
	KeyF8        = 66
	KeyF9        = 67
	KeyF10       = 68
	KeyUp        = 200
	KeyLeft      = 203
	KeyRight     = 205
	KeyDown      = 208
	KeyRShift    = 54
	KeyRControl  = 157
)

var dikToKey = map[int]window.Key{
	KeyEscape:    window.KeyEscape,
	Key1:         window.Key1,
	Key2:         window.Key2,
	Key3:         window.Key3,
	Key4:         window.Key4,
	Key5:         window.Key5,
	Key6:         window.Key6,
	Key7:         window.Key7,
	Key8:         window.Key8,
	Key9:         window.Key9,
	Key0:         window.Key0,
	KeyMinus:     window.KeyMinus,
	KeyEquals:    window.KeyEqual,
	KeyBackspace: window.KeyBackspace,
	KeyTab:       window.KeyTab,
	KeyQ:         window.KeyQ,
	KeyW:         window.KeyW,
	KeyE:         window.KeyE,
	KeyR:         window.KeyR,
	KeyT:         window.KeyT,
	KeyY:         window.KeyY,
	KeyU:         window.KeyU,
	KeyI:         window.KeyI,
	KeyO:         window.KeyO,
	KeyP:         window.KeyP,
	KeyLBracket:  window.KeyLeftBracket,
	KeyRBracket:  window.KeyRightBracket,
	KeyEnter:     window.KeyEnter,
	KeyLControl:  window.KeyLeftControl,
	KeyA:         window.KeyA,
	KeyS:         window.KeyS,
	KeyD:         window.KeyD,
	KeyF:         window.KeyF,
	KeyG:         window.KeyG,
	KeyH:         window.KeyH,
	KeyJ:         window.KeyJ,
	KeyK:         window.KeyK,
	KeyL:         window.KeyL,
	KeyLShift:    window.KeyLeftShift,
	KeyZ:         window.KeyZ,
	KeyX:         window.KeyX,
	KeyC:         window.KeyC,
	KeyV:         window.KeyV,
	KeyB:         window.KeyB,
	KeyN:         window.KeyN,
	KeyM:         window.KeyM,
	KeySpace:     window.KeySpace,
	KeyF1:        window.KeyF1,
	KeyF2:        window.KeyF2,
	KeyF3:        window.KeyF3,
	KeyF4:        window.KeyF4,
	KeyF5:        window.KeyF5,
	KeyF6:        window.KeyF6,
	KeyF7:        window.KeyF7,
	KeyF8:        window.KeyF8,
	KeyF9:        window.KeyF9,
	KeyF10:       window.KeyF10,
	KeyUp:        window.KeyUp,
	KeyLeft:      window.KeyLeft,
	KeyRight:     window.KeyRight,
	KeyDown:      window.KeyDown,
	KeyRShift:    window.KeyRightShift,
	KeyRControl:  window.KeyRightControl,
}

var namedKeys = map[string]int{
	"key_escape": KeyEscape, "key_esc": KeyEscape,
	"key_1": Key1, "key_2": Key2, "key_3": Key3, "key_4": Key4, "key_5": Key5,
	"key_6": Key6, "key_7": Key7, "key_8": Key8, "key_9": Key9, "key_0": Key0,
	"key_q": KeyQ, "key_w": KeyW, "key_e": KeyE, "key_r": KeyR, "key_t": KeyT,
	"key_y": KeyY, "key_u": KeyU, "key_i": KeyI, "key_o": KeyO, "key_p": KeyP,
	"key_a": KeyA, "key_s": KeyS, "key_d": KeyD, "key_f": KeyF, "key_g": KeyG,
	"key_h": KeyH, "key_j": KeyJ, "key_k": KeyK, "key_l": KeyL,
	"key_z": KeyZ, "key_x": KeyX, "key_c": KeyC, "key_v": KeyV, "key_b": KeyB,
	"key_n": KeyN, "key_m": KeyM,
	"key_minus": KeyMinus, "key_equals": KeyEquals,
	"key_space": KeySpace, "key_enter": KeyEnter, "key_tab": KeyTab,
	"key_lshift": KeyLShift, "key_rshift": KeyRShift,
	"key_lcontrol": KeyLControl, "key_rcontrol": KeyRControl,
	"key_up": KeyUp, "key_down": KeyDown, "key_left": KeyLeft, "key_right": KeyRight,
}
