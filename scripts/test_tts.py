#!/usr/bin/env python3
"""Plain-assert tests for tts.py helpers (no pytest). Run: python3 scripts/test_tts.py"""
import sys

from tts import sr_lat_to_cyr, phrase_key, phrase_speech, md_phrases, prompt_phrases, fill_sentence, exercise_phrases

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


def check(label, got, want):
    if got != want:
        print(f"FAIL  {label}: got {got!r}, want {want!r}")
        return 1
    return 0


def phrase_tests() -> int:
    f = 0
    # key is case/punctuation/whitespace-insensitive; web/src/lib/phraseAudio.ts must agree
    f += check("key norm", phrase_key("Govorim  ruski!"), phrase_key("govorim ruski"))
    f += check("key vector", phrase_key("Govorim ruski."), "adc562fc95c4145a")
    f += check("key vector diacritics", phrase_key("Šta radiš?"), "0f8a5cb231c832f4")
    f += check("speech", phrase_speech("ću/ćeš/će…"), "ћу, ћеш, ће")
    f += check("speech paren", phrase_speech("radim (radi)"), "радим")
    md = "- `Govorim ruski.` [x] *(y)*\n`-im` и `sam + причастие`, `č`, `avion`, `a\nb`, `1.250`"
    f += check("md", md_phrases(md), ["Govorim ruski.", "avion"])
    f += check("prompt", prompt_phrases("«Ona radi ___ lekar.» и «Govorim ruski.» и «Привет»"), ["Govorim ruski."])
    for prompt, word, want in FILL_CASES:
        f += check(f"fill {prompt!r}", fill_sentence(prompt, word), want)
    f += check("choice voices only the answer", exercise_phrases(
        {"type": "choice", "prompt": "Входя, спрашиваешь —", "options": ["Dobar dan!", "Prijatno.", "Ko je poslednji?"], "answer": "Ko je poslednji?"}),
        ["Ko je poslednji?"])
    f += check("listen has no phrase clip", exercise_phrases({"type": "listen", "say": "Odakle si?", "accept": ["Odakle si?"]}), [])
    f += check("word_bank chips not voiced", exercise_phrases({"type": "word_bank", "prompt": "Собери", "bank": ["Govoriš", "srpski"], "accept": ["Govoriš srpski?"]}), ["Govoriš srpski?"])
    return f


FILL_CASES = [
    ("«Я за вами» → Ja sam ____ vama.", "iza", "Ja sam iza vama."),
    ("Утро, заходишь в пекару: «___ jutro!»", "Dobro", "Dobro jutro!"),
    ("Ti ___ student? (ты)", "si", "Ti si student?"),
    ("«Горло» → ___", "grlo", "grlo"),
    ("«полдень» по-сербски — ___", "podne", "podne"),
    ("У меня есть сестра: «Imam ___.»", "sestru", "Imam sestru."),
    ("Привет ___ и ___", "x", None),
]


def main() -> int:
    fails = phrase_tests()
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
