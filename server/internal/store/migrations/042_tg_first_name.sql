-- The first name a Telegram user goes by, as Telegram last told us (Mini App initData or a message to the bot).
--
-- users.name of a Telegram account is mostly the @username (the account is created with the username, see
-- resolveTelegramLogin), so it is no good for «Привет, <name>!». The first name is kept here, per identity, and
-- refreshed every time Telegram shows it to us. Empty until then, so a message that wants a name has to cope without one.
ALTER TABLE identities ADD COLUMN tg_first_name TEXT;
