package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/syntax"
)

type catalog struct {
	hover map[string]string
	names []string
}

func newCatalog() *catalog {
	c := &catalog{hover: map[string]string{}}
	for _, k := range keywordList() {
		c.add(k, "BitShin BASIC keyword")
	}
	for _, k := range builtinList() {
		c.add(k, "BitShin BASIC builtin")
	}
	for k, cmd := range syntax.EntityMethods {
		c.add(k, "Entity method → "+cmd)
	}
	for k, v := range syntax.WeatherConstants {
		c.add(k, "Weather constant `"+v+"`")
	}
	for k := range syntax.NetConstants {
		c.add(k, "Network event constant")
	}
	for k := range syntax.KeyConstants {
		c.add(k, "Key scan-code constant")
	}
	return c
}

func (c *catalog) add(name, doc string) {
	key := lex.IdentKey(name)
	if key == "" {
		return
	}
	if _, ok := c.hover[key]; !ok {
		c.names = append(c.names, name)
	}
	if doc != "" {
		c.hover[key] = doc
	}
}

func (c *catalog) loadRoot(root string) {
	if root == "" {
		return
	}
	path := filepath.Join(root, "docs", "COMMANDS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	tick := regexp.MustCompile("`([A-Za-z][A-Za-z0-9$%#]*)`")
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches := tick.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		doc := strings.TrimSpace(tick.ReplaceAllString(line, ""))
		doc = strings.Trim(doc, "|- ")
		for _, m := range matches {
			name := m[1]
			if !unicode.IsLetter(rune(name[0])) {
				continue
			}
			if len(name) < 2 && strings.ToLower(name) != "if" {
				continue
			}
			c.add(name, strings.TrimSpace(doc))
		}
	}
}

func keywordList() []string {
	return []string{
		"If", "Then", "Else", "ElseIf", "EndIf", "While", "Wend", "EndWhile",
		"For", "To", "Step", "Next", "Repeat", "Until", "Function", "EndFunction",
		"End", "Return", "Dim", "ReDim", "Global", "Local", "Const", "Enum", "EndEnum",
		"Select", "Case", "Default", "EndSelect", "Data", "Read", "Restore",
		"And", "Or", "Not", "Xor", "Mod", "Include", "Import", "Exit",
		"Type", "EndType", "Struct", "EndStruct", "Method", "EndMethod",
		"Namespace", "EndNamespace", "Field", "New", "As", "True", "False",
		"Yes", "No", "Null", "Pi",
	}
}

func builtinList() []string {
	return []string{
		"Print", "Sin", "Cos", "Tan", "ASin", "ACos", "ATan", "ATan2", "Sqr", "Abs",
		"Int", "Floor", "Ceil", "Float", "Sgn", "Min", "Max", "Rnd", "Rand", "SeedRnd",
		"MilliSecs", "Len", "Left", "Right", "Mid", "Chr", "Asc", "Str", "Instr",
		"Lower", "Upper", "Trim", "Hex", "Clamp", "Lerp", "InvLerp", "SmoothStep",
		"EaseIn", "EaseOut", "Approach", "WrapAngle", "AngleDelta", "ApproachAngle",
		"Pow", "Dist", "Distance2D", "PointDistance", "Distance3D", "Length2D", "Length3D",
		"NormX", "NormY", "NormX3", "NormY3", "NormZ3", "DirX", "DirY", "DirZ",
		"MovePointX", "MovePointY", "MovePointZ", "PointYaw", "PointPitch",
		"Dot2D", "Dot3D", "CrossX", "CrossY", "CrossZ", "ReflectX", "ReflectY",
		"BounceX", "BounceY", "RotatedX", "RotatedY", "CreateList", "ListAdd",
		"ListGet", "ListSet", "ListCount", "ListRemove", "ArraySize",
		"BackBuffer", "FrontBuffer", "SetBuffer",
	}
}
