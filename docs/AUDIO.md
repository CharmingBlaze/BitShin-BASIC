# Audio

Sounds are short clips decoded up front. Music streams from disk. Both play through Oto. There is no OpenAL. The command list is in [COMMANDS.md](COMMANDS.md).

`LoadSound` returns an id. `PlaySound` and `LoopSound` start it. `PauseSound` holds the same voice. `ResumeSound` continues it. `StopSound` also pauses, which is the older name for the same hold. `FreeSound` closes the voice and drops the id.

`LoadMusic` is the streaming form. `PlayMusic` loops it. A stream can pause, resume, pan, and fade. It cannot seek, and `SetSoundEcho` does nothing on it.

## Volume, pan, fade

`SetSoundVolume s, n` sets the clip's own volume, from 0 to 1. `SetSoundPitch s, n` changes speed while it plays. `1` is the recorded pitch.

`SetSoundPan s, pan` is stereo balance. `-1` is left, `0` is center, `1` is right. It applies to the voice that is already playing.

`FadeSound s, volume, seconds` moves the clip volume to `volume` over `seconds`. A zero time jumps. The fade runs on `Flip`, using the frame delta. `SetSoundVolume` cancels a fade in progress.

`SeekSound s, seconds` jumps a decoded clip. It returns `1` when the voice moved and `0` for a stream or a sound that is not playing. `SoundTime(s)` is how many seconds of a decoded clip have been read.

```basic
step = LoadSound("step.ogg")
SetSoundPan step, -0.4
PlaySound step
FadeSound step, 0, 0.8
```

## Buses

A bus is a volume group. `SoundBus s, "music"` puts that clip on a bus. `SetAudioBus "music", 0.4` scales every clip on it. `SetMasterVolume n` scales every playing voice. The heard volume is clip × bus × master, clamped to 0–1.

Changing a bus or the master updates voices that are already playing. A one-shot `PlaySound3D` volume is applied at the start. The next `SetSoundVolume`, fade, or bus change uses the clip's own volume again.

## Echo

`SetSoundEcho s, wet` mixes an 80 ms slapback on a decoded clip. `wet` is 0 (dry) through 1. This is a short delay, not a room, not occlusion, and not a music stream.

## Where a sound is

`EmitSound s, x, y, z` sets volume and pan from the listener once, when the sound starts. `PlaySound3D` sets volume from distance and leaves pan alone. Neither follows the source after that. `SetListener e` picks the listener for `EmitSound`. The default is the active camera. `SetSoundRange s, min, max` is the near and far distance used by that calculation. `PlaySound3D` uses its own `maxDist` argument.
