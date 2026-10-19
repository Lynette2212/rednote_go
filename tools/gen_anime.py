# -*- coding: utf-8 -*-
"""生成 3 段原创二次元风格测试视频（程序化绘制 + 合成配乐）。"""
import math
import os
import random
import shutil
import subprocess
import wave
from array import array

from PIL import Image, ImageDraw, ImageFilter

ROOT = "C:/Users/kaola/Desktop/rednote_go"
FFMPEG = f"{ROOT}/desktop/native/third_party/ffmpeg/bin/ffmpeg.exe"
OUT_DIR = f"{ROOT}/desktop/assets/videos"
TMP = "C:/tmp/anime_frames"

W, H = 1080, 1920          # 1.5x 超采样
OW, OH = 720, 1280
FPS = 24
DURATION = 10
FRAMES = FPS * DURATION


def lerp(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def gradient(stops):
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / (H - 1)
        for i in range(len(stops) - 1):
            p0, c0 = stops[i]
            p1, c1 = stops[i + 1]
            if p0 <= t <= p1:
                local = (t - p0) / max(1e-6, p1 - p0)
                d.line([(0, y), (W, y)], fill=lerp(c0, c1, local))
                break
    return img.convert("RGBA")


def glow(d, center, radius, color, strength=60, steps=6):
    x, y = center
    for i in range(steps, 0, -1):
        r = radius * i / steps
        a = int(strength * (1 - i / steps) ** 2)
        d.ellipse((x - r, y - r, x + r, y + r), fill=(*color, a))


def draw_stars(d, rng, count, max_y):
    for _ in range(count):
        x = rng.randint(0, W)
        y = rng.randint(0, int(max_y))
        r = rng.choice((1, 1, 1, 2, 2, 3))
        a = rng.randint(60, 210)
        d.ellipse((x - r, y - r, x + r, y + r), fill=(255, 255, 255, a))


def draw_sakura_bg(rng):
    img = gradient([(0.0, (10, 12, 32)), (0.35, (28, 20, 58)),
                    (0.62, (74, 38, 86)), (1.0, (16, 12, 30))])
    ov = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(ov)
    draw_stars(d, rng, 220, H * 0.72)
    glow(d, (800, 330), 330, (250, 232, 255), 42, 8)
    d.ellipse((660, 190, 940, 470), fill=(252, 246, 214, 255))
    img = Image.alpha_composite(img, ov)

    d = ImageDraw.Draw(img)
    # 远山
    d.polygon([(0, 1240), (220, 1110), (400, 1230), (560, 1160), (760, 1260), (920, 1140), (1080, 1240), (1080, 1450), (0, 1450)],
              fill=(20, 18, 44))
    d.polygon([(0, 1360), (260, 1260), (520, 1360), (760, 1280), (1080, 1360), (1080, 1550), (0, 1550)],
              fill=(14, 14, 34))
    # 山丘
    d.ellipse((120, 1390, 1150, 2050), fill=(10, 12, 28))

    # 鸟居
    red = (86, 24, 48)
    d.rectangle((250, 1180, 286, 1510), fill=red)
    d.rectangle((430, 1180, 466, 1510), fill=red)
    d.polygon([(190, 1160), (526, 1140), (526, 1178), (190, 1198)], fill=red)
    d.rectangle((220, 1250, 500, 1278), fill=red)
    d.rectangle((330, 1170, 360, 1300), fill=red)
    # 灯笼
    for x, y in ((232, 1300), (452, 1290)):
        glow(d, (x, y), 55, (255, 170, 90), 80, 5)
        d.ellipse((x - 18, y - 24, x + 18, y + 24), fill=(255, 186, 102, 255))

    # 少女与地面
    draw_girl_back(d, 540, 1660, 1.5, (16, 10, 24), sway=0.0)
    return img


def draw_girl_back(d, x, y, s, color, sway=0.0, hat=False):
    head_r = 30 * s
    head_y = y - 300 * s
    # 长发
    d.ellipse((x - head_r * 1.35, head_y - head_r * 1.5, x + head_r * 1.35, head_y + head_r * 1.2), fill=color)
    d.polygon([(x - head_r, head_y), (x + head_r, head_y),
               (x + head_r * 1.5 + sway, y - 60 * s), (x + head_r * 0.4, y - 40 * s),
               (x - head_r * 1.4 + sway, y - 70 * s)], fill=color)
    # 身体
    d.polygon([(x - 34 * s, head_y + 26 * s), (x + 34 * s, head_y + 26 * s),
               (x + 72 * s + sway, y - 50 * s), (x - 72 * s + sway, y - 50 * s)], fill=color)
    # 手臂
    d.line([(x - 32 * s, head_y + 40 * s), (x - 60 * s + sway * 0.5, y - 120 * s)], fill=color, width=int(13 * s))
    d.line([(x + 32 * s, head_y + 40 * s), (x + 60 * s + sway * 0.5, y - 120 * s)], fill=color, width=int(13 * s))
    # 腿
    d.line([(x - 18 * s, y - 60 * s), (x - 20 * s, y)], fill=color, width=int(16 * s))
    d.line([(x + 18 * s, y - 60 * s), (x + 20 * s, y)], fill=color, width=int(16 * s))
    if hat:
        d.ellipse((x - 120 * s, head_y - 40 * s, x + 120 * s, head_y + 10 * s), fill=(30, 22, 20))
        d.pieslice((x - 60 * s, head_y - 70 * s, x + 60 * s, head_y + 40 * s), 180, 360, fill=(36, 26, 22))
    # 发带 / 轮廓光
    d.line([(x + head_r * 1.1, head_y - head_r), (x + head_r * 1.5 + sway, y - 80 * s)],
           fill=(255, 170, 200, 90), width=3)


def draw_sunset_bg(rng):
    horizon = int(H * 0.62)
    img = gradient([(0.0, (34, 28, 76)), (0.3, (120, 56, 112)),
                    (0.5, (226, 110, 104)), (0.62, (255, 190, 128)), (1.0, (255, 190, 128))])
    ov = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(ov)
    glow(d, (520, horizon - 20), 360, (255, 210, 150), 80, 9)
    d.ellipse((390, 1030, 650, 1290), fill=(255, 226, 170, 255))
    # 云
    clouds = [(100, 620, 520, 90), (520, 760, 700, 70), (620, 480, 540, 60), (40, 940, 460, 50)]
    for x, y, w, h in clouds:
        d.ellipse((x, y, x + w, y + h), fill=(255, 180, 160, 80))
        d.ellipse((x + w * 0.2, y - h * 0.4, x + w * 0.8, y + h * 0.6), fill=(255, 205, 170, 70))
    img = Image.alpha_composite(img, ov)

    d = ImageDraw.Draw(img)
    # 海
    d.rectangle((0, horizon, W, H), fill=(30, 48, 94))
    for y in range(horizon, H, 4):
        t = (y - horizon) / (H - horizon)
        d.line([(0, y), (W, y)], fill=lerp((54, 76, 130), (10, 18, 42), t))
    # 太阳倒影
    for i in range(46):
        y = horizon + i * 12
        t = i / 46
        width = int(90 * (1 - t * 0.5))
        offset = int(math.sin(i * 0.8) * 24)
        d.line([(520 - width + offset, y), (520 + width + offset, y)],
               fill=(255, 214, 150, max(20, 170 - int(t * 150))), width=max(2, 8 - i // 8))
    # 浪
    for i in range(14):
        y = horizon + 20 + i * 58
        shift = int(math.sin(i * 0.9) * 40)
        d.arc((60 + shift, y, W - 60 + shift, y + 90), 200, 340,
              fill=(255, 235, 220, 40), width=3)

    # 少女与帽子
    draw_girl_back(d, 800, 1810, 1.6, (26, 14, 24), sway=10.0, hat=True)
    # 栏杆
    d.rectangle((0, 1730, W, 1752), fill=(30, 18, 24))
    d.rectangle((0, 1830, W, 1852), fill=(30, 18, 24))
    for x in range(60, W, 180):
        d.rectangle((x, 1730, x + 18, H), fill=(26, 16, 22))
    return img


def draw_neon_bg(rng):
    img = gradient([(0.0, (6, 8, 22)), (0.55, (18, 14, 36)), (1.0, (10, 10, 26))])
    d = ImageDraw.Draw(img)
    # 楼
    buildings = [(0, 620, 190, 1600), (150, 430, 200, 1600), (330, 540, 170, 1600),
                 (470, 350, 220, 1600), (660, 500, 160, 1600), (790, 600, 290, 1600)]
    for idx, (x, top, w, bottom) in enumerate(buildings):
        d.rectangle((x, top, x + w, bottom), fill=(10, 10, 24))
        for wy in range(top + 40, bottom - 60, 70):
            for wx in range(x + 24, x + w - 24, 46):
                if rng.random() > 0.45:
                    d.rectangle((wx, wy, wx + 18, wy + 26), fill=(255, 206, 120, 180))
    # 霓虹招牌
    signs = [(90, 760, 22, 220, (255, 62, 142)), (250, 600, 26, 160, (64, 224, 255)),
             (390, 880, 20, 200, (255, 214, 92)), (520, 520, 26, 240, (255, 62, 142)),
             (700, 820, 20, 180, (64, 224, 255)), (880, 680, 22, 210, (255, 120, 200))]
    for x, y, w, h, color in signs:
        for k in range(3):
            pad = 10 + k * 10
            d.rounded_rectangle((x - pad, y - pad, x + w + pad, y + h + pad),
                                radius=14, fill=(*color, 22 - k * 6))
        d.rectangle((x, y, x + w, y + h), fill=(*color, 235))
    # 湿街与倒影
    d.rectangle((0, 1600, W, H), fill=(8, 8, 20))
    for x, y, w, h, color in signs:
        for i in range(0, 260, 18):
            d.line([(x + 4 + rng.randint(-6, 6), 1610 + i), (x + w - 4 + rng.randint(-6, 6), 1610 + i)],
                   fill=(*color, max(10, 90 - i // 3)), width=6)
    for _ in range(10):
        cx = rng.randint(0, W)
        cy = rng.randint(1680, H - 40)
        rw = rng.randint(60, 160)
        d.ellipse((cx - rw, cy - 20, cx + rw, cy + 20), fill=(40, 60, 100, 26))
    return img


def draw_umbrella_girl(d, x, y, s, t, color=(10, 10, 20)):
    head_y = y - 300 * s
    # 伞
    tilt = math.sin(t * 1.2) * 0.03
    cx = x + 20 * s + tilt * 400
    r = 210 * s
    d.pieslice((cx - r, head_y - 150 * s - r * 0.55, cx + r, head_y - 150 * s + r * 1.1),
               180, 360, fill=color)
    for i in range(-3, 4):
        d.line([(cx, head_y - 150 * s), (cx + i * r / 3.5, head_y - 150 * s - r * 0.55)],
               fill=(30, 30, 48), width=3)
    d.line([(cx, head_y - 150 * s), (cx, y - 120 * s)], fill=(40, 40, 60), width=6)
    # 人物
    draw_girl_back(d, x, y, s, color, sway=6.0)
    # 霓虹轮廓
    d.arc((cx - r, head_y - 150 * s - r * 0.55, cx + r, head_y - 150 * s + r * 1.1),
          185, 300, fill=(255, 80, 170, 170), width=4)


def draw_cat(d, x, y, s):
    color = (8, 8, 16)
    d.ellipse((x - 40 * s, y - 60 * s, x + 40 * s, y), fill=color)
    d.ellipse((x - 26 * s, y - 96 * s, x + 26 * s, y - 44 * s), fill=color)
    d.polygon([(x - 24 * s, y - 90 * s), (x - 10 * s, y - 122 * s), (x - 2 * s, y - 90 * s)], fill=color)
    d.polygon([(x + 24 * s, y - 90 * s), (x + 10 * s, y - 122 * s), (x + 2 * s, y - 90 * s)], fill=color)
    d.arc((x + 20 * s, y - 50 * s, x + 100 * s, y + 10 * s), 250, 100, fill=color, width=int(10 * s))


def petals_layer(d, t, rng):
    for i in range(85):
        x0 = rng.randint(-40, W + 40)
        y0 = rng.randint(0, H)
        speed = rng.uniform(50, 120)
        size = rng.uniform(5, 12)
        phase = rng.uniform(0, 6.28)
        x = x0 + math.sin(t * 0.8 + phase) * 55
        y = (y0 + t * speed) % (H + 100) - 50
        color = rng.choice(((255, 190, 214, 200), (255, 214, 228, 180), (246, 160, 196, 190)))
        d.ellipse((x - size, y - size * 0.55, x + size, y + size * 0.55), fill=color)


def fireflies_layer(d, t, rng):
    for i in range(14):
        x = rng.randint(100, W - 100) + math.sin(t * 0.6 + i) * 30
        y = rng.randint(900, 1650) + math.cos(t * 0.5 + i) * 20
        r = rng.randint(3, 6)
        glow(d, (x, y), 30, (255, 220, 120), 40, 4)
        d.ellipse((x - r, y - r, x + r, y + r), fill=(255, 240, 170, 230))


def render_scene(name, bg_factory, dynamic, seed):
    rng = random.Random(seed)
    bg = bg_factory(rng)
    frame_dir = os.path.join(TMP, name)
    shutil.rmtree(frame_dir, ignore_errors=True)
    os.makedirs(frame_dir, exist_ok=True)
    for i in range(FRAMES):
        t = i / FPS
        data = dynamic(bg, t, rng)
        data = data.convert("RGB").resize((OW, OH), Image.LANCZOS)
        data.save(os.path.join(frame_dir, f"{i:04d}.jpg"), quality=92)
        if i % 48 == 0:
            print(name, i, "/", FRAMES, flush=True)
    wav_path = os.path.join(TMP, f"{name}.wav")
    make_audio(name, wav_path)
    out = os.path.join(OUT_DIR, f"{name}.mp4")
    subprocess.run([
        FFMPEG, "-hide_banner", "-loglevel", "error", "-y",
        "-framerate", str(FPS), "-i", os.path.join(frame_dir, "%04d.jpg"),
        "-i", wav_path,
        "-c:v", "libx264", "-preset", "medium", "-crf", "20", "-pix_fmt", "yuv420p",
        "-c:a", "aac", "-b:a", "128k", "-shortest", "-movflags", "+faststart", out,
    ], check=True)
    shutil.rmtree(frame_dir, ignore_errors=True)
    print("written", out, flush=True)


def sakura_dynamic(bg, t, rng):
    frame = bg.copy()
    ov = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(ov)
    # 云
    for k, (x, y, s) in enumerate(((80, 520, 1.0), (460, 700, 0.8), (700, 560, 1.1))):
        dx = (t * 12 + k * 80) % (W + 500) - 250
        d.ellipse((dx + x, y, dx + x + 420 * s, y + 90 * s), fill=(255, 255, 255, 14))
    petals_layer(d, t, rng)
    fireflies_layer(d, t, rng)
    frame = Image.alpha_composite(frame, ov)
    d = ImageDraw.Draw(frame)
    # 少女发丝摆动
    draw_girl_back(d, 540, 1660, 1.5, (16, 10, 24), sway=math.sin(t * 1.4) * 8)
    return frame


def sunset_dynamic(bg, t, rng):
    frame = bg.copy()
    ov = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(ov)
    for k, (x, y, s) in enumerate(((0, 760, 1.2), (420, 560, 0.9), (680, 860, 1.0))):
        dx = (t * 16 + k * 160) % (W + 600) - 300
        d.ellipse((dx + x, y, dx + x + 480 * s, y + 70 * s), fill=(255, 190, 170, 36))
    for k in range(6):
        bx = (t * 80 + k * 170) % (W + 120) - 60
        by = 700 - (k % 3) * 45 + math.sin(t * 2 + k) * 8
        d.line([(bx - 16, by), (bx, by - 8), (bx + 16, by)], fill=(70, 40, 60, 210), width=4)
    frame = Image.alpha_composite(frame, ov)
    d = ImageDraw.Draw(frame)
    draw_girl_back(d, 800, 1810, 1.6, (26, 14, 24), sway=math.sin(t * 1.8) * 14, hat=True)
    return frame


def neon_dynamic(bg, t, rng):
    frame = bg.copy()
    ov = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(ov)
    # 雨
    for i in range(120):
        x0 = rng.randint(-100, W)
        y0 = rng.randint(0, H)
        speed = rng.uniform(900, 1500)
        length = rng.uniform(26, 70)
        x = (x0 + t * 180) % (W + 200) - 100
        y = (y0 + t * speed) % (H + 100) - 50
        d.line([(x, y), (x - 8, y + length)], fill=(180, 210, 255, rng.randint(30, 90)), width=2)
    # 霓虹闪烁
    for x, y, w, h in ((90, 760, 22, 220), (250, 600, 26, 160), (390, 880, 20, 200),
                       (520, 520, 26, 240), (700, 820, 20, 180), (880, 680, 22, 210)):
        a = int(40 + 30 * (0.5 + 0.5 * math.sin(t * 3 + x)))
        d.rectangle((x - 4, y - 4, x + w + 4, y + h + 4), fill=(255, 255, 255, a))
    # 水洼涟漪
    for k in range(4):
        cx = 160 + k * 250
        cy = 1770 + (k % 2) * 80
        r = 20 + (t * 40 + k * 30) % 80
        d.ellipse((cx - r, cy - r * 0.28, cx + r, cy + r * 0.28),
                  outline=(120, 220, 255, max(10, 80 - int(r))), width=2)
    frame = Image.alpha_composite(frame, ov)
    d = ImageDraw.Draw(frame)
    draw_umbrella_girl(d, 540, 1810, 1.55, t)
    draw_cat(d, 880, 1840, 1.2)
    return frame


def make_audio(name, path):
    sr = 48000
    n = int(sr * DURATION)
    buf = [0.0] * n

    progressions = {
        "anime_sakura": [[220.00, 261.63, 329.63], [174.61, 220.00, 261.63],
                         [261.63, 329.63, 392.00], [196.00, 246.94, 293.66]],
        "anime_sunset": [[261.63, 329.63, 392.00], [196.00, 246.94, 293.66],
                         [220.00, 261.63, 329.63], [174.61, 220.00, 261.63]],
        "anime_neon": [[146.83, 174.61, 220.00], [116.54, 146.83, 174.61],
                       [174.61, 220.00, 261.63], [130.81, 164.81, 196.00]],
    }
    chords = progressions[name]

    def add(start, freq, dur, amp, decay=2.5, harmonics=(1.0, 0.35, 0.12)):
        begin = int(start * sr)
        end = min(n, begin + int(dur * sr))
        for i in range(begin, end):
            t = (i - begin) / sr
            env = math.exp(-t * decay) * min(1.0, t * 12)
            v = sum(h * math.sin(2 * math.pi * freq * (k + 1) * t) for k, h in enumerate(harmonics))
            buf[i] += amp * env * v

    # 铺底和弦
    seg = DURATION / len(chords)
    for idx, chord in enumerate(chords):
        for note in chord:
            add(idx * seg, note, seg * 1.35, 0.045, 0.9)
    # 琶音
    step = 0.3125
    t = 0.0
    k = 0
    while t < DURATION:
        chord = chords[int(t / seg) % len(chords)]
        note = chord[k % len(chord)] * 2
        add(t, note, 1.2, 0.055, 3.2)
        t += step
        k += 1
    # 随机铃声
    rng = random.Random(7 if name == "anime_sakura" else 11 if name == "anime_sunset" else 13)
    for _ in range(10):
        start = rng.uniform(0, DURATION - 1)
        note = rng.choice(chords)[0] * 4
        add(start, note, 1.6, 0.03, 2.0)
    # 底噪（霓虹雨夜）
    if name == "anime_neon":
        for i in range(n):
            buf[i] += rng.uniform(-1, 1) * 0.002
    # 淡入淡出 + 归一
    fade = int(sr * 0.6)
    for i in range(fade):
        buf[i] *= i / fade
        buf[n - 1 - i] *= i / fade
    peak = max(1e-6, max(abs(v) for v in buf))
    scale = 0.72 / peak
    data = array("h", (int(max(-1, min(1, v * scale)) * 32767) for v in buf))
    with wave.open(path, "wb") as f:
        f.setnchannels(1)
        f.setsampwidth(2)
        f.setframerate(sr)
        f.writeframes(data.tobytes())


def main():
    os.makedirs(OUT_DIR, exist_ok=True)
    os.makedirs(TMP, exist_ok=True)
    render_scene("anime_sakura", draw_sakura_bg, sakura_dynamic, 3)
    render_scene("anime_sunset", draw_sunset_bg, sunset_dynamic, 5)
    render_scene("anime_neon", draw_neon_bg, neon_dynamic, 9)


if __name__ == "__main__":
    main()
