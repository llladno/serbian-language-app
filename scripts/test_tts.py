#!/usr/bin/env python3
"""Plain-assert tests for tts.py helpers (no pytest). Run: python3 scripts/test_tts.py"""
import sys

from tts import sr_lat_to_cyr

CASES = {
    "Zašto ne radiš danas?": "Зашто не радиш данас?",
    "Ja sam iz Rusije.": "Ја сам из Русије.",
    "Dobar dan, kako ste?": "Добар дан, како сте?",
    "Ko je poslednji?": "Ко је последњи?",
    "Možete li sporije, molim vas?": "Можете ли спорије, молим вас?",
    "Sto pedeset dinara.": "Сто педесет динара.",
    "Šta radiš danas?": "Шта радиш данас?",
    "Mi živimo u Novom Sadu.": "Ми живимо у Новом Саду.",
    # digraphs
    "Njegoš": "Његош",
    "ljudi": "људи",
    "džak": "џак",
    "LJUBAV": "ЉУБАВ",
}


def main() -> int:
    fails = 0
    for latin, want in CASES.items():
        got = sr_lat_to_cyr(latin)
        if got != want:
            fails += 1
            print(f"FAIL  {latin!r} -> {got!r}  (want {want!r})")
    if fails:
        print(f"\n{fails} failed")
        return 1
    print(f"ok — {len(CASES)} cases")
    return 0


if __name__ == "__main__":
    sys.exit(main())
