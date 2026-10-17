-- «Уровень»: 299 ₽, and 199 ₽ for the first purchase (was 299 and 249). A price is a
-- row in `tariffs`, so changing it is one more UPDATE like this one. The public offer
-- (landing/app/pages/oferta.vue, п. 4.1) quotes the prices: change both together.
UPDATE tariffs SET price_kopecks = 29900, first_price_kopecks = 19900, updated_at = '2026-10-17T00:00:00Z'
 WHERE code = 'level';
