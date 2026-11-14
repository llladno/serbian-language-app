-- Which Telegram app a person last opened the course in: the Mini App version Telegram reports (Telegram.WebApp.version,
-- the Bot API level the app understands, e.g. '10.1') and its platform ('ios', 'android', 'tdesktop', 'macos', 'weba'...).
--
-- The Bot API tells a bot nothing about the app on the other end, so this is the only way to know whether a person can
-- see newer message kinds (rich messages since 10.1, widgets since 10.4). It describes the device the Mini App was last
-- opened on, not every device the person reads Telegram on. NULL until the Mini App is opened after this migration;
-- client_seen_at is when it was last reported (RFC 3339, UTC).
ALTER TABLE users ADD COLUMN tg_client_version TEXT;
ALTER TABLE users ADD COLUMN tg_client_platform TEXT;
ALTER TABLE users ADD COLUMN tg_client_seen_at TEXT;
