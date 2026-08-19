// Package syntax holds shared BitShin BASIC QoL mappings (dot methods, vec/hex, constants).
package syntax

import (
	"strconv"
	"strings"

	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/value"
)

// EntityMethods maps a short method name to a command (handle is prepended).
// Only used when the call is written as recv.Name(args). Field access recv.Name stays a field.
var EntityMethods = map[string]string{
	"position":  "setposition",
	"scale":     "setscale",
	"rotate":    "setrotation",
	"rotation":  "setrotation",
	"color":     "entitycolor",
	"alpha":     "entityalpha",
	"move":      "moveentity",
	"translate": "translateentity",
	"turn":      "turnentity",
	"point":     "pointentity",
	"lookat":    "pointentity",
	"hide":      "hideentity",
	"show":      "showentity",
	"name":      "nameentity",
	"parent":    "entityparent",
	"texture":   "entitytexture",
	"shininess": "entityshininess",
	"specular":  "entityspecular",
	"follow":    "camerafollow",
}

// ResolveMethod returns the command for a dotted method name.
func ResolveMethod(name string) (string, bool) {
	cmd, ok := EntityMethods[lex.IdentKey(name)]
	return cmd, ok
}

var chainCommands map[string]bool

func init() {
	chainCommands = map[string]bool{}
	for _, cmd := range EntityMethods {
		chainCommands[cmd] = true
	}
	for _, cmd := range []string{
		"positionentity", "scaleentity", "rotateentity", "entitycolor", "entityalpha",
		"moveentity", "translateentity", "turnentity", "pointentity",
		"hideentity", "showentity", "nameentity", "entityparent",
		"entitytexture", "entityshininess", "entityspecular", "camerafollow",
	} {
		chainCommands[cmd] = true
	}
}

// ReturnsEntity is true for mutating entity commands that should yield the handle (chaining).
func ReturnsEntity(name string) bool {
	return chainCommands[lex.IdentKey(name)]
}

// WeatherNames are the string values of WEATHER_* constants (also accepted by SetWeather).
var WeatherNames = []string{"clear", "rain", "snow", "fog", "storm"}

// WeatherConstants maps WEATHER_* identifiers to the string SetWeather accepts.
var WeatherConstants = map[string]string{
	"weather_clear": "clear",
	"weather_rain":  "rain",
	"weather_snow":  "snow",
	"weather_fog":   "fog",
	"weather_storm": "storm",
}

// NetConstants are PollNetwork event kinds (ENet order: none, connect, disconnect, receive).
var NetConstants = map[string]int{
	"net_none":       0,
	"net_connect":    1,
	"net_disconnect": 2,
	"net_receive":    3,
	"net_recv":       3,
}

// WeatherMode normalizes a SetWeather argument (name, WEATHER_* string, or 0–4).
func WeatherMode(v value.Value) string {
	if v.Kind == value.KindStr {
		s := strings.ToLower(strings.TrimSpace(v.Str))
		if strings.HasPrefix(s, "weather_") {
			s = strings.TrimPrefix(s, "weather_")
		}
		if s != "" {
			return s
		}
	}
	if v.Kind == value.KindNum {
		i := v.Int()
		if i >= 0 && i < len(WeatherNames) {
			return WeatherNames[i]
		}
	}
	s := strings.ToLower(strings.TrimSpace(v.String()))
	return s
}

// KeyConstants are Blitz3D-style scan codes installed as globals (KEY_ESCAPE, KEY_W, …).
var KeyConstants = map[string]float64{
	"key_escape": 1, "key_esc": 1,
	"key_1": 2, "key_2": 3, "key_3": 4, "key_4": 5, "key_5": 6,
	"key_6": 7, "key_7": 8, "key_8": 9, "key_9": 10, "key_0": 11,
	"key_minus": 12, "key_equals": 13,
	"key_backspace": 14, "key_tab": 15,
	"key_q": 16, "key_w": 17, "key_e": 18, "key_r": 19, "key_t": 20,
	"key_y": 21, "key_u": 22, "key_i": 23, "key_o": 24, "key_p": 25,
	"key_lbracket": 26, "key_rbracket": 27,
	"key_enter": 28, "key_lcontrol": 29, "key_left_control": 29, "key_ctrl": 29, "key_control": 29, "key_lctrl": 29,
	"key_a": 30, "key_s": 31, "key_d": 32, "key_f": 33, "key_g": 34,
	"key_h": 35, "key_j": 36, "key_k": 37, "key_l": 38,
	"key_lshift": 42, "key_left_shift": 42, "key_shift": 42, "key_rshift": 54, "key_right_shift": 54,
	"key_z": 44, "key_x": 45, "key_c": 46, "key_v": 47, "key_b": 48,
	"key_n": 49, "key_m": 50,
	"key_alt": 56, "key_lalt": 56, "key_ralt": 184,
	"key_space": 57,
	"key_f1":    59, "key_f2": 60, "key_f3": 61, "key_f4": 62, "key_f5": 63,
	"key_f6": 64, "key_f7": 65, "key_f8": 66, "key_f9": 67, "key_f10": 68,
	"key_up": 200, "key_left": 203, "key_right": 205, "key_down": 208,
	"key_rcontrol": 157, "key_right_control": 157, "key_rctrl": 157,
}

// ParseHexInt parses FFECc8 / $FFECc8 into an integer (typically packed RGB).
func ParseHexInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return strconv.ParseInt(s, 16, 64)
}

// PackedRGB splits a 24-bit hex color into 0–255 channels.
func PackedRGB(n int) (r, g, b float64) {
	return float64((n >> 16) & 255), float64((n >> 8) & 255), float64(n & 255)
}

func isXYZCommand(name string) bool {
	switch lex.IdentKey(name) {
	case "setposition", "positionentity", "setcameraposition",
		"setscale", "scaleentity",
		"setrotation", "rotateentity", "setcamerarotation",
		"moveentity", "translateentity", "turnentity",
		"setweatherwind", "setwind":
		return true
	}
	return false
}

func isColorCommand(name string) bool {
	switch lex.IdentKey(name) {
	case "entitycolor", "setentitycolor", "setmaterialcolor",
		"lightcolor", "setlightcolor",
		"ambientlight", "setambientlight",
		"cameraclscolor", "setcameraclscolor":
		return true
	}
	return false
}

// ExpandCommandArgs unpacks a trailing vec3 or packed hex color for transform/color commands.
func ExpandCommandArgs(name string, args []value.Value) []value.Value {
	if isXYZCommand(name) {
		return expandHandleXYZ(args)
	}
	if isColorCommand(name) {
		return expandHandleColor(args)
	}
	return args
}

func expandHandleXYZ(args []value.Value) []value.Value {
	if len(args) < 1 {
		return args
	}
	// Position [x,y,z]  or  Position handle, [x,y,z]
	if len(args) == 1 {
		if x, y, z, ok := args[0].XYZ(); ok {
			return []value.Value{value.Num(x), value.Num(y), value.Num(z)}
		}
		return args
	}
	if len(args) == 2 {
		if x, y, z, ok := args[1].XYZ(); ok {
			return []value.Value{args[0], value.Num(x), value.Num(y), value.Num(z)}
		}
	}
	return args
}

func expandHandleColor(args []value.Value) []value.Value {
	if len(args) == 1 {
		return unpackColorArg(nil, args[0])
	}
	if len(args) == 2 {
		return unpackColorArg([]value.Value{args[0]}, args[1])
	}
	if len(args) == 3 {
		// r,g,b with no handle (AmbientLight / CameraClsColor)
		if _, _, _, ok := args[0].XYZ(); !ok && args[0].Kind != value.KindVec {
			return args
		}
	}
	return args
}

func unpackColorArg(prefix []value.Value, v value.Value) []value.Value {
	if x, y, z, ok := v.XYZ(); ok {
		return append(append([]value.Value{}, prefix...), value.Num(x), value.Num(y), value.Num(z))
	}
	if v.Kind == value.KindNum {
		r, g, b := PackedRGB(v.Int())
		return append(append([]value.Value{}, prefix...), value.Num(r), value.Num(g), value.Num(b))
	}
	if v.Kind == value.KindStr {
		if n, err := ParseHexInt(v.Str); err == nil {
			r, g, b := PackedRGB(int(n))
			return append(append([]value.Value{}, prefix...), value.Num(r), value.Num(g), value.Num(b))
		}
	}
	return append(append([]value.Value{}, prefix...), v)
}
