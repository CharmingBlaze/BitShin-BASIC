package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"moonbasic/internal/lex"
	"moonbasic/internal/runtime"
	"moonbasic/internal/syntax"
)

type catalog struct {
	hover map[string]string
	names []string
}

func newCatalog() *catalog {
	c := &catalog{hover: map[string]string{}}
	for _, k := range keywordList() {
		c.add(k, "moonBASIC keyword")
	}
	for _, k := range builtinList() {
		c.add(k, "moonBASIC builtin")
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
	for k, v := range syntax.PhysicsConstants {
		c.add(k, fmt.Sprintf("Physics constant %g", v))
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
	defer c.seedCommands()
	if root == "" {
		return
	}
	path := filepath.Join(root, "docs", "COMMANDS.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	tick := regexp.MustCompile("`([^`]+)`")
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		c.ingestDocLine(line, tick)
	}
}

// ingestDocLine records the command at the start of each backtick span.
// Docs write `CameraRange cam, near, far`, so the name is only the first word.
func (c *catalog) ingestDocLine(line string, tick *regexp.Regexp) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || tick == nil {
		return
	}
	matches := tick.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return
	}
	doc := strings.TrimSpace(tick.ReplaceAllString(line, ""))
	doc = strings.Trim(doc, "|- ")
	for _, m := range matches {
		name := commandToken(m[1])
		if name == "" {
			continue
		}
		c.add(name, strings.TrimSpace(doc))
	}
}

func commandToken(span string) string {
	span = strings.TrimSpace(span)
	if span == "" || !unicode.IsLetter(rune(span[0])) {
		return ""
	}
	i := 0
	for i < len(span) {
		r := rune(span[i])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '$' || r == '%' || r == '#' {
			i++
			continue
		}
		break
	}
	name := span[:i]
	if len(name) < 2 && strings.ToLower(name) != "if" {
		return ""
	}
	return name
}

// seedCommands marks every runtime command known, including aliases that
// the docs mention only with their arguments inside the backticks.
func (c *catalog) seedCommands() {
	for _, name := range runtime.CommandNames() {
		key := lex.IdentKey(name)
		if key == "" {
			continue
		}
		if _, ok := c.hover[key]; ok {
			continue
		}
		c.add(name, "moonBASIC command")
	}
}

func keywordList() []string {
	return []string{
		"If", "Then", "Else", "ElseIf", "EndIf", "While", "Wend", "EndWhile",
		"For", "To", "Step", "Next", "Repeat", "Until", "Function", "EndFunction",
		"End", "Return", "Dim", "ReDim", "Global", "Local", "Const", "Enum", "EndEnum",
		"Strict", "Try", "Catch", "EndTry",
		"Select", "Case", "Default", "EndSelect", "Data", "Read", "Restore",
		"And", "Or", "Not", "Xor", "Mod", "Include", "Import", "Exit",
		"Type", "EndType", "Struct", "EndStruct", "Method", "EndMethod",
		"Namespace", "EndNamespace", "Field", "New", "As", "In", "Each", "True", "False",
		"Yes", "No", "Null", "Pi",
	}
}

func builtinList() []string {
	return []string{
		"Print", "Sin", "Cos", "Tan", "ASin", "ACos", "ATan", "ATan2", "Sqr", "Abs",
		"Int", "Floor", "Ceil", "Float", "Sgn", "Min", "Max", "Rnd", "Rand", "SeedRnd",
		"MilliSecs", "Len", "Left", "Right", "Mid", "Chr", "Asc", "Str", "Instr", "Replace", "String",
		"Lower", "Upper", "Trim", "Hex", "Clamp", "Lerp", "InvLerp", "SmoothStep",
		"EaseIn", "EaseOut", "Approach", "WrapAngle", "AngleDelta", "ApproachAngle",
		"MoveWish", "Accelerate", "TurnToward", "Land", "Material",
		"Pow", "Dist", "Distance2D", "PointDistance", "Distance3D", "Length2D", "Length3D",
		"NormX", "NormY", "NormX3", "NormY3", "NormZ3", "DirX", "DirY", "DirZ",
		"MovePointX", "MovePointY", "MovePointZ", "PointYaw", "PointPitch",
		"Dot2D", "Dot3D", "CrossX", "CrossY", "CrossZ", "ReflectX", "ReflectY",
		"BounceX", "BounceY", "RotatedX", "RotatedY", "CreateList", "ListAdd",
		"ListGet", "ListSet", "ListCount", "ListRemove", "ArraySize",
		"CreateMap", "MapSet", "MapGet", "MapHas", "MapDelete", "MapCount", "Copy", "Callback",
		"CreateStateMachine", "AddState", "GoState", "StateName", "UpdateState",
		"GraphicsApp", "ReadText", "WriteText", "ListDir", "DroppedFile", "DropCount",
		"ReadFile", "WriteFile", "OpenFile", "CloseFile", "Eof", "FilePos", "SeekFile",
		"ReadLine", "WriteLine", "ReadByte", "WriteByte", "ReadShort", "WriteShort",
		"ReadInt", "WriteInt", "ReadFloat", "WriteFloat", "ReadString", "WriteString", "FileSize",
		"BindAction", "UnbindAction", "ActionDown", "ActionHit", "SaveBinds", "LoadBinds",
		"TableCreate", "TableInsert", "TableGet", "TableCount", "TableSave", "TableLoad",
		"SaveSlot", "LoadSlot", "SavePrefab", "SpawnPrefab",
		"CreateAnimGraph", "AnimState", "AnimGo", "AnimUpdate", "ReloadScripts",
		"SolveTwoBone", "IKBend", "GuiPickFile",
		"GuiBeginMenuBar", "GuiEndMenuBar", "GuiMenu", "GuiEndMenu", "GuiMenuItem",
		"GuiBeginTabBar", "GuiEndTabBar", "GuiTabItem", "GuiEndTabItem",
		"GuiTree", "GuiTreePop", "GuiBeginChild", "GuiEndChild",
		"GuiOpenPopup", "GuiBeginPopup", "GuiEndPopup", "GuiSelectable",
		"BackBuffer", "FrontBuffer", "SetBuffer",
	}
}
