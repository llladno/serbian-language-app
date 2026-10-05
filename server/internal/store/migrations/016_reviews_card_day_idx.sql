-- Migration 016: composite index for the once-a-day review check.
--
-- GradeCard asks, before every graded review, whether this card already has a
-- review inside the learner's local day: user_id and card_id equal, reviewed_at
-- in a range (see cardReviewedOnDayTx). The only index on reviews is
-- reviews_user (user_id), so that lookup scans the learner's whole review
-- history, which only grows. This index matches the query shape exactly.
--
-- Version 015 is deliberately skipped: it is reserved for the history backfill.

CREATE INDEX IF NOT EXISTS reviews_user_card_time ON reviews (user_id, card_id, reviewed_at);
