#!/usr/bin/env python3
"""Sanity-check every Russian [transcription] in the course, repo-wide.

Run this after adding/editing any word (content/vocab.yaml) or any teach-step
markdown fragment that carries a `word [транскрипция]` annotation. It never
knows if a stress is *correct* (no native-speaker dictionary is consulted) --
it only catches mechanical mistakes that are unambiguously wrong:

  1. final-syllable-stress -- Serbian never stresses a word's last syllable.
  2. stray-serbian-letters  -- a Serbian-only Cyrillic/Latin-diacritic char
     (ј, њ, љ, ђ, ћ, џ, č, ć, ž, š, đ, ...) leaked into what must be a pure
     Russian-alphabet transcription.
  3. inconsistent           -- the same Serbian word/phrase got a different
     transcription in two places (vocab.yaml vs. a lesson, or two lessons).
  4. unbalanced markdown    -- mismatched `` ` `` / `[` `]` / `(` `)` inside
     a paragraph (usually a forgotten closing bracket).
  5. missing                -- a vocab.yaml entry with no `transcription:` at
     all.

Usage:
    python3 scripts/check_transcriptions.py            # scan everything
    python3 scripts/check_transcriptions.py --quiet     # only print if issues found

Exit code is 1 if any issue was found (0 otherwise), so this can gate a
review the same way `go test ./server/internal/content/...` does.

See CLAUDE.md -> "Русская транскрипция слов" for the conventions this
enforces, and scripts/translit_ru.py to generate new transcriptions.
"""
import argparse
import glob
import os
import re
import sys
from collections import defaultdict

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VOCAB_PATH = os.path.join(ROOT, 'content', 'vocab.yaml')
LESSON_MD_GLOB = os.path.join(ROOT, 'content', 'lessons', '*', '*.md')

VOWELS = set('аеёиоуыэюя')
ACUTE = '́'
SERBIAN_ONLY = set('јљњђћџЈЉЊЂЋЏ')
LATIN_DIACRITICS = set('čćžšđČĆŽŠĐ')


def analyze_word(w):
    vowel_idxs, stressed_idxs = [], []
    i, n = 0, len(w)
    while i < n:
        c = w[i].lower()
        if c in VOWELS:
            vowel_idxs.append(i)
            if i + 1 < n and w[i + 1] == ACUTE:
                stressed_idxs.append(i)
        i += 1
    if not vowel_idxs:
        return None
    return {
        'n_vowels': len(vowel_idxs),
        'last_stressed': vowel_idxs[-1] in stressed_idxs,
        'n_stressed': len(stressed_idxs),
    }


def check_stress(text, loc, out):
    for tok in re.findall(r"[а-яёА-ЯЁ́]+", text):
        info = analyze_word(tok)
        if not info:
            continue
        clean = tok.replace(ACUTE, '')
        if info['n_vowels'] > 1 and info['last_stressed']:
            out['final-syllable-stress'].append((clean, loc))
        if info['n_stressed'] > 1:
            out['multi-stress'].append((clean, loc))


def check_stray_chars(text, loc, out):
    bad = set(ch for ch in text if ch in SERBIAN_ONLY or ch in LATIN_DIACRITICS)
    if bad:
        out['stray-serbian-letters'].append((text.strip(), loc, ''.join(sorted(bad))))


def check_balance(md_path, out):
    text = open(md_path, encoding='utf-8').read()
    rel = os.path.relpath(md_path, ROOT)
    for para in text.split('\n\n'):
        if para.strip().startswith('```'):
            continue
        if para.count('`') % 2 != 0:
            out['unbalanced-markdown'].append((f'odd backticks: {para[:70]!r}', rel))
        if para.count('[') != para.count(']'):
            out['unbalanced-markdown'].append((f'unmatched []: {para[:70]!r}', rel))
        if para.count('(') != para.count(')'):
            out['unbalanced-markdown'].append((f'unmatched (): {para[:70]!r}', rel))


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('--quiet', action='store_true', help='only print output when issues are found')
    args = ap.parse_args()

    out = defaultdict(list)

    with open(VOCAB_PATH, encoding='utf-8') as f:
        vocab = yaml.safe_load(f)

    canon = {}
    for d in vocab:
        tr = d.get('transcription')
        if not tr:
            out['missing'].append((d['id'], VOCAB_PATH))
            continue
        check_stress(tr, f"vocab.yaml:{d['id']}", out)
        check_stray_chars(tr, f"vocab.yaml:{d['id']}", out)
        canon[d['latin'].lower()] = tr

    md_files = sorted(glob.glob(LESSON_MD_GLOB))
    occurrences = defaultdict(set)
    pair_re = re.compile(r'`([^`]+)`\s*\[([^\]]+)\]')

    for path in md_files:
        rel = os.path.relpath(path, ROOT)
        check_balance(path, out)
        text = open(path, encoding='utf-8').read()
        for para in text.split('\n\n'):
            flat = para.replace('\n', ' ')
            for m in re.finditer(r'\[([^\]]*)\]', flat):
                check_stress(m.group(1), rel, out)
                check_stray_chars(m.group(1), rel, out)
            for m in pair_re.finditer(flat):
                latin, tr = m.group(1).strip(), m.group(2).strip()
                occurrences[latin.lower()].add((tr, rel))

    for latin, trs in occurrences.items():
        uniq = {t for t, _ in trs}
        if len(uniq) > 1:
            # ignore pure sentence-initial-capitalization differences
            if len({t[:1].lower() + t[1:] for t in uniq}) > 1:
                out['inconsistent'].append((latin, sorted(trs)))
        if latin in canon and canon[latin] not in uniq and len(trs) :
            for tr, rel in trs:
                if tr != canon[latin] and tr[:1].lower() + tr[1:] != canon[latin][:1].lower() + canon[latin][1:]:
                    out['vocab-mismatch'].append((latin, tr, rel, canon[latin]))

    # dedupe within each category
    total = 0
    order = ['missing', 'final-syllable-stress', 'multi-stress', 'stray-serbian-letters',
             'inconsistent', 'vocab-mismatch', 'unbalanced-markdown']
    for kind in order:
        items = out[kind]
        if not items:
            continue
        seen = []
        for item in items:
            if item not in seen:
                seen.append(item)
        total += len(seen)
        print(f'\n=== {kind}: {len(seen)} ===')
        for item in seen:
            print(' ', item)

    if total == 0:
        if not args.quiet:
            print('OK -- no issues found across', len(vocab), 'vocab entries and', len(md_files), 'lesson files.')
        sys.exit(0)
    print(f'\n{total} issue(s) found.')
    sys.exit(1)


if __name__ == '__main__':
    main()
