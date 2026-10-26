# План: выбор уровня после оплаты

Спека: `docs/superpowers/specs/2026-10-26-level-choice-design.md`. Порядок: сначала сервер (тесты первыми), потом сайт.

## Сервер

1. `store`: `CompletePayment` не открывает уровень при пустом `granted_phase`. Тесты: оплата без уровня платит бонус и
   считается в `HasPaidLevel`, уровень закрыт; старый платёж с уровнем открывает как раньше.
2. `store`: `UnclaimedLevelPayments(userID)` и `ClaimLevel(userID, paymentID, phase, now)` (транзакция, один раз,
   повтор того же уровня ок, другой `ErrAlreadyClaimed`, чужой и неподходящий `ErrNotClaimable`). Тесты на SQLite и
   Postgres.
3. `api`: `levelChoices(us)` (порядок курса, закрытые и в продаже, `lessons`, `after`); `nextLockedPhase` через него.
4. `api`: `createOrder` для уровня: пустой `granted_phase`, описание «на выбор», 409 `level_pending_choice`.
5. `api`: `POST /api/me/orders/{id}/level` + маршрут, `orderDTO.choose`.
6. `api`: `GET /api/me/tariffs` отдаёт `choices` и `credits`.
7. Переписать старые тесты платежей (оплата, Т-Касса, лимит, бонус, «открыт ли уровень после оплаты»).

## Сайт

8. `types.ts`, `api.ts` (`chooseLevel`), `stores/tariffs.ts` (`choices`, `credits`, `hasCredit`, `choose`), тесты стора.
9. `lib/levelChoiceModal.ts`, `LevelChoiceModal.vue` (+ тесты), монтаж в `AppNav`.
10. `LevelCreditNotice.vue` (+ тесты), в профиле и на странице тарифов; `TariffsPromo`/`TariffsCard` учитывают право.
11. `TariffsView`: тексты, картинки, окно после возврата с оплаты и при заходе; тесты страницы.
12. Проверка в браузере: светлая и тёмная темы, телефон.

## Прочее

13. Админка: «ждёт выбора» в списке платежей.
14. `CLAUDE.md`, память, `docs/promo-icons-image-prompts.md` не трогаем.
