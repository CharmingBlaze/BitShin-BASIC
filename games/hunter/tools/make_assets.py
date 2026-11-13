"""Make the Greenwood sound effects. Standard library only.
Run from the game folder: python tools/make_assets.py"""
import math
import pathlib
import random
import struct
import wave

OUT = pathlib.Path(__file__).resolve().parent.parent / "assets"
OUT.mkdir(exist_ok=True)
random.seed(64)
RATE = 22050


def save(name, samples, rate=RATE):
    peak = max(1e-6, max(abs(s) for s in samples))
    k = 0.9 / peak if peak > 0.9 else 1.0
    with wave.open(str(OUT / f"{name}.wav"), "wb") as f:
        f.setnchannels(1)
        f.setsampwidth(2)
        f.setframerate(rate)
        f.writeframes(b"".join(struct.pack("<h", int(max(-1, min(1, s * k)) * 32767)) for s in samples))


def env(t, a, d):
    """Attack then exponential decay."""
    if t < a:
        return t / a
    return math.exp(-(t - a) / d)


def lowpass(xs, k):
    y = 0.0
    out = []
    for x in xs:
        y += (x - y) * k
        out.append(y)
    return out


def noise(n):
    return [random.uniform(-1, 1) for _ in range(n)]


def secs(s):
    return int(RATE * s)


# Bowstring drawn: a dry creak, wood and string under tension.
def draw():
    n = secs(0.55)
    out = []
    ph = 0.0
    for i in range(n):
        t = i / RATE
        f = 90 + 60 * t
        ph += f / RATE
        grain = 1.0 if random.random() < 0.02 + 0.06 * t else 0.0
        s = grain * random.uniform(0.3, 1) * math.sin(ph * 6.283) * 0.8
        s += random.uniform(-1, 1) * 0.05 * t
        out.append(s * min(1, t * 8) * (1 - t / 0.55) ** 0.3)
    return lowpass(out, 0.35)


# Release: a plucked string, then the arrow's whistle.
def twang():
    n = secs(0.45)
    out = []
    for i in range(n):
        t = i / RATE
        s = math.sin(6.283 * 132 * t) * 0.7 + math.sin(6.283 * 264 * t) * 0.3
        s *= env(t, 0.002, 0.07) * (1 + 0.3 * math.sin(6.283 * 9 * t))
        s += random.uniform(-1, 1) * 0.18 * env(t, 0.03, 0.12)
        out.append(s)
    return out


def thunk():
    n = secs(0.25)
    out = []
    for i in range(n):
        t = i / RATE
        s = math.sin(6.283 * (180 - 300 * t) * t) * env(t, 0.001, 0.05)
        s += random.uniform(-1, 1) * env(t, 0.001, 0.015) * 0.6
        out.append(s)
    return lowpass(out, 0.5)


def hit():
    n = secs(0.3)
    out = []
    for i in range(n):
        t = i / RATE
        s = random.uniform(-1, 1) * env(t, 0.001, 0.04) * 0.8
        s += math.sin(6.283 * (220 - 260 * t) * t) * env(t, 0.002, 0.08)
        out.append(s)
    return lowpass(out, 0.4)


# A deer's alarm: a sharp nasal snort-bleat.
def deer():
    n = secs(0.5)
    out = []
    ph = 0.0
    for i in range(n):
        t = i / RATE
        f = 520 - 340 * t
        ph += f / RATE
        s = (math.sin(ph * 6.283) + 0.5 * math.sin(ph * 12.566) + 0.25 * math.sin(ph * 18.85))
        s *= env(t, 0.02, 0.12)
        s += random.uniform(-1, 1) * 0.35 * env(t, 0.005, 0.06)
        out.append(s * 0.6)
    return lowpass(out, 0.45)


def boar():
    n = secs(0.7)
    out = []
    ph = 0.0
    for i in range(n):
        t = i / RATE
        f = 85 + 25 * math.sin(6.283 * 3 * t)
        ph += f / RATE
        pulse = 1.0 if (ph % 1) < 0.18 else -0.2
        s = pulse * env(t, 0.03, 0.25)
        s += random.uniform(-1, 1) * 0.4 * env(t, 0.02, 0.2)
        out.append(s)
    return lowpass(out, 0.18)


def squeak():
    n = secs(0.18)
    out = []
    ph = 0.0
    for i in range(n):
        t = i / RATE
        ph += (1500 + 900 * math.sin(6.283 * 18 * t)) / RATE
        out.append(math.sin(ph * 6.283) * env(t, 0.005, 0.05) * 0.5)
    return out


def hurt():
    n = secs(0.35)
    out = []
    for i in range(n):
        t = i / RATE
        s = math.sin(6.283 * (300 - 400 * t) * t) * env(t, 0.003, 0.1)
        s += random.uniform(-1, 1) * env(t, 0.002, 0.04) * 0.5
        out.append(s)
    return out


def bell(notes, step, length, decay):
    n = secs(length)
    out = [0.0] * n
    for k, f in enumerate(notes):
        st = secs(step * k)
        for i in range(st, n):
            t = (i - st) / RATE
            e = env(t, 0.004, decay)
            out[i] += (math.sin(6.283 * f * t) + 0.3 * math.sin(6.283 * f * 2.01 * t) + 0.12 * math.sin(6.283 * f * 3.98 * t)) * e * 0.4
    return out


def note(name):
    names = {"C": -9, "D": -7, "E": -5, "F": -4, "G": -2, "A": 0, "B": 2}
    octave = int(name[-1])
    semi = names[name[0]] + (1 if "#" in name else 0) + (octave - 4) * 12
    return 440 * 2 ** (semi / 12)


# A short rising phrase for a trophy, and a longer one for the white stag.
def chime():
    return bell([note(x) for x in ("G5", "A5", "B5", "D6")], 0.07, 1.1, 0.35)


def fanfare():
    return bell([note(x) for x in ("D5", "F#5", "A5", "D6", "A5", "D6", "F#6")], 0.11, 2.4, 0.6)


def pickup():
    return bell([note("E6"), note("B6")], 0.05, 0.4, 0.12)


# The wood: wind in leaves, a stream far off, birds now and then. 12 s, loops cleanly.
def forest():
    n = secs(12)
    wind = lowpass(noise(n), 0.02)
    gust = [0.55 + 0.45 * math.sin(6.283 * i / n * 2) * math.sin(6.283 * i / n * 3 + 1) for i in range(n)]
    out = [w * g * 2.2 for w, g in zip(wind, gust)]
    hiss = lowpass(noise(n), 0.25)
    for i in range(n):
        out[i] += hiss[i] * 0.03
    # Birds: little chirps and a two-note call
    for _ in range(26):
        st = random.randint(0, n - secs(0.5))
        base = random.choice([2600, 3100, 3600, 4200])
        kind = random.random()
        count = random.randint(1, 4)
        for c in range(count):
            s0 = st + c * secs(0.11)
            ln = secs(0.07 if kind < 0.6 else 0.16)
            for i in range(ln):
                if s0 + i >= n:
                    break
                t = i / RATE
                f = base * (1 + 0.25 * math.sin(6.283 * 30 * t)) if kind < 0.6 else base * (1.15 - 0.4 * t / (ln / RATE))
                out[s0 + i] += math.sin(6.283 * f * t) * math.sin(math.pi * i / ln) * 0.09
    # Crossfade the end into the start so the loop has no seam
    fade = secs(0.6)
    for i in range(fade):
        k = i / fade
        out[i] = out[i] * k + out[n - fade + i] * (1 - k)
    return out[: n - fade]


for name, fn in [("draw", draw), ("twang", twang), ("thunk", thunk), ("hit", hit), ("deer", deer), ("boar", boar),
                 ("squeak", squeak), ("hurt", hurt), ("chime", chime), ("fanfare", fanfare), ("pickup", pickup),
                 ("forest", forest)]:
    save(name, fn())
print("ok")
