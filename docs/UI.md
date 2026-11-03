# UI and the profiler

Two UI stacks share the 3D window. Dear ImGui is immediate: you call `Gui*` every frame and the panel is drawn after the scene. G3N widgets are nodes: you create them once and they stay on the window. 2D games use `Text`, `DrawImage`, and sprites instead. ImGui and G3N widgets need `Graphics3D`.

## A panel

```basic
Graphics3D 1280, 720, 0, 2
While Not KeyDown(1)
    GuiBegin "Game"
    GuiText "Health"
    If GuiButton("Pause") Then paused = 1
    GuiEnd
    RenderWorld
    Flip
Wend
```

`GuiBegin` opens a window. `GuiEnd` closes it. A missing `GuiEnd` is closed on `Flip`. `GuiButton` returns 1 on the frame it is clicked. `GuiSlider` returns the value, and passing the old value back in keeps the slider where the script put it.

## Theme, scale, and type

`UITheme "dark"` or `"light"` is ImGui's dark or light style. The choice is remembered if you call it before the first panel. `UIScale n` sets the font scale, clamped from 0.5 to 3.

`UIWrap width` is the wrap width for `GuiTextWrapped`. `UIAlign 0` is left, `1` centres, `2` is right. Alignment applies to `GuiText` and `GuiTextWrapped`.

`GuiRich` reads color tags in the string. `{#rrggbb}` sets the color. `{#}` or `{}` returns to the default. The tag is not drawn.

```basic
UITheme "light"
UIScale 1.25
UIAlign 1
GuiRich "{#c45c26}Paused{#}  press P"
```

## Rows, columns, anchors

`GuiSameLine` puts the next widget on the current row. `GuiHBox count [, gap]` does that for the next `count` buttons, text lines, sliders, or checkboxes, with `gap` pixels between them (default 8).

`GuiColumns n` starts an ImGui column row. There is no cell spanning.

`GuiAnchor x, y` places the next `GuiBegin` window. Both numbers are 0 to 1 across the framebuffer. `1, 0` is the top right. The window's own corner follows the same fraction, so `1, 1` sits the window's bottom-right on the screen's bottom-right.

`UINav flags` turns on ImGui navigation. `1` is keyboard, `2` is gamepad, `3` is both. Call it any time. It is applied when the ImGui context exists.

## G3N widgets

`CreatePanel`, `CreateButton`, `CreateLabel`, `CreateSlider`, `CreateCheckbox`, and `CreateEdit` are scene-graph widgets. `SetWidgetPos e, x, y` is pixels from the top left. `SetWidgetAnchor e, ax, ay` pins the widget to a fraction of the window, using the widget's size so `1, 0` sits on the right edge.

`TweenWidget e, x, y, seconds` slides it there. A duration of 0 moves it on this call. The slide uses the frame delta. It does not ease.

```basic
b = CreateButton("OK")
SetWidgetAnchor b, 1, 1
TweenWidget b, 24, 24, 0.4
```

Anchors are reapplied every frame from the window size. A tween and an anchor on the same widget fight. Use one.

## What this is not

There is no nine-slice on ImGui. `DrawNineSlice image, x, y, w, h, border` is a 2D image: the corners stay the border size, and the edges and the centre stretch. There is no localization table, no screen reader, and no extra widget toolkit. Compose `Gui*` or the G3N widgets.

## Profiler

`ShowProfiler 1` draws the corner overlay: FPS, frame time, script, physics, render, GPU, draws, triangles, heap, and GC.

Per function, the interpreter records inclusive time and exclusive time. Exclusive time is the function minus the functions it called. `StatsFuncMs(name$)` is the exclusive milliseconds since the program started. `StatsFuncCalls(name$)` is the call count. `StatsTopFunc$()` is the name with the highest exclusive time. Names are stored without case, so `Work` is `"work"`.

`StatsAssetKB(name$)` is the file size of a loaded mesh or texture, in kilobytes. The key is `"mesh "` or `"tex "` plus the file's base name, for example `"mesh hero.glb"`. `StatsAssetMB()` sums every tracked file. `StatsVRAMMB()` sums texture memory: the compressed byte size when the file is compressed, otherwise width × height × 4.

Run from the IDE (`MOON_IDE=1`) and the runtime prints one `@prof` line on stderr about once a second. The Profiler icon in the activity bar shows the frame lines, the slowest functions, and those asset sizes. A terminal run does not print `@prof`.
