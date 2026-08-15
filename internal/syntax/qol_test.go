package syntax

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestResolveMethod(t *testing.T) {
	cmd, ok := ResolveMethod("Position")
	if !ok || cmd != "setposition" {
		t.Fatalf("Position → %q %v", cmd, ok)
	}
	if _, ok := ResolveMethod("x"); ok {
		t.Fatal("field name x must not be a method")
	}
}

func TestExpandVecAndHex(t *testing.T) {
	xyz := ExpandCommandArgs("setposition", []value.Value{value.Num(3), value.Vec3(0, 2.2, -8)})
	if len(xyz) != 4 || xyz[1].Number() != 0 || xyz[2].Number() != 2.2 || xyz[3].Number() != -8 {
		t.Fatalf("vec expand %v", xyz)
	}
	classic := ExpandCommandArgs("positionentity", []value.Value{value.Num(3), value.Num(1), value.Num(2), value.Num(3)})
	if len(classic) != 4 || classic[3].Number() != 3 {
		t.Fatalf("classic xyz %v", classic)
	}
	col := ExpandCommandArgs("entitycolor", []value.Value{value.Num(3), value.Num(0xFFECc8)})
	if len(col) != 4 || col[1].Number() != 0xFF || col[2].Number() != 0xEC || col[3].Number() != 0xC8 {
		t.Fatalf("hex color %v", col)
	}
	hexStr := ExpandCommandArgs("entitycolor", []value.Value{value.Num(3), value.Str("FF0000")})
	if len(hexStr) != 4 || hexStr[1].Number() != 255 || hexStr[3].Number() != 0 {
		t.Fatalf("hex string %v", hexStr)
	}
}

func TestWeatherMode(t *testing.T) {
	if WeatherMode(value.Str("snow")) != "snow" {
		t.Fatal(WeatherMode(value.Str("snow")))
	}
	if WeatherMode(value.Str("WEATHER_SNOW")) != "snow" {
		t.Fatalf("const string → %q", WeatherMode(value.Str("WEATHER_SNOW")))
	}
	if WeatherMode(value.Num(2)) != "snow" {
		t.Fatalf("index 2 → %q", WeatherMode(value.Num(2)))
	}
	if WeatherConstants["weather_snow"] != "snow" {
		t.Fatal(WeatherConstants)
	}
	if NetConstants["net_none"] != 0 || NetConstants["net_connect"] != 1 || NetConstants["net_disconnect"] != 2 || NetConstants["net_receive"] != 3 {
		t.Fatal(NetConstants)
	}
}
