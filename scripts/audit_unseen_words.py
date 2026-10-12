#!/usr/bin/env python3
"""Строгий аудит «слова из заданий, которых ученик ещё не видел».

Для каждого упражнения манифест-уроков берёт сербские токены и проверяет,
встречался ли токен в ТЕКСТЕ курса раньше: в теории (teach-md), чтении
(reading-md), репликах диалогов предыдущих уроков и более ранних шагов этого
урока. Поле `lesson` в vocab.yaml за доказательство не считается — слово может
числиться за уроком, а в самом уроке не звучать.

Категории находок (по убыванию важности):
  LISTEN   — диктант: фраза, которую ученик слышит и пишет
  CONJ     — таблица спряжения: формы, которые ученик вводит
  ANSWER   — правильный ответ (accept/answer/bank), который ученик пишет/выбирает
  DLG-ME   — реплика героя в диалоге (ученик её произносит)
  MATCH    — слова в «соедини»
  OPTION   — неверные варианты choice (отвлекалки)
  PROMPT   — латиница в тексте задания
  DLG-NPC  — реплика собеседника (дана с переводом)

    python3 scripts/audit_unseen_words.py --strict
    python3 scripts/audit_unseen_words.py --strict --lesson 04
    python3 scripts/audit_unseen_words.py --strict --cat LISTEN,CONJ
"""
import argparse
import re
import sys
import unicodedata
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
CONTENT = ROOT / "content"

FUNCTION_WORDS = set("i a ali ili ne da li se u na o za iz do od s sa po bez".split())
TOKEN_RE = re.compile(r"[A-Za-zĆćČčĐđŠšŽž]+")
CATS = ["LISTEN", "CONJ", "ANSWER", "DLG-ME", "MATCH", "OPTION", "PROMPT", "DLG-NPC"]


def norm_tokens(text: str):
    return TOKEN_RE.findall(unicodedata.normalize("NFC", text).lower())


def fold(tok: str) -> str:
    return (tok.replace("č", "c").replace("ć", "c").replace("š", "s")
            .replace("ž", "z").replace("đ", "dj"))


def load_yaml(p: Path):
    return yaml.safe_load(p.read_text(encoding="utf-8"))


def flat(accept):
    out = []
    for a in accept or []:
        out.extend(a if isinstance(a, list) else [a])
    return out


def classified(e: dict):
    """-> [(category, string)] для одного упражнения."""
    t = e.get("type")
    out = []
    prompt = e.get("prompt") or ""
    out.append(("PROMPT", prompt))
    if t == "listen":
        out.append(("LISTEN", e.get("say") or ""))
        out += [("LISTEN", a) for a in flat(e.get("accept"))]
    elif t == "conjugate":
        out += [("CONJ", a) for a in flat(e.get("accept"))]
        out += [("CONJ", f) for f in e.get("forms") or []]
    elif t == "choice":
        ans = e.get("answer")
        for o in e.get("options") or []:
            out.append(("ANSWER" if o == ans else "OPTION", o))
    elif t == "match":
        for pr in e.get("pairs") or []:
            for side in pr:
                out.append(("MATCH", side))
    else:
        out += [("ANSWER", a) for a in flat(e.get("accept"))]
        if e.get("answer"):
            out.append(("ANSWER", e["answer"]))
        out += [("ANSWER", b) for b in e.get("bank") or []]
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--lesson")
    ap.add_argument("--strict", action="store_true",
                    help="не доверять allow-words (кроме имён): только то, что было в тексте")
    ap.add_argument("--cat", help="только эти категории, через запятую")
    ap.add_argument("--new-only", action="store_true",
                    help="скрыть токены, у которых есть увиденное слово с той же основой "
                         "(скорее всего форма разобранной модели, а не новое слово)")
    args = ap.parse_args()
    want = set(args.cat.split(",")) if args.cat else set(CATS)

    course = load_yaml(CONTENT / "course.yaml")
    order = [lid for ph in course["phases"] for lid in ph["lessons"]]
    allow = load_yaml(CONTENT / "allow-words.yaml")
    keep = {"names"} if args.strict else set(allow)
    allowed = set()
    for key, lst in allow.items():
        if key in keep:
            for w in lst:
                allowed.update(norm_tokens(str(w)))
    if args.strict:
        allowed |= FUNCTION_WORDS

    seen_exact: dict[str, str] = {}
    seen_fold: dict[str, str] = {}

    def see(tokens, where):
        for tok in tokens:
            seen_exact.setdefault(tok, where)
            seen_fold.setdefault(fold(tok), where)

    def unseen(text, also):
        res = []
        for tok in norm_tokens(text):
            if tok in allowed or tok in also or tok in res:
                continue
            if tok in seen_exact or fold(tok) in seen_fold:
                continue
            if args.new_only and has_stem(tok):
                continue
            res.append(tok)
        return res

    def has_stem(tok):
        n = max(3, min(4, len(tok) - 1))
        stem = fold(tok)[:n]
        return any(k.startswith(stem) for k in seen_fold)

    counts = {c: 0 for c in CATS}
    for lid in order:
        meta = course["lessons"].get(lid, {})
        f = meta.get("file", "")
        if not f.endswith(".yaml") or not (CONTENT / f).exists():
            continue
        man = load_yaml(CONTENT / f)
        rows = []
        for step in man.get("steps", []):
            also = set()
            for a in step.get("also_ok") or []:
                also.update(norm_tokens(str(a)))
            if step.get("md"):
                md = (CONTENT / "lessons" / step["md"]).read_text(encoding="utf-8")
                see(norm_tokens(md), f"{lid}/{step['id']}")
            if step["kind"] == "dialogue":
                for turn in step.get("turns", []):
                    sr = turn.get("sr") or ""
                    cat = "DLG-ME" if turn.get("who") == "me" else "DLG-NPC"
                    u = unseen(sr, also)
                    if u:
                        rows.append((step["id"], cat, sr, u))
                    see(norm_tokens(sr), f"{lid}/{step['id']}")
                    if turn.get("exercise"):
                        collect(turn["exercise"], unseen, also, rows)
            for e in step.get("exercises") or []:
                collect(e, unseen, also, rows)
        rows = [r for r in rows if r[1] in want]
        if rows and (not args.lesson or args.lesson == lid):
            print(f"\n=== урок {lid}: {meta.get('title')} ===")
            for sid, cat, text, toks in rows:
                print(f"  [{sid}] {cat:8} {', '.join(toks):28} ← {text!r}")
        for r in rows:
            if not args.lesson or args.lesson == lid:
                counts[r[1]] += 1
    print("\nИтого:", {k: v for k, v in counts.items() if k in want})


def collect(e, unseen, also, rows):
    seen_cat = set()
    for cat, text in classified(e):
        if not isinstance(text, str) or not text:
            continue
        u = unseen(text, also)
        key = (cat, tuple(u))
        if u and key not in seen_cat:
            seen_cat.add(key)
            rows.append((e["id"], cat, text, u))


if __name__ == "__main__":
    sys.exit(main())
