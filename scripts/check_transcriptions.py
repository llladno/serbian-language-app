#!/usr/bin/env python3
"""Sanity-check every Russian [transcription] in the course, repo-wide.

Run this after adding/editing any word (content/vocab.yaml,
content/false-friends.yaml) or any teach-step markdown fragment that
carries a `word [транскрипция]` annotation. It never
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
  5. missing                -- a vocab.yaml/false-friends.yaml entry with no
     `transcription:` at all.
  6. engine-mismatch        -- the transcription is not what scripts/translit_ru.py
     produces for its Latin source (with the stress marks it carries). This is
     what enforces the letter rules everywhere: e -> э, đ -> дьжь, ji -> йи,
     lj/nj + vowel, и т.д. (engine-unverifiable = the pair can't be aligned
     with its source, or an accent sits where the engine never puts one.)
  7. stressed-clitic        -- an accent on je/se/mi/da/li/ću... inside a phrase.

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

from translit_ru import translit_word, WORD_RE

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VOCAB_PATH = os.path.join(ROOT, 'content', 'vocab.yaml')
FALSE_FRIENDS_PATH = os.path.join(ROOT, 'content', 'false-friends.yaml')
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


CYR_RUN = re.compile(r"[а-яёА-ЯЁ́]+")
# Unstressed little words that must carry no accent inside a phrase. Almost all
# are one syllable (the engine never accents those anyway); the multi-syllable
# future clitics are the ones this list actually guards.
CLITICS = {'je', 'su', 'sam', 'si', 'smo', 'ste', 'se', 'me', 'te', 'ga', 'ih',
           'mi', 'ti', 'mu', 'joj', 'nam', 'vam', 'im', 'li', 'da', 'bi',
           'ću', 'ćeš', 'će', 'ćemo', 'ćete'}


def _key(latin):
    """Compare Latin sources across places ignoring case, punctuation and
    spacing, so "žao mi je" (vocab) and "Žao mi je." (a lesson) are the same."""
    return ' '.join(re.sub(r'[^\w\s]', ' ', latin.lower()).split())


def _tkey(tr):
    """Compare transcriptions by letters and stress only -- not by trailing
    punctuation, slash spacing, parentheses or a leading hyphen ("-ти" vs "ти")."""
    return ' '.join(re.sub(r'[^\w\s\u0301]', ' ', tr.lower()).split())


def _unwrap(text):
    """Undo markdown line wrapping: collapse whitespace runs and the "> " that
    a blockquote adds at the start of each wrapped line."""
    return re.sub(r'\s+(?:>\s+)?', ' ', text).strip()


def _acute_index(tok):
    """1-based index of the accented vowel (counting Russian vowels), 'R' when
    the accent sits on a syllabic р, None when there is no accent."""
    n = 0
    for i, c in enumerate(tok):
        nxt_acute = i + 1 < len(tok) and tok[i + 1] == ACUTE
        if c.lower() in VOWELS:
            n += 1
            if nxt_acute:
                return n
        elif c.lower() == 'р' and nxt_acute:
            return 'R'
    return None


def expected_transcription(latin, stored):
    """Rebuild `stored` from `latin` with scripts/translit_ru.py, keeping the
    stress positions `stored` already carries (the engine can't know them).
    Returns (expected, error); error is a short reason when `stored` can't be
    aligned with `latin` token by token."""
    latin = re.sub(r'\s*\n>\s*', ' ', latin).replace('\n', ' ')
    lt, ot = latin.split(' '), stored.split(' ')
    if len(lt) != len(ot):
        return None, 'token count differs'
    out = []
    for l, o in zip(lt, ot):
        m = WORD_RE.search(l)
        if not m:
            out.append(o)
            continue
        core, post = m.group(0), l[m.end():]
        if WORD_RE.search(post):  # "slobodan/slobodna": split on the slash
            if '/' in l and '/' in o:
                sub, err = expected_transcription(l.replace('/', ' / '), o.replace('/', ' / '))
                if err:
                    return None, err
                out.append(sub.replace(' / ', '/'))
                continue
            return None, 'several words in one token'
        mo = CYR_RUN.search(o)
        if not mo:
            return None, 'no Russian letters under ' + l
        ocore, opre, opost = mo.group(0), o[:mo.start()], o[mo.end():]
        oi = _acute_index(ocore)
        idx = None
        if oi is not None:
            for k in range(1, 15):
                if _acute_index(translit_word(core, k)) == oi:
                    idx = k
                    break
            else:
                return None, f'stress on {ocore!r} is where the engine puts none (monosyllable?)'
        new = translit_word(core, idx)
        if ocore[0].isupper():
            new = new[0].upper() + new[1:]
        out.append(opre + new + opost)
    return ' '.join(out), None


def check_engine(latin, stored, loc, out):
    """Every transcription must be exactly what the engine produces for its
    Latin source (given its stress marks): this is what keeps the letter rules
    (e -> э, đ -> дьжь, ji -> йи, ...) applied everywhere, not just in new text."""
    exp, err = expected_transcription(latin, stored)
    if err:
        out['engine-unverifiable'].append((latin, stored, err, loc))
    elif exp != stored:
        out['engine-mismatch'].append((latin, stored, 'engine gives: ' + exp, loc))


def check_clitics(latin, stored, loc, out):
    lt, ot = latin.split(' '), stored.split(' ')
    if len(lt) != len(ot) or len(lt) < 2:
        return
    for l, o in zip(lt, ot):
        m = WORD_RE.search(l)
        if m and m.group(0).lower() in CLITICS and ACUTE in o:
            out['stressed-clitic'].append((latin, stored, loc))
            return


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
        check_engine(d['latin'], tr, f"vocab.yaml:{d['id']}", out)
        check_clitics(d['latin'], tr, f"vocab.yaml:{d['id']}", out)
        canon[_key(d['latin'])] = tr

    with open(FALSE_FRIENDS_PATH, encoding='utf-8') as f:
        false_friends = yaml.safe_load(f)

    for d in false_friends:
        tr = d.get('transcription')
        if not tr:
            out['missing'].append((d['id'], FALSE_FRIENDS_PATH))
            continue
        check_stress(tr, f"false-friends.yaml:{d['id']}", out)
        check_stray_chars(tr, f"false-friends.yaml:{d['id']}", out)
        check_engine(d['sr'], tr, f"false-friends.yaml:{d['id']}", out)
        check_clitics(d['sr'], tr, f"false-friends.yaml:{d['id']}", out)

    md_files = sorted(glob.glob(LESSON_MD_GLOB))
    occurrences = defaultdict(set)
    # a pair may be split across lines inside a blockquote ("`x`\n> [tr]")
    pair_re = re.compile(r'`([^`]+)`\s*(?:>\s*)?\[([^\]]+)\]')

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
                latin, tr = _unwrap(m.group(1)), _unwrap(m.group(2))
                occurrences[_key(latin)].add((tr, rel))
                if re.search(r'[а-яё]', tr.lower()):
                    check_engine(latin, tr, rel, out)
                    check_clitics(latin, tr, rel, out)

    for latin, trs in occurrences.items():
        uniq = {t for t, _ in trs}
        if len({_tkey(t) for t in uniq}) > 1:
            out['inconsistent'].append((latin, sorted(trs)))
        if latin in canon and _tkey(canon[latin]) not in {_tkey(t) for t in uniq}:
            for tr, rel in trs:
                if _tkey(tr) != _tkey(canon[latin]):
                    out['vocab-mismatch'].append((latin, tr, rel, canon[latin]))

    # dedupe within each category
    total = 0
    order = ['missing', 'final-syllable-stress', 'multi-stress', 'stray-serbian-letters',
             'engine-mismatch', 'engine-unverifiable', 'stressed-clitic',
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
            print('OK -- no issues found across', len(vocab), 'vocab entries,',
                  len(false_friends), 'false-friends entries, and', len(md_files), 'lesson files.')
        sys.exit(0)
    print(f'\n{total} issue(s) found.')
    sys.exit(1)


if __name__ == '__main__':
    main()
