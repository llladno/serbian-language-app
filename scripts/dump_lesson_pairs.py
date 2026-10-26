#!/usr/bin/env python3
"""Print the Serbian-Russian pairs of lessons for a read-through, or find clashes.

Every place where a Serbian phrase is given a Russian meaning (listen-and-choose,
matching pairs, dictation, dialogue lines, glosses in the slides) is listed per
lesson, so a reviewer can read the whole lesson as the student sees it.

  python3 scripts/dump_lesson_pairs.py 06 06b 07     # these lessons, in the order given
  python3 scripts/dump_lesson_pairs.py                # every lesson of the course map
  python3 scripts/dump_lesson_pairs.py --conflicts    # the same Serbian text glossed differently

--conflicts is what catches "zdravo is «привет» in one lesson and «здравствуй» in another".
Some groups it prints are fine (a gloss with a note in brackets, «ruski» and «русский язык»);
read the list, do not expect it to be empty.
"""
import collections
import os
import re
import sys

import yaml

ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "content")


def norm(s):
    return re.sub(r"[^\wЀ-ӿ ]", "", s.lower()).strip()


def course_order():
    c = yaml.safe_load(open(os.path.join(ROOT, "course.yaml"), encoding="utf-8"))
    return [lid for ph in c["phases"] for lid in ph["lessons"]]


def pairs_of(lid):
    """Yield (kind, serbian, russian) for one lesson."""
    d = yaml.safe_load(open(os.path.join(ROOT, "lessons", f"{lid}.yaml"), encoding="utf-8"))
    for s in d["steps"]:
        for e in s.get("exercises") or []:
            t = e["type"]
            if t == "choice" and e.get("say"):
                yield "listen-choice", e["say"], e["answer"]
            elif t == "match":
                for a, b in e["pairs"]:
                    yield "match", a, b
            elif t == "listen":
                for r in e["ru"]:
                    yield "dictation", e["say"], r
        for t in s.get("turns") or []:
            yield "dialogue", t["sr"], t["ru"]
        if s["kind"] == "teach" and s.get("md"):
            txt = open(os.path.join(ROOT, "lessons", s["md"]), encoding="utf-8").read()
            for m in re.finditer(r"`([^`]+)`\s*\[[^\]]*\]\s*\*\(([^)]*)\)\*", txt):
                gloss = m.group(2)
                yield "slide", m.group(1), gloss.split("—", 1)[1] if "—" in gloss else gloss


def main():
    args = [a for a in sys.argv[1:] if a != "--conflicts"]
    if "--conflicts" in sys.argv[1:]:
        seen = collections.defaultdict(lambda: collections.defaultdict(set))
        for lid in args or course_order():
            for _, sr, ru in pairs_of(lid):
                if norm(sr) and norm(ru):
                    seen[norm(sr)][norm(ru)].add(lid)
        n = 0
        for sr, glosses in sorted(seen.items()):
            if len(glosses) > 1:
                print(sr, "=>", {k: sorted(v) for k, v in glosses.items()})
                n += 1
        print(f"{n} groups")
        return
    for lid in args or course_order():
        print("=====", lid)
        for kind, sr, ru in pairs_of(lid):
            print(f"  {kind:13} {sr}  =  {ru}")


if __name__ == "__main__":
    main()
