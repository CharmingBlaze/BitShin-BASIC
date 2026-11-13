"""Generate the PS1-style textures (small, nearest-filtered, hand-noisy) and sound effects.
Standard library only. Run from the game folder: python tools/make_assets.py"""
import math
import pathlib
import random
import struct
import wave
import zlib

OUT = pathlib.Path(__file__).resolve().parent.parent / "assets"
OUT.mkdir(exist_ok=True)
random.seed(1987)


def png(name, w, h, px):
    raw = bytearray()
    for y in range(h):
        raw.append(0)
        for x in range(w):
            raw.extend(max(0, min(255, int(c))) for c in px(x, y))

    def chunk(tag, data):
        return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", zlib.crc32(tag + data))

    body = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0))
    body += chunk(b"IDAT", zlib.compress(bytes(raw), 9)) + chunk(b"IEND", b"")
    (OUT / f"{name}.png").write_bytes(body)


def noise_grid(size, amount):
    return [[random.uniform(-amount, amount) for _ in range(size)] for _ in range(size)]


def grey(v):
    return (v, v, v)


def textures():
    n = noise_grid(32, 5)
    png("ground", 32, 32, lambda x, y: grey(176 + n[y][x] + 0))

    n2 = noise_grid(32, 4)

    def stone(x, y):
        row = y // 8
        off = 8 if row % 2 else 0
        mortar = y % 8 == 0 or (x + off) % 16 == 0
        return grey((150 if mortar else 205) + n2[y][x])

    png("stone", 32, 32, stone)

    n3 = noise_grid(32, 4)

    def wood(x, y):
        plank = x % 8 == 0
        grain = math.sin((y + x * 3) * 0.7) * 14
        return grey((130 if plank else 196 + grain * 0.4) + n3[y][x])

    png("wood", 32, 32, wood)

    n4 = noise_grid(32, 3)

    def plaster(x, y):
        beam = y < 3 or y > 28 or x < 3 or x > 28 or 14 < x < 18
        return (150, 112, 92) if beam else grey(208 + n4[y][x])

    png("plaster", 32, 32, plaster)

    n5 = noise_grid(32, 4)

    def shingle(x, y):
        row = y // 4
        off = 4 if row % 2 else 0
        edge = y % 4 == 3 or (x + off) % 8 == 0
        return grey((160 if edge else 215) + n5[y][x])

    png("shingle", 32, 32, shingle)

    n6 = noise_grid(16, 24)
    png("cloth", 16, 16, lambda x, y: grey(196 + n6[y][x] + (14 if (x + y) % 2 else -14)))

    n7 = noise_grid(16, 48)
    png("fur", 16, 16, lambda x, y: grey(170 + n7[y][x] + (34 if (x + y * 2) % 4 == 0 else 0)))

    n8 = noise_grid(16, 6)

    def bark(x, y):
        return grey((160 if x % 5 == 0 else 205) + n8[y][x])

    png("bark", 16, 16, bark)


RATE = 22050  # a console-era sample rate


def wav(name, secs, fn, gain=0.8):
    frames = bytearray()
    count = int(RATE * secs)
    for i in range(count):
        t = i / RATE
        v = max(-1, min(1, fn(t, t / secs) * gain))
        frames += struct.pack("<h", int(v * 32000))
    with wave.open(str(OUT / f"{name}.wav"), "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(RATE)
        w.writeframes(bytes(frames))


def noise():
    return random.uniform(-1, 1)


def sounds():
    wav("swing", 0.2, lambda t, k: noise() * math.sin(k * math.pi) ** 2 * (0.5 + 0.5 * math.sin(t * 900)), 0.55)
    wav("gun", 0.55, lambda t, k: (noise() * (1 - k) ** 5 + math.sin(t * 2 * math.pi * 240) * (1 - k) ** 3 * 0.6
                                   + math.sin(t * 2 * math.pi * 1760) * 0.2 * (1 - k) ** 2), 0.95)
    wav("hit", 0.14, lambda t, k: (noise() * 0.7 + math.sin(t * 2 * math.pi * 160) * 0.7) * (1 - k) ** 3, 0.8)
    wav("hurt", 0.32, lambda t, k: math.sin(t * 2 * math.pi * (300 - 180 * k)) * (1 - k) ** 2 + noise() * 0.2 * (1 - k), 0.7)
    wav("growl", 0.7, lambda t, k: (math.sin(t * 2 * math.pi * 66) * 0.6 + noise() * 0.5
                                    * (0.5 + 0.5 * math.sin(t * 2 * math.pi * 31))) * math.sin(k * math.pi), 0.8)

    def chime(t, k):
        s = sum(math.sin(t * 2 * math.pi * f) * a for f, a in ((659.3, .5), (987.8, .3), (1661, .2), (2637, .08)))
        return s * (1 - k) ** 2.4

    wav("chime", 1.3, chime, 0.6)
    wav("pickup", 0.28, lambda t, k: math.sin(t * 2 * math.pi * (880 if k < 0.45 else 1318)) * (1 - k), 0.5)

    # Night air: a slow wind and a drone that loops without a click
    secs = 12
    state = [0.0]

    def night(t, k):
        state[0] += (noise() - state[0]) * 0.03
        fade = min(1, k * 40, (1 - k) * 40)
        swell = 0.6 + 0.4 * math.sin(k * 2 * math.pi * 2)
        drone = math.sin(t * 2 * math.pi * 55) * 0.16 + math.sin(t * 2 * math.pi * 82.5) * 0.08
        return (state[0] * 2.2 * swell + drone) * fade

    wav("night", secs, night, 0.5)


textures()
sounds()
print("assets written to", OUT)
