#!/usr/bin/env python3
"""Serbian-Latin -> Russian-Cyrillic phonetic transliteration helper.

Separates the *linguistic* judgment call (which syllable is stressed) from
the *mechanical* letter-mapping (error-prone to hand-type at scale). You
decide the stressed syllable by counting vowels left to right (1-based; a
lj/nj/j + vowel combo counts as one vowel); the script does the rest.

Usage:
    python3 scripts/translit_ru.py molim 1
        -> мо́лим
    python3 scripts/translit_ru.py "kako da dođem" 1 0 1
        -> ка́ко да до́дем   (0/None skips stress on that word, e.g. clitics)
    python3 scripts/translit_ru.py --self-test

Import from Python for batch use:
    from translit_ru import translit_word, translit_phrase_auto

See CLAUDE.md -> "Русская транскрипция слов" for the letter-mapping table,
the stress rules, and the full add-a-word/add-a-lesson checklist. Sanity-
check any batch of new transcriptions with scripts/check_transcriptions.py
before committing.
"""
import argparse
import re
import sys

IOT = {'a': 'я', 'e': 'е', 'i': 'и', 'o': 'ё', 'u': 'ю'}
PLAIN_VOWEL = {'a': 'а', 'e': 'е', 'i': 'и', 'o': 'о', 'u': 'у'}
CONS = {
    'b': 'б', 'v': 'в', 'g': 'г', 'd': 'д', 'z': 'з', 'k': 'к',
    'l': 'л', 'm': 'м', 'n': 'н', 'p': 'п', 's': 'с', 't': 'т', 'f': 'ф',
    'c': 'ц', 'h': 'х', 'š': 'ш', 'ž': 'ж', 'č': 'ч', 'ć': 'ч',
}
VOWELS = set('aeiou')
ACUTE = '́'


def translit_word(latin: str, stress_idx):
    """stress_idx: 1-based index of the vowel-event to stress, or None/0."""
    s = latin
    n = len(s)
    i = 0
    out = []
    vowel_positions = []
    event = 0

    while i < n:
        c = s[i]
        c2 = s[i:i + 2].lower()

        if c2 == 'lj':
            nxt = s[i + 2].lower() if i + 2 < n else ''
            if nxt in VOWELS:
                event += 1
                out.append('л')
                pos = len(out)
                out.append(IOT[nxt])
                vowel_positions.append((event, pos))
                i += 3
                continue
            out.append('ль')
            i += 2
            continue

        if c2 == 'nj':
            nxt = s[i + 2].lower() if i + 2 < n else ''
            if nxt in VOWELS:
                event += 1
                out.append('н')
                pos = len(out)
                out.append(IOT[nxt])
                vowel_positions.append((event, pos))
                i += 3
                continue
            out.append('нь')
            i += 2
            continue

        if c2 == 'dž':
            out.append('дж')
            i += 2
            continue

        lc = c.lower()

        if lc == 'đ':
            # đ collapses onto dž's "дж" -- standard practical transliteration
            # doesn't distinguish them (unlike a phonology lesson's prose might).
            out.append('дж')
            i += 1
            continue

        if lc == 'j':
            nxt = s[i + 1].lower() if i + 1 < n else ''
            if nxt in VOWELS:
                event += 1
                pos = len(out)
                out.append(IOT[nxt])
                vowel_positions.append((event, pos))
                i += 2
                continue
            out.append('й')
            i += 1
            continue

        if lc in VOWELS:
            event += 1
            pos = len(out)
            # word-initial or post-vowel bare 'e' -> э (no preceding consonant
            # to soften); after a consonant -> е. Only matters for 'e'.
            if lc == 'e' and (i == 0 or s[i - 1].lower() in VOWELS):
                out.append('э')
            else:
                out.append(PLAIN_VOWEL[lc])
            vowel_positions.append((event, pos))
            i += 1
            continue

        if lc in CONS:
            out.append(CONS[lc])
            i += 1
            continue

        if lc == 'r':
            prev = s[i - 1].lower() if i > 0 else ''
            nxt = s[i + 1].lower() if i + 1 < n else ''
            syllabic = prev not in VOWELS and nxt not in VOWELS
            out.append('р')
            if syllabic:
                event += 1  # nucleus, but there's no way to accent a bare consonant
            i += 1
            continue

        out.append(c)  # punctuation etc. passes through
        i += 1

    total_events = event
    if stress_idx and total_events > 1:
        for ev, pos in vowel_positions:
            if ev == stress_idx:
                out[pos] = out[pos] + ACUTE
                break
    return ''.join(out)


WORD_RE = re.compile(r"[A-Za-zČĆŽŠĐčćžšđ]+")


def translit_phrase_auto(text: str, stresses):
    """Auto-tokenizes on whitespace; each token's letter-run gets the next
    stress index from `stresses` (missing/0/None = no mark on that word),
    punctuation passes through unchanged."""
    stresses = list(stresses)
    si = 0
    out_tokens = []
    for tok in text.split(' '):
        m = WORD_RE.search(tok)
        if not m:
            out_tokens.append(tok)
            continue
        pre, core, post = tok[:m.start()], m.group(0), tok[m.end():]
        idx = stresses[si] if si < len(stresses) else None
        si += 1
        out_tokens.append(pre + translit_word(core, idx) + post)
    return ' '.join(out_tokens)


SELF_TESTS = [
    ('molim', 1, 'мо́лим'),
    ('hvala', 1, 'хва́ла'),
    ('izvolite', 2, 'изво́лите'),
    ('izvinite', 2, 'изви́ните'),
    ('nema', 1, 'не́ма'),
    ('čemu', 1, 'че́му'),
    ('ljubav', 1, 'лю́бав'),
    ('konj', None, 'конь'),
    ('doviđenja', 2, 'дови́дженя'),
    ('engleski', 1, 'э́нглески'),
    ('evo', 1, 'э́во'),
]


def self_test():
    ok = True
    for w, idx, expected in SELF_TESTS:
        got = translit_word(w, idx)
        status = 'OK' if got == expected else 'MISMATCH'
        if got != expected:
            ok = False
        print(f'{status}: {w} ({idx}) -> {got}  (expected {expected})')
    sys.exit(0 if ok else 1)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('--self-test', action='store_true', help='run the built-in regression tests and exit')
    ap.add_argument('phrase', nargs='?', help='Serbian Latin word or phrase (quote multi-word phrases)')
    ap.add_argument('stress', nargs='*', type=int, help='1-based stressed vowel index per word (0 = no stress)')
    args = ap.parse_args()

    if args.self_test:
        self_test()
    if not args.phrase:
        ap.print_help()
        sys.exit(1)
    print(translit_phrase_auto(args.phrase, args.stress))


if __name__ == '__main__':
    main()
