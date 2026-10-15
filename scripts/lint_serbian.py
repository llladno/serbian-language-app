#!/usr/bin/env python3
"""Линтер сербского текста курса: орфография, экавица, смешение алфавитов.

Закрывает класс ошибок «слова нет в сербском / оно хорватское или иекавское /
в слове перемешаны латиница и кириллица» (см. docs/SERBIAN_ERRORS.md).
Проверяет ТОЛЬКО то, что обязано быть правильным: словарь (`latin`), примеры,
правильные ответы (`answer` / `accept` / `say`), реплики диалогов, сербские
код-спаны теории и строки текстов для чтения. Намеренно неверные варианты
(отвлекающие `options`, `fix_error.prompt`, «не `X`» в теории) не трогает.

Категории (все — ошибки, код возврата 1):
  SPELL   — слова нет в Hunspell-словаре сербского (LibreOffice, sr-Latn)
  YAT     — иекавизм (`lijepo`, `vrijeme`) или хорватизм (`tisuća`, `kruh`) в
            курсе по экавице
  DIAC    — слово без диакритики, у которого «двойник с ć/č/š/ž/đ» тоже слово
            (`skoli` ↔ `školi`): орфография молчит, нужно осознанное решение —
            подтвердить в `diac_ok` (причина обязательна) или исправить
  CYR     — `cyrillic` в vocab.yaml не совпадает с транслитерацией `latin`
  SCRIPT  — в одном слове латиница вперемешку с кириллицей (омоглифы)

Исключения — `content/lint-serbian.yaml`; у каждой записи ОБЯЗАТЕЛЬНА причина.

    pip install spylls pyyaml
    python3 scripts/lint_serbian.py                 # весь курс
    python3 scripts/lint_serbian.py --lesson 49     # один урок
    python3 scripts/lint_serbian.py --prose         # + Latin в `explain` (шумно, предупреждения)
    python3 scripts/lint_serbian.py --self-test     # проверка самого линтера

Словарь (~4.5 МБ) скачивается при первом запуске в scripts/.cache/ (gitignore).
Лицензия словаря: LGPL-3 / MPL-2 / GPL-3, автор Милутин Смиляньич.
"""
from __future__ import annotations

import argparse
import glob
import itertools
import os
import re
import sys
import unicodedata
import urllib.request
from dataclasses import dataclass

import yaml

try:
    from spylls.hunspell import Dictionary
except ImportError:
    sys.exit("spylls не установлен — выполни: pip install spylls pyyaml")

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CONTENT = os.path.join(ROOT, "content")
CACHE = os.path.join(ROOT, "scripts", ".cache")
DICT_URL = "https://raw.githubusercontent.com/LibreOffice/dictionaries/master/sr/"
DICT_BASE = os.path.join(CACHE, "sr-Latn")
CONFIG = os.path.join(CONTENT, "lint-serbian.yaml")

LAT = "A-Za-zĆćČčĐđŠšŽž"
TOKEN_RE = re.compile(rf"[{LAT}]+(?:-[{LAT}]+)*")
ANY_LETTERS_RE = re.compile(rf"[{LAT}Ѐ-ӿ]+")
CYR_RE = re.compile(r"[Ѐ-ӿ]")
LAT_RE = re.compile(rf"[{LAT}]")
# код-спан теории, который не является сербской фразой: суффикс, формула, цифры
NOT_PHRASE_RE = re.compile(r"[+=→<>*_\n\d…]|\.\.\.|[Ѐ-ӿ]")
# «не `X`», «не «X»», «❌ …» — намеренно неверные примеры в теории
NEGATED_RE = re.compile(r"(?:не|NOT|not|вместо)\s+[«`][^»`]*[»`]|❌[^\n]*", re.I)


# Латиница → кириллица, 1:1. Копия таблицы из tts.py (sr_lat_to_cyr): сам tts.py не
# импортировать — он требует edge-tts. Менять синхронно.
_DIGRAPHS = [("DŽ", "Џ"), ("Dž", "Џ"), ("dž", "џ"), ("LJ", "Љ"), ("Lj", "Љ"), ("lj", "љ"),
             ("NJ", "Њ"), ("Nj", "Њ"), ("nj", "њ")]
_LAT2CYR = str.maketrans(dict(zip("abcčćdđefghijklmnoprsštuvzž", "абцчћдђефгхијклмнопрсштувзж")))
_LAT2CYR.update(str.maketrans({k.upper(): v.upper() for k, v in zip("abcčćdđefghijklmnoprsštuvzž", "абцчћдђефгхијклмнопрсштувзж")}))


def lat_to_cyr(s: str) -> str:
    s = unicodedata.normalize("NFC", s)
    for a, b in _DIGRAPHS:
        s = s.replace(a, b)
    return s.translate(_LAT2CYR)


def vocab_cyrillic_findings() -> list["Finding"]:
    """`cyrillic` должно быть транслитом `latin` — целиком, либо без пояснений в скобках,
    либо только первый вариант из «a / b»; апострофы не учитываются."""
    out = []
    for e in load_yaml(os.path.join(CONTENT, "vocab.yaml")) or []:
        lat, cy = e.get("latin", ""), unicodedata.normalize("NFC", e.get("cyrillic", ""))
        if not lat or not cy:
            continue
        full = lat.replace("'", "").strip()
        base = re.sub(r"\s*\([^)]*\)", "", full).strip()    # без пояснений в скобках
        ok = {lat_to_cyr(x) for x in (full, base, base.split(" / ")[0].strip())}
        if cy not in ok:
            out.append(Finding("CYR", f"vocab.yaml:{e.get('id', '?')}", cy,
                               f"не совпадает с транслитом latin → {lat_to_cyr(base)}", lat))
    return out


# ----------------------------------------------------------------- словарь --

def ensure_dictionary() -> Dictionary:
    os.makedirs(CACHE, exist_ok=True)
    for ext in ("aff", "dic"):
        path = f"{DICT_BASE}.{ext}"
        if not os.path.exists(path):
            print(f"скачиваю словарь sr-Latn.{ext} …", file=sys.stderr)
            req = urllib.request.Request(DICT_URL + f"sr-Latn.{ext}", headers={"User-Agent": "ucimo-lint/1"})
            with urllib.request.urlopen(req, timeout=120) as r, open(path + ".part", "wb") as f:
                f.write(r.read())
            os.replace(path + ".part", path)
    return Dictionary.from_files(DICT_BASE)


@dataclass
class Config:
    allow: dict[str, str]        # токен -> причина
    allow_in: dict[str, str]     # префикс места (файл / файл:шаг упр.) -> причина
    yat_ok: dict[str, str]       # слово с ije/je, верное и в экавице -> причина
    diac_ok: dict[str, str]      # слово без диакритики, у которого есть «двойник» -> причина
    croatisms: dict[str, str]    # хорватизм -> сербский вариант


def load_config() -> Config:
    raw = yaml.safe_load(open(CONFIG, encoding="utf-8")) if os.path.exists(CONFIG) else {}
    raw = raw or {}
    cfg = Config(
        allow={k.lower(): v for k, v in (raw.get("allow") or {}).items()},
        allow_in=dict(raw.get("allow_in") or {}),
        yat_ok={k.lower(): v for k, v in (raw.get("yat_ok") or {}).items()},
        diac_ok={k.lower(): v for k, v in (raw.get("diac_ok") or {}).items()},
        croatisms={k.lower(): v for k, v in (raw.get("croatisms") or {}).items()},
    )
    for section in ("allow", "allow_in", "yat_ok", "diac_ok"):
        for tok, why in getattr(cfg, section).items():
            if not (why or "").strip():
                sys.exit(f"{CONFIG}: у «{tok}» ({section}) нет причины — исключение без причины недопустимо")
    return cfg


# ----------------------------------------------------------------- проверки --

class Checker:
    def __init__(self, d: Dictionary, cfg: Config):
        self.d, self.cfg = d, cfg
        self._cache: dict[str, bool] = {}

    def known(self, tok: str) -> bool:
        if tok in self._cache:
            return self._cache[tok]
        # Hunspell сам учитывает начало предложения (Kuća → kuća). Своих вариантов
        # регистра не пробуем: tok.title() превратил бы опечатку kuča в имя Kuča.
        ok = bool(self.d.lookup(tok))
        self._cache[tok] = ok
        return ok

    def spell_ok(self, token: str) -> bool:
        low = token.lower()
        if low in self.cfg.allow:
            return True
        parts = token.split("-")
        return all(p.lower() in self.cfg.allow or self.known(p) for p in parts)

    def yat(self, token: str) -> str | None:
        """Сообщение, если слово похоже на иекавизм/хорватизм, иначе None."""
        low = token.lower()
        if low in self.cfg.yat_ok:
            return None
        if low in self.cfg.croatisms:
            return f"хорватизм → {self.cfg.croatisms[low]}"
        if len(low) < 5:  # nije, sije, pije … — слишком коротко для эвристики
            return None
        # Только внутри слова: конечное -ije — это и сравнительная степень (ranije,
        # sporije), и «režije/serije»; настоящие иекавизмы на -ije/-je (dvije,
        # prije, ovdje, gdje) заведены явно в croatisms.
        body = low[:-3] if low.endswith("ije") else low[:-2] if low.endswith("je") else low
        variants = {body.replace("ije", "e") + low[len(body):]}
        # короткий рефлекс: согласная + je (mjesto → mesto), но не lj/nj и не начало слова
        variants.add(re.sub(r"(?<=[bcčćdđfghkmprsštvzž])je", "e", body) + low[len(body):])
        for v in variants:
            if v != low and len(v) >= 4 and self.known(low) and self.known(v):
                return f"иекавизм? экавица: {v}"
        return None


    def diac_twins(self, token: str) -> list[str]:
        """Слова-«двойники» токена с диакритикой (skoli → školi), если сам токен
        без диакритики. Пробуем до трёх замен c→č/ć, s→š, z→ž, dj→đ."""
        low = token.lower()
        if any(ch in low for ch in "čćšžđ"):
            return []
        spots = [(m.start(), m.group()) for m in re.finditer(r"dj|c|s|z", low)]
        out: set[str] = set()
        for r in range(1, min(len(spots), 3) + 1):
            for comb in itertools.combinations(spots, r):
                choices = [[(pos, len(g), {"dj": "đ", "c": "č", "s": "š", "z": "ž"}[g])]
                           + ([(pos, 1, "ć")] if g == "c" else []) for pos, g in comb]
                for pick in itertools.product(*choices):
                    t = low
                    for pos, ln, ch in sorted(pick, reverse=True):
                        t = t[:pos] + ch + t[pos + ln:]
                    if self.known(t):
                        out.add(t)
        return sorted(out)


def mixed_script_word(text: str) -> str | None:
    """Первое слово, где латиница вперемешку с кириллицей (омоглиф), иначе None."""
    for m in ANY_LETTERS_RE.finditer(unicodedata.normalize("NFC", text)):
        w = m.group(0)
        if CYR_RE.search(w) and LAT_RE.search(w):
            return w
    return None


@dataclass
class Item:
    kind: str     # "must" | "prose"
    loc: str
    text: str


def load_yaml(path: str):
    return yaml.safe_load(open(path, encoding="utf-8"))


def flat(x):
    if isinstance(x, str):
        yield x
    elif isinstance(x, list):
        for y in x:
            yield from flat(y)


def exercise_items(ex: dict, loc: str):
    t = ex.get("type")
    here = f"{loc} {ex.get('id', '?')}"
    must = []
    if t == "choice":
        must += [ex.get("answer")]
    elif t in ("translate", "fix_error", "fill_blank", "word_bank", "conjugate"):
        must += list(flat(ex.get("accept")))
    elif t == "listen":
        must += [ex.get("say")] + list(flat(ex.get("accept")))
    elif t == "match":
        must += list(flat(ex.get("pairs")))
    if t in ("fill_blank", "choice", "conjugate"):
        must.append(ex.get("prompt"))   # в промптах стоит сербский контекст ("Ja sam u ___.")
    for s in must:
        if isinstance(s, str) and s.strip():
            yield Item("must", here, s)
    if ex.get("explain"):
        yield Item("prose", here + " explain", ex["explain"])


def lesson_items(only: str | None):
    for path in sorted(glob.glob(os.path.join(CONTENT, "lessons", "*.yaml"))):
        name = os.path.basename(path)
        if name == "_TEMPLATE.yaml" or (only and name != f"{only}.yaml"):
            continue
        doc = load_yaml(path) or {}
        for step in doc.get("steps") or []:
            loc = f"lessons/{name}:{step.get('id', '?')}"
            for ex in step.get("exercises") or []:
                yield from exercise_items(ex, loc)
            for turn in step.get("turns") or []:
                if turn.get("sr"):
                    yield Item("must", f"{loc} реплика", turn["sr"])
                if turn.get("exercise"):
                    yield from exercise_items(turn["exercise"], loc)
            md = step.get("md")
            if not md:
                continue
            mdpath = os.path.join(CONTENT, "lessons", md)
            if not os.path.exists(mdpath):
                continue
            text = open(mdpath, encoding="utf-8").read()
            if step.get("kind") == "reading":
                head = re.split(r"^---\s*$", text, maxsplit=1, flags=re.M)[0]
                for i, line in enumerate(head.splitlines(), 1):
                    line = re.sub(r"^\s*[—–-]\s*", "", line).strip()
                    if line:
                        yield Item("must", f"lessons/{md}:{i}", line)
            else:
                cleaned = NEGATED_RE.sub(lambda m: " " * len(m.group(0)), text)
                for m in re.finditer(r"`([^`]+)`", cleaned):
                    span = m.group(1).strip()
                    if mixed_script_word(span):
                        # Омоглиф в слове — ошибка в любом код-спане, даже в том, который
                        # иначе пропускается как «не фраза» (в нём есть кириллица).
                        yield Item("must", f"lessons/{md}:{cleaned.count(chr(10), 0, m.start()) + 1}", span)
                        continue
                    if span.startswith("-") or span.endswith("-") or NOT_PHRASE_RE.search(span):
                        continue
                    if len(span) <= 2 or (len(TOKEN_RE.findall(span)) == 1 and len(span) <= 3 and "j" in span):
                        continue   # буква / звук / их сочетание (`c`, `lj`, `dž`), а не фраза
                    line = cleaned.count("\n", 0, m.start()) + 1
                    yield Item("must", f"lessons/{md}:{line}", span)


def reference_items():
    for e in load_yaml(os.path.join(CONTENT, "vocab.yaml")) or []:
        loc = f"vocab.yaml:{e.get('id', '?')}"
        for field in ("latin", "example_sr"):
            if e.get(field):
                yield Item("must", loc, e[field])
    for e in load_yaml(os.path.join(CONTENT, "false-friends.yaml")) or []:
        loc = f"false-friends.yaml:{e.get('id', '?')}"
        for field in ("sr", "correct"):
            if e.get(field):
                yield Item("must", loc, e[field])
    for e in load_yaml(os.path.join(CONTENT, "grammar.yaml")) or []:
        loc = f"grammar.yaml:{e.get('id', '?')}"
        if e.get("example_sr"):
            yield Item("must", loc, e["example_sr"])
        for it in e.get("items") or []:
            for a in flat(it.get("accept")):
                yield Item("must", loc, a)


@dataclass
class Finding:
    cat: str
    loc: str
    token: str
    msg: str
    context: str


def check_item(chk: Checker, item: Item) -> list[Finding]:
    out: list[Finding] = []
    text = unicodedata.normalize("NFC", item.text)
    for m in ANY_LETTERS_RE.finditer(text):
        w = m.group(0)
        if CYR_RE.search(w) and LAT_RE.search(w):
            out.append(Finding("SCRIPT", item.loc, w, "латиница и кириллица в одном слове", text))
    for m in TOKEN_RE.finditer(text):
        tok = m.group(0)
        if len(tok) < 2 and "-" not in tok:
            continue
        if len(tok) <= 3 and not re.search("[aeiour]", tok.lower()):
            continue   # lj / nj / dž / dj — буквосочетания, а не слова (у настоящих слов есть гласная или r)
        # сначала YAT: «tko» надо назвать хорватизмом, а не просто «нет в словаре»
        hits = [(p, chk.yat(p)) for p in tok.split("-")]
        hits = [(p, y) for p, y in hits if y]
        if hits:
            out += [Finding("YAT", item.loc, p, y, text) for p, y in hits]
            continue
        if not chk.spell_ok(tok):
            sug = ""
            try:
                sug = ", ".join(list(chk.d.suggest(tok))[:3])
            except Exception:
                pass
            out.append(Finding("SPELL", item.loc, tok, f"нет в словаре{' → ' + sug if sug else ''}", text))
            continue
        for part in tok.split("-"):
            if len(part) >= 3 and part.lower() not in chk.cfg.diac_ok and part.lower() not in chk.cfg.allow:
                twins = chk.diac_twins(part)
                if twins:
                    out.append(Finding("DIAC", item.loc, part,
                                       f"нет ли пропущенной диакритики? {', '.join(twins[:3])} "
                                       f"(верно как есть → diac_ok с причиной)", text))
    return out


def run(only: str | None, prose: bool) -> int:
    chk = Checker(ensure_dictionary(), load_config())
    findings: list[Finding] = []
    n_items = 0
    seen: set[tuple] = set()
    items = list(reference_items()) if not only else []
    items += list(lesson_items(only))
    for it in items:
        if it.kind == "prose" and not prose:
            continue
        n_items += 1
        for f in check_item(chk, it):
            if f.cat != "SCRIPT" and any(f.loc.startswith(pre) for pre in chk.cfg.allow_in):
                continue
            key = (f.cat, f.loc, f.token)
            if key not in seen:
                seen.add(key)
                findings.append(f)
    if not only:
        findings += vocab_cyrillic_findings()
    by_cat: dict[str, list[Finding]] = {}
    for f in findings:
        by_cat.setdefault(f.cat, []).append(f)
    for cat in ("SCRIPT", "CYR", "SPELL", "YAT", "DIAC"):
        rows = by_cat.get(cat, [])
        if not rows:
            continue
        print(f"\n== {cat}: {len(rows)}")
        for f in rows:
            ctx = f.context if len(f.context) <= 90 else f.context[:87] + "…"
            print(f"  {f.loc:<34} {f.token:<16} {f.msg}\n  {'':<34} └ {ctx}")
    print(f"\nпроверено фрагментов: {n_items}; находок: {len(findings)}")
    return 1 if findings else 0


# ----------------------------------------------------------------- самопроверка --

def self_test() -> int:
    chk = Checker(ensure_dictionary(), Config(allow={}, allow_in={}, yat_ok={}, diac_ok={}, croatisms={"tisuća": "hiljada", "kruh": "hleb", "tko": "ko"}))
    good = ["kuća", "kuću", "nije", "idemo", "Beograd", "Beogradu", "hvala", "dete", "mleko", "lepo", "vreme", "njegov"]
    bad_spell = ["kuča", "hvla", "razumen", "pozoriste", "tko"]
    bad_yat = ["lijepo", "vrijeme", "mlijeko", "dijete", "tisuća", "kruh", "mjesto", "pjesma", "svijet"]
    fails = []
    for w in good:
        if not chk.spell_ok(w) or chk.yat(w):
            fails.append(f"ложная тревога на «{w}»")
    for w in bad_spell:
        if chk.spell_ok(w):
            fails.append(f"SPELL пропустил «{w}»")
    for w in bad_yat:
        if not chk.yat(w):
            fails.append(f"YAT пропустил «{w}»")
    sk = check_item(chk, Item("must", "t", "Ana je u skoli."))
    if not any(f.cat == "DIAC" and f.token == "skoli" for f in sk):
        fails.append("DIAC пропустил skoli (школи без š)")
    if any(f.cat == "DIAC" for f in check_item(chk, Item("must", "t", "Ana je u školi."))):
        fails.append("DIAC ложно сработал на školi")
    if lat_to_cyr("Njegov džem, čokolada") != "Његов џем, чоколада":
        fails.append("lat_to_cyr: неверная транслитерация")
    if lat_to_cyr(re.sub(r"\s*\([^)]*\)", "", "morati (moram)")) != "морати":
        fails.append("lat_to_cyr: скобки не отброшены")
    sc = check_item(chk, Item("must", "t", "kuћa"))  # кириллическая ћ внутри латинского слова
    if not any(f.cat == "SCRIPT" for f in sc):
        fails.append("SCRIPT пропустил смешение алфавитов")
    for f in fails:
        print("FAIL:", f)
    print("self-test:", "OK" if not fails else f"{len(fails)} провал(ов)")
    return 1 if fails else 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--lesson", help="только урок NN (например 49)")
    ap.add_argument("--prose", action="store_true", help="проверять и Latin в `explain` (шумно)")
    ap.add_argument("--self-test", action="store_true", help="проверить сам линтер на эталонных словах")
    ap.add_argument("--content", help="другая папка content/ (для мутационных тестов)")
    a = ap.parse_args()
    if a.content:
        global CONTENT, CONFIG
        CONTENT = os.path.abspath(a.content)
        CONFIG = os.path.join(CONTENT, "lint-serbian.yaml")
    if a.self_test:
        return self_test()
    return run(a.lesson, a.prose)


if __name__ == "__main__":
    sys.exit(main())
