package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The «return» campaign's storage (docs/superpowers/specs/2026-11-08-return-campaign-design.md): who is in it, which wave and
// button style each person got, and what came of it. The rules (who is eligible, how the waves split) live in the campaign
// package and api/campaign.go; this file only reads and writes.

// Kinds of campaign_sends row and the channels a campaign writes through.
const (
	CampaignSent    = "sent"    // the person gets a message
	CampaignHoldout = "holdout" // the person is never written to; the row is there for the comparison
	ChannelTelegram = "tg"
	ChannelEmail    = "email"
)

// systemOnboardingStamp is the updated_at that migration 039 put on every account that existed when the tour was introduced. It
// tells «the system put it off» from «the learner pressed "later"».
const systemOnboardingStamp = "2026-11-04T00:00:00Z"

// campaignQuiet is how long someone must have been away (from their last activity, or from registering if they never studied)
// before the campaign writes to them: a person who is still around does not need calling back.
const campaignQuiet = 48 * time.Hour

// CampaignCandidate is one person the campaign would write to.
type CampaignCandidate struct {
	UserID  string
	Segment int // 1 did nothing, 2 started lesson 00, 3 finished 00, 4 finished 01 (never opened 02)
	Channel string
	ChatID  int64  // Telegram chat, for ChannelTelegram
	Email   string // verified address, for ChannelEmail
}

// campaignSegment is the segment of a learner who has not opened lesson 02: how far they got in the first lessons.
func campaignSegment(started00, done00, done01 bool) int {
	switch {
	case done01:
		return 4
	case done00:
		return 3
	case started00:
		return 2
	}
	return 1
}

// CampaignAudience works out who the campaign would write to as of now: everyone who never opened lesson 02 and has been quiet for
// two days, that is not already in this campaign, together with the channel to reach them by. Telegram first (linked, study
// reminders on, the bot not known to be blocked); otherwise a verified e-mail that has not been unsubscribed. Someone with
// neither is not in the audience.
func (s *Store) CampaignAudience(campaign string, now time.Time) ([]CampaignCandidate, error) {
	type learner struct {
		created                   time.Time
		started00, done00, done01 bool
		reached02                 bool
		chat                      int64
		email                     string
		learningOff               bool
	}
	people := map[string]*learner{}
	var order []string

	rows, err := s.db.Query(`SELECT id, created_at FROM users`)
	if err != nil {
		return nil, fmt.Errorf("campaign audience: users: %w", err)
	}
	for rows.Next() {
		var id, created string
		if err := rows.Scan(&id, &created); err != nil {
			rows.Close()
			return nil, fmt.Errorf("campaign audience: users: %w", err)
		}
		people[id] = &learner{created: parseStamp(created)}
		order = append(order, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("campaign audience: users: %w", err)
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT user_id, lesson, status FROM lesson_progress WHERE lesson IN ('00', '01', '02')`)
	if err != nil {
		return nil, fmt.Errorf("campaign audience: progress: %w", err)
	}
	for rows.Next() {
		var id, lesson, status string
		if err := rows.Scan(&id, &lesson, &status); err != nil {
			rows.Close()
			return nil, fmt.Errorf("campaign audience: progress: %w", err)
		}
		l := people[id]
		if l == nil {
			continue
		}
		switch lesson {
		case "00":
			l.started00 = true
			l.done00 = l.done00 || status == "done"
		case "01":
			l.done01 = l.done01 || status == "done"
		case "02":
			l.reached02 = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("campaign audience: progress: %w", err)
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT user_id, provider, provider_uid, COALESCE(email, ''), COALESCE(email_verified_at, '') FROM identities`)
	if err != nil {
		return nil, fmt.Errorf("campaign audience: identities: %w", err)
	}
	for rows.Next() {
		var id, provider, uid, email, verified string
		if err := rows.Scan(&id, &provider, &uid, &email, &verified); err != nil {
			rows.Close()
			return nil, fmt.Errorf("campaign audience: identities: %w", err)
		}
		l := people[id]
		if l == nil {
			continue
		}
		switch {
		case provider == "telegram" && !strings.HasPrefix(uid, "pending:"):
			if chat, err := strconv.ParseInt(uid, 10, 64); err == nil {
				l.chat = chat
			}
		case provider == "password" && email != "" && verified != "":
			l.email = email
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("campaign audience: identities: %w", err)
	}
	rows.Close()

	if err := s.collectStrings(`SELECT user_id FROM bot_prefs WHERE learning = 0`, func(id string) {
		if l := people[id]; l != nil {
			l.learningOff = true
		}
	}); err != nil {
		return nil, fmt.Errorf("campaign audience: prefs: %w", err)
	}
	blocked := map[int64]bool{}
	if err := s.collectStrings(`SELECT DISTINCT chat_id FROM bot_outbox WHERE status = 'failed'
		AND (error LIKE '%blocked%' OR error LIKE '%deactivated%' OR error LIKE '%chat not found%')`, func(chat string) {
		if n, err := strconv.ParseInt(chat, 10, 64); err == nil {
			blocked[n] = true
		}
	}); err != nil {
		return nil, fmt.Errorf("campaign audience: blocked chats: %w", err)
	}
	optedOut := map[string]bool{}
	if err := s.collectStrings(`SELECT user_id FROM email_optout`, func(id string) { optedOut[id] = true }); err != nil {
		return nil, fmt.Errorf("campaign audience: e-mail opt-outs: %w", err)
	}
	inCampaign := map[string]bool{}
	if err := s.collectStrings(`SELECT user_id FROM campaign_sends WHERE campaign = ?`, func(id string) { inCampaign[id] = true }, campaign); err != nil {
		return nil, fmt.Errorf("campaign audience: existing rows: %w", err)
	}
	last, err := s.LastActivityByUser()
	if err != nil {
		return nil, fmt.Errorf("campaign audience: %w", err)
	}

	var out []CampaignCandidate
	for _, id := range order {
		l := people[id]
		if l.reached02 || inCampaign[id] {
			continue
		}
		seen := l.created
		if t, ok := last[id]; ok && t.After(seen) {
			seen = t
		}
		if now.Sub(seen) < campaignQuiet {
			continue
		}
		c := CampaignCandidate{UserID: id, Segment: campaignSegment(l.started00, l.done00, l.done01)}
		switch {
		case l.chat != 0 && !l.learningOff && !blocked[l.chat]:
			c.Channel, c.ChatID = ChannelTelegram, l.chat
		case l.email != "" && !optedOut[id]:
			c.Channel, c.Email = ChannelEmail, l.email
		default:
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// collectStrings runs a one-column query and hands each value to f.
func (s *Store) collectStrings(query string, f func(string), args ...any) error {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		f(v)
	}
	return rows.Err()
}

// CampaignRow is one person to put into a campaign's frozen audience.
type CampaignRow struct {
	UserID  string
	Channel string
	Segment int
	Kind    string // CampaignSent or CampaignHoldout
	Ord     int64  // place in the order the waves take people in
}

// CampaignFreeze writes the audience. A person already in the campaign is left as they are.
func (s *Store) CampaignFreeze(campaign string, rows []CampaignRow, now time.Time) error {
	for _, r := range rows {
		if _, err := s.db.Exec(`INSERT INTO campaign_sends (campaign, user_id, channel, segment, kind, ord, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT (campaign, user_id) DO NOTHING`,
			campaign, r.UserID, r.Channel, r.Segment, r.Kind, r.Ord, stamp(now)); err != nil {
			return fmt.Errorf("campaign freeze: %w", err)
		}
	}
	return nil
}

// CampaignFrozen reports whether the campaign already has an audience.
func (s *Store) CampaignFrozen(campaign string) (bool, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM campaign_sends WHERE campaign = ?`, campaign).Scan(&n); err != nil {
		return false, fmt.Errorf("campaign frozen: %w", err)
	}
	return n > 0, nil
}

// CampaignWavesOpened is how many waves have been opened, in any channel (waves open one after another).
func (s *Store) CampaignWavesOpened(campaign string) (int, error) {
	var n sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(wave) FROM campaign_waves WHERE campaign = ?`, campaign).Scan(&n); err != nil {
		return 0, fmt.Errorf("campaign waves opened: %w", err)
	}
	return int(n.Int64), nil
}

// CampaignTaken is a person a wave has just taken, with what they were given.
type CampaignTaken struct {
	UserID  string
	Segment int
	Style   int
}

// CampaignTake takes the next limit people (all of them if limit < 0) of a channel who have not been taken by a wave, in the
// order they were frozen in, gives each a button style from styleOf and a click token, and returns them. Holdout people are never
// taken. With commit false nothing is written: it only says who would be taken and with which style.
func (s *Store) CampaignTake(campaign, channel string, wave, limit int, styleOf func(userID string) int, commit bool) ([]CampaignTaken, error) {
	q := `SELECT user_id, segment FROM campaign_sends
		WHERE campaign = ? AND channel = ? AND kind = ? AND wave = 0 ORDER BY ord ASC, id ASC`
	args := []any{campaign, channel, CampaignSent}
	if limit >= 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("campaign take: %w", err)
	}
	var out []CampaignTaken
	for rows.Next() {
		var t CampaignTaken
		if err := rows.Scan(&t.UserID, &t.Segment); err != nil {
			rows.Close()
			return nil, fmt.Errorf("campaign take: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("campaign take: %w", err)
	}
	rows.Close()
	for i := range out {
		if !commit {
			out[i].Style = styleOf(out[i].UserID)
			continue
		}
		var raw [8]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return nil, fmt.Errorf("campaign take: token: %w", err)
		}
		out[i].Style = styleOf(out[i].UserID)
		if _, err := s.db.Exec(`UPDATE campaign_sends SET wave = ?, style = ?, token = ?
			WHERE campaign = ? AND user_id = ? AND wave = 0`,
			wave, out[i].Style, hex.EncodeToString(raw[:]), campaign, out[i].UserID); err != nil {
			return nil, fmt.Errorf("campaign take: %w", err)
		}
	}
	return out, nil
}

// CampaignRecordWave notes that a wave was opened in a channel, with the leading style and its share (in thousandths).
func (s *Store) CampaignRecordWave(campaign string, wave int, channel string, leader, leaderShare int, at time.Time) error {
	if _, err := s.db.Exec(`INSERT INTO campaign_waves (campaign, wave, channel, opened_at, leader, leader_share)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (campaign, wave, channel) DO NOTHING`,
		campaign, wave, channel, stamp(at), leader, leaderShare); err != nil {
		return fmt.Errorf("campaign record wave: %w", err)
	}
	return nil
}

// CampaignStyleStat is how a button style has done so far in a channel.
type CampaignStyleStat struct {
	Sent    int
	Clicked int
}

// CampaignStyleStats counts, per style, the messages that went out in a channel and the ones whose button was pressed.
func (s *Store) CampaignStyleStats(campaign, channel string) (map[int]CampaignStyleStat, error) {
	rows, err := s.db.Query(`SELECT style, COUNT(*), SUM(CASE WHEN clicked_at <> '' THEN 1 ELSE 0 END)
		FROM campaign_sends WHERE campaign = ? AND channel = ? AND kind = ? AND sent_at <> '' GROUP BY style`,
		campaign, channel, CampaignSent)
	if err != nil {
		return nil, fmt.Errorf("campaign style stats: %w", err)
	}
	defer rows.Close()
	out := map[int]CampaignStyleStat{}
	for rows.Next() {
		var style int
		var st CampaignStyleStat
		if err := rows.Scan(&style, &st.Sent, &st.Clicked); err != nil {
			return nil, fmt.Errorf("campaign style stats: %w", err)
		}
		out[style] = st
	}
	return out, rows.Err()
}

// CampaignUnsent counts the people a wave has taken whose message has not gone out yet.
func (s *Store) CampaignUnsent(campaign string) (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM campaign_sends WHERE campaign = ? AND kind = ? AND wave > 0 AND sent_at = ''`,
		campaign, CampaignSent).Scan(&n); err != nil {
		return 0, fmt.Errorf("campaign unsent: %w", err)
	}
	return n, nil
}

// CampaignDueRow is a taken person whose message is still to be sent.
type CampaignDueRow struct {
	ID      int64
	UserID  string
	Channel string
	Segment int
	Style   int
	Token   string
}

// CampaignDue lists the messages taken by a wave and not sent yet, oldest wave first.
func (s *Store) CampaignDue(campaign string) ([]CampaignDueRow, error) {
	rows, err := s.db.Query(`SELECT id, user_id, channel, segment, style, token FROM campaign_sends
		WHERE campaign = ? AND kind = ? AND wave > 0 AND sent_at = '' ORDER BY wave ASC, ord ASC`, campaign, CampaignSent)
	if err != nil {
		return nil, fmt.Errorf("campaign due: %w", err)
	}
	defer rows.Close()
	var out []CampaignDueRow
	for rows.Next() {
		var r CampaignDueRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.Channel, &r.Segment, &r.Style, &r.Token); err != nil {
			return nil, fmt.Errorf("campaign due: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CampaignSkip takes a person out of the people to be written to, because they can no longer be reached (they switched the
// reminders off, unsubscribed or unlinked Telegram between the wave opening and their message). Their row stays, as «skipped»,
// so the waves do not wait for a message that will never go out.
func (s *Store) CampaignSkip(id int64) error {
	if _, err := s.db.Exec(`UPDATE campaign_sends SET kind = 'skipped' WHERE id = ? AND sent_at = ''`, id); err != nil {
		return fmt.Errorf("campaign skip: %w", err)
	}
	return nil
}

// CampaignClaim takes a campaign message for sending: it stamps sent_at in one statement and reports whether this caller got it. The
// sweep claims a row before it sends, so two sweeps running at once (the old and the new container during a deploy) cannot both send
// the same person's message. A crash after the claim loses that message and never repeats it.
func (s *Store) CampaignClaim(id int64, at time.Time) (bool, error) {
	res, err := s.db.Exec(`UPDATE campaign_sends SET sent_at = ? WHERE id = ? AND kind = ? AND sent_at = ''`, stamp(at), id, CampaignSent)
	if err != nil {
		return false, fmt.Errorf("campaign claim: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("campaign claim: %w", err)
	}
	return n == 1, nil
}

// CampaignMarkSent records that the message of one row went out.
func (s *Store) CampaignMarkSent(id int64, at time.Time) error {
	if _, err := s.db.Exec(`UPDATE campaign_sends SET sent_at = ? WHERE id = ? AND sent_at = ''`, stamp(at), id); err != nil {
		return fmt.Errorf("campaign mark sent: %w", err)
	}
	return nil
}

// NoteCampaignClick records the first press on the button of the message with this token and reports whether it was the first.
// An unknown token is ignored.
func (s *Store) NoteCampaignClick(token string, at time.Time) (bool, error) {
	if token == "" {
		return false, nil
	}
	res, err := s.db.Exec(`UPDATE campaign_sends SET clicked_at = ? WHERE token = ? AND clicked_at = ''`, stamp(at), token)
	if err != nil {
		return false, fmt.Errorf("note campaign click: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("note campaign click: %w", err)
	}
	return n == 1, nil
}

// CampaignUserByToken finds who a campaign message was sent to, for the unsubscribe link in it.
func (s *Store) CampaignUserByToken(token string) (string, bool, error) {
	if token == "" {
		return "", false, nil
	}
	var id string
	err := s.db.QueryRow(`SELECT user_id FROM campaign_sends WHERE token = ?`, token).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("campaign user by token: %w", err)
	}
	return id, true, nil
}

// SetEmailOptout records that a person does not want study e-mails.
func (s *Store) SetEmailOptout(userID string, at time.Time) error {
	if _, err := s.db.Exec(`INSERT INTO email_optout (user_id, created_at) VALUES (?, ?) ON CONFLICT (user_id) DO NOTHING`,
		userID, stamp(at)); err != nil {
		return fmt.Errorf("set e-mail opt-out: %w", err)
	}
	return nil
}

// EmailOptedOut reports whether the person asked not to get study e-mails.
func (s *Store) EmailOptedOut(userID string) (bool, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM email_optout WHERE user_id = ?`, userID).Scan(&n); err != nil {
		return false, fmt.Errorf("e-mail opt-out: %w", err)
	}
	return n > 0, nil
}

// VerifiedEmail is the person's confirmed e-mail address, if they have one.
func (s *Store) VerifiedEmail(userID string) (string, bool) {
	var email string
	err := s.db.QueryRow(`SELECT email FROM identities WHERE user_id = ? AND provider = 'password'
		AND email IS NOT NULL AND email <> '' AND email_verified_at IS NOT NULL AND email_verified_at <> ''`, userID).Scan(&email)
	if err != nil {
		return "", false
	}
	return email, true
}

// ResetSystemOnboarding lets the swallow's tour show again to a learner whose tour was only put off by migration 039 (every
// account that existed then). A learner who pressed «later» themselves, or who finished it, keeps that.
func (s *Store) ResetSystemOnboarding(userID string) error {
	if _, err := s.db.Exec(`DELETE FROM onboarding WHERE user_id = ? AND status = 'postponed' AND updated_at = ?`,
		userID, systemOnboardingStamp); err != nil {
		return fmt.Errorf("reset onboarding: %w", err)
	}
	return nil
}

// CampaignCell is one line of the results: a wave, channel, segment and style with how many people it took, how many messages went
// out, and how many buttons were pressed.
type CampaignCell struct {
	Wave, Segment, Style int
	Channel              string
	Taken, Sent, Clicked int
}

// CampaignGroup compares the people who got a message with the group that never did, for one segment: how many reached the lesson
// that follows their segment after the audience was frozen.
type CampaignGroup struct {
	Segment int
	Kind    string
	People  int
	Reached int
}

// CampaignResults is everything needed to read a campaign.
type CampaignResults struct {
	Cells  []CampaignCell
	Groups []CampaignGroup
	Waves  []CampaignWaveInfo
}

// CampaignWaveInfo is how a wave was opened in a channel.
type CampaignWaveInfo struct {
	Wave, Leader, LeaderShare int
	Channel, OpenedAt         string
}

// campaignTarget is the lesson a learner of a segment is being called back to, and whether finishing it (rather than only opening
// it) is what counts.
func campaignTarget(segment int) (lesson string, mustFinish bool) {
	switch segment {
	case 1, 2:
		return "00", true
	case 3:
		return "01", true
	}
	return "02", false
}

// CampaignResults reads the campaign's results.
func (s *Store) CampaignResults(campaign string) (CampaignResults, error) {
	var out CampaignResults
	rows, err := s.db.Query(`SELECT wave, channel, segment, style, COUNT(*),
			SUM(CASE WHEN sent_at <> '' THEN 1 ELSE 0 END), SUM(CASE WHEN clicked_at <> '' THEN 1 ELSE 0 END)
		FROM campaign_sends WHERE campaign = ? AND kind = ? AND wave > 0
		GROUP BY wave, channel, segment, style ORDER BY wave, channel, segment, style`, campaign, CampaignSent)
	if err != nil {
		return out, fmt.Errorf("campaign results: %w", err)
	}
	for rows.Next() {
		var c CampaignCell
		if err := rows.Scan(&c.Wave, &c.Channel, &c.Segment, &c.Style, &c.Taken, &c.Sent, &c.Clicked); err != nil {
			rows.Close()
			return out, fmt.Errorf("campaign results: %w", err)
		}
		out.Cells = append(out.Cells, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, fmt.Errorf("campaign results: %w", err)
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT wave, channel, leader, leader_share, opened_at FROM campaign_waves WHERE campaign = ? ORDER BY wave, channel`, campaign)
	if err != nil {
		return out, fmt.Errorf("campaign results: waves: %w", err)
	}
	for rows.Next() {
		var w CampaignWaveInfo
		if err := rows.Scan(&w.Wave, &w.Channel, &w.Leader, &w.LeaderShare, &w.OpenedAt); err != nil {
			rows.Close()
			return out, fmt.Errorf("campaign results: waves: %w", err)
		}
		out.Waves = append(out.Waves, w)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, fmt.Errorf("campaign results: waves: %w", err)
	}
	rows.Close()

	type person struct {
		id, kind, frozen string
		segment          int
	}
	var people []person
	rows, err = s.db.Query(`SELECT user_id, kind, created_at, segment FROM campaign_sends
		WHERE campaign = ? AND (kind = ? OR (kind = ? AND sent_at <> ''))`, campaign, CampaignHoldout, CampaignSent)
	if err != nil {
		return out, fmt.Errorf("campaign results: people: %w", err)
	}
	for rows.Next() {
		var p person
		if err := rows.Scan(&p.id, &p.kind, &p.frozen, &p.segment); err != nil {
			rows.Close()
			return out, fmt.Errorf("campaign results: people: %w", err)
		}
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, fmt.Errorf("campaign results: people: %w", err)
	}
	rows.Close()

	groups := map[[2]string]*CampaignGroup{}
	for _, p := range people {
		lesson, mustFinish := campaignTarget(p.segment)
		var n int
		var err error
		if mustFinish {
			err = s.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ? AND lesson = ? AND status = 'done' AND completed_at > ?`,
				p.id, lesson, p.frozen).Scan(&n)
		} else {
			err = s.db.QueryRow(`SELECT COUNT(*) FROM lesson_progress WHERE user_id = ? AND lesson = ? AND started_at > ?`,
				p.id, lesson, p.frozen).Scan(&n)
		}
		if err != nil {
			return out, fmt.Errorf("campaign results: reached: %w", err)
		}
		key := [2]string{strconv.Itoa(p.segment), p.kind}
		g := groups[key]
		if g == nil {
			g = &CampaignGroup{Segment: p.segment, Kind: p.kind}
			groups[key] = g
		}
		g.People++
		if n > 0 {
			g.Reached++
		}
	}
	for seg := 1; seg <= 4; seg++ {
		for _, kind := range []string{CampaignSent, CampaignHoldout} {
			if g := groups[[2]string{strconv.Itoa(seg), kind}]; g != nil {
				out.Groups = append(out.Groups, *g)
			}
		}
	}
	return out, nil
}
