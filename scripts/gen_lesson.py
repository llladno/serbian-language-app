#!/usr/bin/env python3
"""Lesson generator for the short-lesson format (levels 1 and 2, see docs/LESSON_AUTHORING.md).

Reads a spec (YAML) and writes content/lessons/<id>.yaml plus the slide fragments
content/lessons/<id>/<n>-<name>.md in the same style as the hand-written pilot lessons.

Spec:
  id, title, subtitle, start (first step number), script (optional), teaches: [vocab ids]
  Regenerating a lesson overwrites its manifest and fragments, so for a lesson that has a spec
  edit the spec, not the YAML. Usage: python3 scripts/gen_lesson.py spec.yaml ...
  python3 scripts/gen_lesson.py --check spec.yaml ...   (writes nothing, exits 1 if a lesson drifted)
  steps: list of
    - slide: {name, title, md, check: {prompt, options, answer, hint?, explain?, title?}}
    - practice: {title, also_ok?, mixed?, kind? (practice | checkpoint), ex: [ {type, ...} ]}
    - reading: {name, title, text, ex?: [...]}
    - dialogue: {title, scene, voice, also_ok?, turns: [{who, sr, ru, ex?}]}
"""
import json
import os
import sys

import yaml

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "content")
LISTEN_PROMPT = "Послушай и выбери перевод."


def q(s):
    return json.dumps(s, ensure_ascii=False)


def ql(lst):
    return "[" + ", ".join(q(x) for x in lst) + "]"


def emit_ex(out, ind, eid, e):
    sp = " " * ind
    t = e["type"]
    out.append(f'{sp}- id: {q(eid)}')
    if e.get("hint"):
        out.append(f'{sp}  hint: {q(e["hint"])}')
    out.append(f"{sp}  type: {t}")
    if e.get("say"):
        out.append(f'{sp}  say: {q(e["say"])}')
    prompt = e.get("prompt")
    if prompt is None and t == "choice" and e.get("say"):
        prompt = LISTEN_PROMPT
    if prompt is not None:
        out.append(f"{sp}  prompt: {q(prompt)}")
    if t == "choice":
        out.append(f'{sp}  options: {ql(e["options"])}')
        out.append(f'{sp}  answer: {q(e["answer"])}')
    elif t == "match":
        out.append(f"{sp}  pairs:")
        for a, b in e["pairs"]:
            out.append(f"{sp}    - {ql([a, b])}")
    elif t == "word_bank":
        out.append(f'{sp}  bank: {ql(e["bank"])}')
        out.append(f'{sp}  accept: {ql(e["accept"])}')
    elif t in ("fill_blank", "translate", "fix_error"):
        out.append(f'{sp}  accept: {ql(e["accept"])}')
    elif t == "listen":
        out.append(f'{sp}  accept: {ql(e["accept"])}')
        out.append(f'{sp}  ru: {ql(e["ru"])}')
    else:
        raise SystemExit(f"unknown exercise type {t} in {eid}")
    if e.get("explain"):
        out.append(f'{sp}  explain: {q(e["explain"])}')


def emit_exercises(out, sid, exs):
    out.append("    exercises:")
    for k, e in enumerate(exs, 1):
        emit_ex(out, 6, f"{sid}.{k}", e)


def build(spec, write=True):
    lid = str(spec["id"])
    n = int(spec.get("start", 1))
    out = [
        f"lesson: {q(lid)}",
        f'title: {q(spec["title"])}',
        f'subtitle: {q(spec["subtitle"])}',
    ]
    if spec.get("script"):
        out.append(f'script: {spec["script"]}   # teaches the letters themselves: stays Latin whatever script the learner reads in')
    teaches = spec.get("teaches", [])
    out.append("teaches: [" + ", ".join(teaches) + "]")
    out.append("")
    out.append("steps:")
    files = {}
    ids = []

    def head(sid, kind, title, extra=None):
        out.append(f'  - id: {q(sid)}')
        out.append(f"    kind: {kind}")
        out.append(f"    title: {q(title)}")
        for line in extra or []:
            out.append(line)

    for item in spec["steps"]:
        if "slide" in item:
            s = item["slide"]
            sid = f"{lid}.{n}"
            rel = f'{lid}/{n}-{s["name"]}.md'
            files[rel] = s["md"].strip() + "\n"
            head(sid, "teach", s["title"], [f"    md: {q(rel)}"])
            out.append("")
            c = s["check"]
            pid = f"{lid}.{n + 1}"
            extra = [f'    also_ok: {ql(c["also_ok"])}'] if c.get("also_ok") else []
            head(pid, "practice", c.get("title", f'Запомним: {s["title"]}'), extra)
            emit_exercises(out, pid, [dict(type="choice", **{k: v for k, v in c.items() if k not in ("title", "also_ok")})])
            out.append("")
            ids += [sid, pid]
            n += 2
        elif "practice" in item:
            p = item["practice"]
            sid = f"{lid}.{n}"
            extra = []
            if p.get("also_ok"):
                extra.append(f'    also_ok: {ql(p["also_ok"])}')
            if p.get("mixed"):
                extra.append("    mixed: true")
            head(sid, p.get("kind", "practice"), p["title"], extra)
            emit_exercises(out, sid, p["ex"])
            out.append("")
            ids.append(sid)
            n += 1
        elif "reading" in item:
            r = item["reading"]
            sid = f"{lid}.{n}"
            rel = f'{lid}/{n}-{r["name"]}.md'
            files[rel] = r["text"].strip() + "\n"
            head(sid, "reading", r["title"], [f"    md: {q(rel)}"])
            if r.get("ex"):
                emit_exercises(out, sid, r["ex"])
            out.append("")
            ids.append(sid)
            n += 1
        elif "dialogue" in item:
            d = item["dialogue"]
            sid = f"{lid}.{n}"
            extra = [f'    scene: {q(d["scene"])}', f'    voice: {d.get("voice", "f")}']
            if d.get("also_ok"):
                extra.append(f'    also_ok: {ql(d["also_ok"])}')
            head(sid, "dialogue", d["title"], extra)
            out.append("    turns:")
            k = 0
            for t in d["turns"]:
                out.append(f'      - who: {t["who"]}')
                out.append(f'        sr: {q(t["sr"])}')
                out.append(f'        ru: {q(t["ru"])}')
                if t["who"] == "me":
                    k += 1
                    out.append("        exercise:")
                    emit_ex(out, 10, f"{sid}.{k}", t["ex"])
                    # emit_ex starts with "- id", the dialogue expects a mapping: fix below
                out.append("")
            ids.append(sid)
            n += 1
        else:
            raise SystemExit(f"unknown step in {lid}: {list(item)}")

    text = "\n".join(out).rstrip() + "\n"
    # a dialogue exercise is a mapping, not a list item: turn "- id:" into "id:" under `exercise:`
    fixed, inside = [], False
    for line in text.split("\n"):
        if line.strip() == "exercise:":
            inside = True
            fixed.append(line)
            continue
        if inside:
            if line.startswith("          - id:"):
                fixed.append("          id:" + line[len("          - id:"):])
                continue
            if line.startswith("            "):
                fixed.append(line[2:])
                continue
            if line.strip() == "":
                inside = False
        fixed.append(line)
    text = "\n".join(fixed)
    if write:
        with open(os.path.join(ROOT, "lessons", f"{lid}.yaml"), "w", encoding="utf-8") as f:
            f.write(text)
        for rel, body in files.items():
            p = os.path.join(ROOT, "lessons", rel)
            os.makedirs(os.path.dirname(p), exist_ok=True)
            with open(p, "w", encoding="utf-8") as f:
                f.write(body)
    return text, files


if __name__ == "__main__":
    args = [a for a in sys.argv[1:] if a != "--check"]
    check = "--check" in sys.argv[1:]
    drift = []
    for path in args:
        spec = yaml.safe_load(open(path, encoding="utf-8"))
        text, files = build(spec, write=not check)
        # self-check: the generated manifest must parse
        doc = yaml.safe_load(text)
        if check:
            # --check writes nothing: it reports lessons whose files no longer match the spec
            lid = str(spec["id"])
            cur = os.path.join(ROOT, "lessons", f"{lid}.yaml")
            same = os.path.exists(cur) and open(cur, encoding="utf-8").read() == text
            for rel, body in files.items():
                p = os.path.join(ROOT, "lessons", rel)
                same = same and os.path.exists(p) and open(p, encoding="utf-8").read() == body
            print(f'{lid}: {"in sync" if same else "DIFFERS from the spec"}')
            if not same:
                drift.append(lid)
            continue
        nex = 0
        for s in doc["steps"]:
            nex += len(s.get("exercises") or [])
            nex += sum(1 for t in (s.get("turns") or []) if t.get("exercise"))
        print(f'{spec["id"]}: {len(doc["steps"])} steps, {nex} exercises, {len(files)} fragments')
    if drift:
        sys.exit(1)
