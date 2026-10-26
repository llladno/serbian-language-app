# Очередь ударений: слова из теории (уровень B)

Сгенерировано `python3 scripts/check_stress_tokens.py --write-queue`. Здесь слова,
у которых Wiktionary даёт **восходящее** ударение, а курс ставит другой слог.
Сербская и хорватская норма в таких словах расходятся, поэтому это очередь
для носителя, а не список ошибок. Нисходящие расхождения (уровень A) исправлены
и не допускаются: `make lint-sr` падает.

| слово | слог в курсе | слог по Wiktionary | где встречается |
|---|---|---|---|
| `aerodrom` | 1 | 3 (rising) | 2 |
| `alergičan` | 3 | 2 (rising) | 3 |
| `apoteka` | 2 | 3 (rising) | 6 |
| `autobus` | 2 | 3 (rising) | 11 |
| `bicikl` | 1 | 2 (rising) | 2 |
| `decembar` | 2 | 1 (rising) | 2 |
| `devedeset` | 1 | 3 (rising) | 1 |
| `devetnaest` | 3 | 2 (rising) | 1 |
| `direktno` | 2 | 1 (rising) | 1 |
| `doručak` | 2 | 1 (rising) | 2 |
| `doručkovati` | 2 | 1 (rising) | 3 |
| `doviđenja` | 2 | 3 (rising) | 2 |
| `dvadeset` | 2 | 1 (rising) | 13 |
| `dvanaest` | 2 | 1 (rising) | 2 |
| `glavobolja` | 3 | 2 (rising) | 2 |
| `iskustvo` | 1 | 2 (rising) | 4 |
| `jedanaest` | 3 | 2 (rising) | 3 |
| `kirija` | 2 | 1 (rising) | 5 |
| `koliko` | 1 | 2 (rising) | 23 |
| `kupovati` | 1 | 2 (rising) | 2 |
| `lubenica` | 1 | 2 (rising) | 1 |
| `minuta` | 1 | 2 (rising) | 8 |
| `naravno` | 2 | 1 (rising) | 2 |
| `novembar` | 2 | 1 (rising) | 2 |
| `odgovarati` | 2 | 3 (rising) | 3 |
| `odvojeno` | 2 | 1 (rising) | 1 |
| `oktobar` | 2 | 1 (rising) | 2 |
| `osamdeset` | 1 | 3 (rising) | 1 |
| `osamnaest` | 3 | 2 (rising) | 1 |
| `osećati` | 2 | 1 (rising) | 2 |
| `otvoreno` | 2 | 1 (rising) | 3 |
| `pedeset` | 1 | 2 (rising) | 4 |
| `petnaest` | 2 | 1 (rising) | 6 |
| `pijaca` | 2 | 1 (rising) | 3 |
| `pomoći` | 1 | 2 (rising) | 3 |
| `ponekad` | 2 | 1 (rising) | 3 |
| `porodica` | 2 | 1 (rising) | 3 |
| `potvrditi` | 1 | 2 (rising) | 2 |
| `pozorište` | 2 | 1 (rising) | 3 |
| `počinjati` | 2 | 1 (rising) | 1 |
| `prestonica` | 2 | 1 (rising) | 1 |
| `priroda` | 2 | 1 (rising) | 2 |
| `prizemlje` | 2 | 1 (rising) | 3 |
| `proleće` | 2 | 1 (rising) | 4 |
| `putovati` | 1 | 2 (rising) | 2 |
| `razumeti` | 3 | 2 (rising) | 3 |
| `rečenica` | 1 | 2 (rising) | 1 |
| `samoposluga` | 1 | 3 (rising) | 2 |
| `sastanak` | 2 | 1 (rising) | 2 |
| `sedamdeset` | 1 | 3 (rising) | 1 |
| `sedamnaest` | 3 | 2 (rising) | 1 |
| `septembar` | 2 | 1 (rising) | 2 |
| `sniženje` | 1 | 2 (rising) | 2 |
| `stanovati` | 1 | 2 (rising) | 2 |
| `subota` | 2 | 1 (rising) | 3 |
| `sudopera` | 2 | 1 (rising) | 2 |
| `terasa` | 1 | 2 (rising) | 5 |
| `testenina` | 1 | 3 (rising) | 2 |
| `trinaest` | 2 | 1 (rising) | 1 |
| `tuširati` | 1 | 2 (rising) | 1 |
| `utorak` | 2 | 1 (rising) | 4 |
| `visoko` | 1 | 2 (rising) | 2 |
| `četiri` | 2 | 1 (rising) | 5 |
| `četrdeset` | 1 | 3 (rising) | 4 |
| `četrnaest` | 3 | 2 (rising) | 1 |
| `četvrtak` | 1 | 2 (rising) | 2 |
| `šargarepa` | 1 | 3 (rising) | 1 |
| `šesnaest` | 2 | 1 (rising) | 1 |
| `šezdeset` | 1 | 2 (rising) | 1 |

Всего: 69.

## Курс совпал с Wiktionary, но ударение восходящее и не на первом слоге

Здесь курс повторяет Wiktionary, а оно описывает преимущественно хорватскую норму.
В сербском произношении такие слова иногда звучат с ударением на слог раньше —
это стоит показать носителю.

| слово | слог в курсе | по Wiktionary | где встречается |
|---|---|---|---|
| `adresa` | 2 | 2 (rising) | 2 |
| `avion` | 2 | 2 (rising) | 2 |
| `biografija` | 3 | 3 (rising) | 3 |
| `budućnost` | 2 | 2 (rising) | 2 |
| `daleko` | 2 | 2 (rising) | 4 |
| `dogovoriti` | 3 | 3 (rising) | 2 |
| `dokument` | 2 | 2 (rising) | 3 |
| `fakultet` | 2 | 2 (rising) | 6 |
| `frižider` | 2 | 2 (rising) | 3 |
| `garancija` | 2 | 2 (rising) | 2 |
| `govoriti` | 2 | 2 (rising) | 4 |
| `ideja` | 2 | 2 (rising) | 3 |
| `jelovnik` | 2 | 2 (rising) | 2 |
| `kafana` | 2 | 2 (rising) | 2 |
| `kancelarija` | 3 | 3 (rising) | 3 |
| `kolega` | 2 | 2 (rising) | 2 |
| `kupatilo` | 2 | 1 (rising), 2 (rising) | 3 |
| `naučiti` | 2 | 2 (rising) | 3 |
| `obavestiti` | 3 | 3 (rising) | 2 |
| `oblačiti` | 2 | 2 (rising) | 1 |
| `odakle` | 2 | 1 (rising), 2 (rising) | 5 |
| `odjednom` | 2 | 2 (rising) | 3 |
| `odmarati` | 2 | 2 (rising) | 1 |
| `odrasti` | 2 | 2 (rising) | 2 |
| `ogledalo` | 2 | 2 (rising) | 2 |
| `original` | 3 | 3 (rising) | 3 |
| `otići` | 2 | 2 (rising) | 3 |
| `otkazati` | 2 | 2 (rising) | 3 |
| `paradajz` | 2 | 2 (rising) | 3 |
| `planina` | 2 | 1 (falling), 2 (rising) | 2 |
| `planirati` | 2 | 2 (rising) | 2 |
| `pokvariti` | 2 | 2 (rising) | 2 |
| `polako` | 2 | 2 (rising) | 2 |
| `ponedeljak` | 2 | 2 (rising) | 3 |
| `ponoviti` | 2 | 2 (rising) | 2 |
| `popodne` | 2 | 2 (rising) | 5 |
| `poručiti` | 2 | 2 (rising) | 1 |
| `potpisati` | 2 | 2 (rising) | 2 |
| `preseliti` | 2 | 2 (rising) | 2 |
| `prijaviti` | 2 | 2 (rising) | 2 |
| `prodavnica` | 2 | 1 (rising), 2 (rising) | 4 |
| `razgovarati` | 3 | 3 (rising) | 2 |
| `restoran` | 2 | 2 (rising) | 2 |
| `rezervacija` | 3 | 3 (rising) | 2 |
| `salata` | 2 | 2 (rising) | 2 |
| `studirati` | 2 | 2 (rising) | 2 |
| `svejedno` | 2 | 2 (rising) | 2 |
| `tableta` | 2 | 2 (rising) | 2 |
| `takođe` | 2 | 2 (rising) | 2 |
| `telefon` | 2 | 2 (rising) | 1 |
| `temperatura` | 4 | 4 (rising) | 2 |
| `toalet` | 2 | 2 (rising) | 2 |
| `zaboraviti` | 2 | 2 (rising) | 2 |
| `zadatak` | 2 | 2 (rising) | 4 |
| `zakasniti` | 2 | 2 (rising) | 2 |
| `zakazati` | 2 | 2 (rising) | 3 |
| `zameniti` | 2 | 2 (rising) | 2 |
| `zanimljiv` | 2 | 2 (rising) | 3 |
| `završiti` | 2 | 2 (rising) | 2 |
| `čokolada` | 3 | 3 (rising) | 2 |

Всего: 60.
