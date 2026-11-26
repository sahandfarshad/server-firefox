#!/usr/bin/env python3
"""Generates ../ui.html from ui.src.html + strings.py + langs.py."""
import json, os
os.chdir(os.path.dirname(os.path.abspath(__file__)))
from strings import KEYS, T
from langs import LANGS
for c, *_ in LANGS:
    assert len(T[c]) == len(KEYS), c
s = open("ui.src.html", encoding="utf-8").read()
s = s.replace("/*LANGS*/", json.dumps([list(x) for x in LANGS], ensure_ascii=False))
s = s.replace("/*I18N*/", json.dumps({c: dict(zip(KEYS, T[c])) for c, *_ in LANGS}, ensure_ascii=False))
open("../ui.html", "w", encoding="utf-8").write(s)
print("ui.html written")
