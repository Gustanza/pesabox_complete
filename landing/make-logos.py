#!/usr/bin/env python3
"""Builds the landing page's logo files in assets/ from the HelaBox logo pack.

    python3 landing/make-logos.py "<path to 'HELABOX FINAL LOGO' folder>"

Takes the Illustrator SVG exports, drops their baked-in background squares,
inlines the class fills (so several logos can live in one file) and writes:
  logo-light.svg  horizontal lockup (icon + wordmark) for light backgrounds
  logo-dark.svg   same, white wordmark / white-top icon for dark backgrounds
  icon.svg        full-colour icon, cropped
  icon-dark.svg   icon with the navy piece in white, for dark backgrounds
"""
import os
import re
import sys

PACK = sys.argv[1] if len(sys.argv) > 1 else os.path.expanduser('~/uploads/logo/HELABOX FINAL LOGO')
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'assets')
ICON_VIEWBOX = '64 64 672 672'          # icon art inside the 800x800 artboard
# "HelaBox" letters. The exports don't share an artboard: White/Reverse are
# 842x595, Fullcolor is 595x448 with the art scaled by ~0.7023 (+2.0, +15.1).
WORDMARK_VIEWBOX = {'white': '38 366 772 162', 'full': '28.7 272.1 542.2 113.8'}


def inner(path):
    s = open(path, encoding='utf-8').read()
    fills = dict(re.findall(r'\.(st\d+)\{fill:(#[0-9A-Fa-f]+);\}', s))
    body = s[s.find('>', s.find('<svg')) + 1:s.rfind('</svg>')]
    body = re.sub(r'<style.*?</style>', '', body, flags=re.S)
    body = re.sub(r'<rect[^>]*/>', '', body)  # background square
    body = re.sub(r'class="(st\d+)"', lambda m: 'fill="%s"' % fills[m.group(1)], body)
    return re.sub(r'\s+', ' ', body).strip()


def lockup(icon, wordmark, wm_viewbox):
    return ('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 416 100">'
            f'<svg x="0" y="0" width="100" height="100" viewBox="{ICON_VIEWBOX}">{icon}</svg>'
            f'<svg x="120" y="20" width="296" height="62" viewBox="{wm_viewbox}" '
            f'preserveAspectRatio="xMinYMid meet">{wordmark}</svg></svg>')


icon_full = inner(f'{PACK}/05-ICON/01-Fullcolor/ICON_HelaBox_Fullcolor.svg')
icon_rev = inner(f'{PACK}/05-ICON/02-Reverse/ICON_HelaBox_Reverse.svg')
wm_full = inner(f'{PACK}/01-Fullcolor/HelaBox_Fullcolor.svg')
wm_white = inner(f'{PACK}/04-White/HelaBox_White.svg')

os.makedirs(OUT, exist_ok=True)
files = {
    'logo-light.svg': lockup(icon_full, wm_full, WORDMARK_VIEWBOX['full']),
    'logo-dark.svg': lockup(icon_rev, wm_white, WORDMARK_VIEWBOX['white']),
    'icon.svg': f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{ICON_VIEWBOX}">{icon_full}</svg>',
    'icon-dark.svg': f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{ICON_VIEWBOX}">{icon_rev}</svg>',
}
for name, svg in files.items():
    with open(os.path.join(OUT, name), 'w', encoding='utf-8') as f:
        f.write(svg + '\n')
    print('wrote', name)
