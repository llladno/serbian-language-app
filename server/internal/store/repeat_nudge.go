package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// The two kinds of repeat_nudges row.
const (
	NudgeSent    = "sent"    // a message went to the learner
	NudgeControl = "control" // the learner is in the half that is never written to; the row only marks the spell
)

// RepeatNudgeInfo is what the reminder sweep needs to decide whether to write to one learner.
type RepeatNudgeInfo struct {
	LastRepeat time.Time // latest time they opened «Повторить»; zero = never
	LastNudge  time.Time // latest nudge row of either kind; zero = none
	Sent       int       // real messages sent so far (control rows not counted)
	OptedOut   bool
}

func parseStamp(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// NoteRepeatOpened records that the learner opened «Повторить» at now, and marks the newest nudge
// written for them as answered if it was not yet. A learner with unlimited repeats has no stored
// batch, so this is the only trace of their repeats.
func (u *UserStore) NoteRepeatOpened(now time.Time) error {
	at := stamp(now)
	if _, err := u.db.Exec(`INSERT INTO repeat_state (user_id, last_opened_at) VALUES (?, ?)
		ON CONFLICT (user_id) DO UPDATE SET last_opened_at = excluded.last_opened_at`, u.user, at); err != nil {
		return fmt.Errorf("note repeat opened: %w", err)
	}
	if _, err := u.db.Exec(`UPDATE repeat_nudges SET opened_at = ?
		WHERE opened_at = '' AND id = (SELECT MAX(id) FROM repeat_nudges WHERE user_id = ?)`, at, u.user); err != nil {
		return fmt.Errorf("note repeat opened: mark nudge: %w", err)
	}
	return nil
}

// RepeatNudgeInfo reads one learner's nudge state. Their last repeat is the later of the open
// recorded by NoteRepeatOpened and the newest stored batch: batches are all there is for repeats
// made before the nudge existed.
func (s *Store) RepeatNudgeInfo(userID string) (RepeatNudgeInfo, error) {
	var info RepeatNudgeInfo
	var opened sql.NullString
	var off sql.NullInt64
	err := s.db.QueryRow(`SELECT last_opened_at, nudge_off FROM repeat_state WHERE user_id = ?`, userID).Scan(&opened, &off)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return info, fmt.Errorf("repeat nudge info: %w", err)
	}
	info.LastRepeat = parseStamp(opened.String)
	info.OptedOut = off.Int64 == 1

	var batch sql.NullString
	if err := s.db.QueryRow(`SELECT MAX(started_at) FROM repeat_sessions WHERE user_id = ?`, userID).Scan(&batch); err != nil {
		return info, fmt.Errorf("repeat nudge info: batches: %w", err)
	}
	if t := parseStamp(batch.String); t.After(info.LastRepeat) {
		info.LastRepeat = t
	}

	var last sql.NullString
	if err := s.db.QueryRow(`SELECT MAX(created_at), COALESCE(SUM(CASE WHEN kind = ? THEN 1 ELSE 0 END), 0)
		FROM repeat_nudges WHERE user_id = ?`, NudgeSent, userID).Scan(&last, &info.Sent); err != nil {
		return info, fmt.Errorf("repeat nudge info: nudges: %w", err)
	}
	info.LastNudge = parseStamp(last.String)
	return info, nil
}

// RecordRepeatNudge writes one nudge row (a sent message or a control-group mark) and reports
// whether it did. It writes nothing when a row already exists since the learner's last repeat:
// the check and the write are one statement, so two sweeps running at once (the old and the new
// container for a minute during a deploy) cannot both pass it.
func (s *Store) RecordRepeatNudge(userID string, variant int, kind string, at, lastRepeat time.Time) (bool, error) {
	res, err := s.db.Exec(`INSERT INTO repeat_nudges (user_id, variant, kind, created_at)
		SELECT ?, ?, ?, ? WHERE NOT EXISTS (
			SELECT 1 FROM repeat_nudges WHERE user_id = ? AND created_at >= ?)`,
		userID, variant, kind, stamp(at), userID, stamp(lastRepeat))
	if err != nil {
		return false, fmt.Errorf("record repeat nudge: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record repeat nudge: %w", err)
	}
	return n == 1, nil
}

// SetRepeatNudgeOff remembers that the learner pressed «Не про повторения».
func (s *Store) SetRepeatNudgeOff(userID string) error {
	if _, err := s.db.Exec(`INSERT INTO repeat_state (user_id, nudge_off) VALUES (?, 1)
		ON CONFLICT (user_id) DO UPDATE SET nudge_off = 1`, userID); err != nil {
		return fmt.Errorf("set repeat nudge off: %w", err)
	}
	return nil
}

// RepeatNudgeGroup is one line of the results: how many learners got one kind of nudge in one
// text, and how many opened «Повторить» within a day of it.
type RepeatNudgeGroup struct {
	Kind            string
	Variant         int
	Count           int
	OpenedWithin24h int
}

// RepeatNudgeStats is everything needed to judge the nudge.
type RepeatNudgeStats struct {
	Groups   []RepeatNudgeGroup
	OptedOut int // learners who pressed «Не про повторения»
}

// RepeatNudgeStats counts nudges by kind and text, with how many were followed by an open of
// «Повторить» within 24 hours. Comparing the control rows with the sent ones says whether the
// message helps; OptedOut says what it costs.
func (s *Store) RepeatNudgeStats() (RepeatNudgeStats, error) {
	var out RepeatNudgeStats
	rows, err := s.db.Query(`SELECT kind, variant, created_at, opened_at FROM repeat_nudges ORDER BY kind, variant, id`)
	if err != nil {
		return out, fmt.Errorf("repeat nudge stats: %w", err)
	}
	defer rows.Close()
	index := map[[2]any]int{}
	for rows.Next() {
		var kind, created, opened string
		var variant int
		if err := rows.Scan(&kind, &variant, &created, &opened); err != nil {
			return out, fmt.Errorf("repeat nudge stats: %w", err)
		}
		key := [2]any{kind, variant}
		i, ok := index[key]
		if !ok {
			i = len(out.Groups)
			index[key] = i
			out.Groups = append(out.Groups, RepeatNudgeGroup{Kind: kind, Variant: variant})
		}
		out.Groups[i].Count++
		if opened != "" && parseStamp(opened).Sub(parseStamp(created)) <= 24*time.Hour {
			out.Groups[i].OpenedWithin24h++
		}
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("repeat nudge stats: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM repeat_state WHERE nudge_off = 1`).Scan(&out.OptedOut); err != nil {
		return out, fmt.Errorf("repeat nudge stats: opt-outs: %w", err)
	}
	return out, nil
}
