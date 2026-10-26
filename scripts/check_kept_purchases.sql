-- Должно вернуть 0 строк: всё, что человек получил (купил за зёрнышки, открыл по промокоду, оплатил рублями),
-- у него на месте. Только чтение. Каждая строка это пропажа: откуда получено, кто, какой уровень.
--   level: номер фазы (1b = уровень 2, 2 = уровень 3, 3 = уровень 4, 4 = уровень 5, 5 = уровень 6), all = годовой доступ.
SELECT 'зёрнышки' AS source, l.user_id, p.ref AS level
FROM currency_ledger l
JOIN products p ON p.kind = 'phase_unlock' AND p.id = (CASE WHEN l.ref ~ '^[0-9]+$' THEN l.ref::bigint END)
WHERE l.kind = 'purchase'
  AND NOT EXISTS (SELECT 1 FROM user_entitlements e WHERE e.user_id = l.user_id AND e.kind = 'phase_unlock' AND e.ref = p.ref)
UNION ALL
SELECT 'промокод', r.user_id, p.ref
FROM promo_redemptions r
JOIN products p ON p.kind = 'phase_unlock' AND p.id = r.product_id
WHERE NOT EXISTS (SELECT 1 FROM user_entitlements e WHERE e.user_id = r.user_id AND e.kind = 'phase_unlock' AND e.ref = p.ref)
UNION ALL
SELECT 'рубли', y.user_id, y.granted_phase
FROM payments y
WHERE y.status = 'succeeded' AND y.tariff_code = 'level' AND y.granted_phase <> ''
  AND NOT EXISTS (SELECT 1 FROM user_entitlements e WHERE e.user_id = y.user_id AND e.kind = 'phase_unlock' AND e.ref = y.granted_phase)
UNION ALL
SELECT 'рубли', y.user_id, 'all'
FROM payments y
WHERE y.status = 'succeeded' AND y.tariff_code = 'year'
  AND NOT EXISTS (SELECT 1 FROM user_entitlements e WHERE e.user_id = y.user_id AND e.kind = 'all_access')
ORDER BY 1, 2, 3;
