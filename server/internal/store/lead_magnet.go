package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// LeadMagnetStart records that a Telegram user followed a magnet's link. The first
// visit wins: a repeat visit never rewrites source or was_member, so "already in the
// channel when they arrived" stays true to the moment it was decided.
func (s *Store) LeadMagnetStart(slug string, tgID int64, source string, wasMember bool, at time.Time) error {
	member := 0
	if wasMember {
		member = 1
	}
	_, err := s.db.Exec(`INSERT INTO lead_magnet_claims (slug, tg_id, source, was_member, started_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT (slug, tg_id) DO NOTHING`,
		slug, strconv.FormatInt(tgID, 10), source, member, at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("lead magnet start: %w", err)
	}
	return nil
}

// LeadMagnetDelivered records a successful send: delivered_at keeps the first time,
// deliveries counts every send (a user who asks again gets the file again).
func (s *Store) LeadMagnetDelivered(slug string, tgID int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE lead_magnet_claims
		SET delivered_at = COALESCE(delivered_at, ?), deliveries = deliveries + 1
		WHERE slug = ? AND tg_id = ?`,
		at.UTC().Format(time.RFC3339), slug, strconv.FormatInt(tgID, 10))
	if err != nil {
		return fmt.Errorf("lead magnet delivered: %w", err)
	}
	return nil
}

// LeadMagnetFileID returns the cached Telegram file_id for a magnet whose content has
// the given sha256, ok=false when none is cached (or the file has changed since).
func (s *Store) LeadMagnetFileID(slug, sha string) (string, bool, error) {
	var id string
	err := s.db.QueryRow(`SELECT file_id FROM lead_magnet_files WHERE slug = ? AND sha = ?`, slug, sha).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lead magnet file id: %w", err)
	}
	return id, true, nil
}

// SaveLeadMagnetFileID remembers the file_id Telegram returned for an upload.
func (s *Store) SaveLeadMagnetFileID(slug, sha, fileID string, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO lead_magnet_files (slug, sha, file_id, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (slug) DO UPDATE SET sha = ?, file_id = ?, updated_at = ?`,
		slug, sha, fileID, at.UTC().Format(time.RFC3339), sha, fileID, at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save lead magnet file id: %w", err)
	}
	return nil
}

// LeadSourceStats is the funnel for one source tag (the part of the link after the slug).
type LeadSourceStats struct {
	Source          string
	Started         int
	Received        int
	SubscribedForIt int
}

// LeadStats is the funnel of one magnet.
type LeadStats struct {
	Started         int // followed the link
	AlreadyMember   int // were in the channel when they arrived
	SubscribedForIt int // were not, and got the file after subscribing: the channel growth this magnet caused
	Received        int // got the file at least once
	Waiting         int // followed the link, no file yet (not subscribed, or never pressed "check")
	BySource        []LeadSourceStats
}

// LeadMagnetStats counts the funnel for slug.
func (s *Store) LeadMagnetStats(slug string) (LeadStats, error) {
	var st LeadStats
	err := s.db.QueryRow(`SELECT COUNT(*),
			COALESCE(SUM(was_member), 0),
			COALESCE(SUM(CASE WHEN was_member = 0 AND delivered_at IS NOT NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN delivered_at IS NOT NULL THEN 1 ELSE 0 END), 0)
		FROM lead_magnet_claims WHERE slug = ?`, slug).
		Scan(&st.Started, &st.AlreadyMember, &st.SubscribedForIt, &st.Received)
	if err != nil {
		return LeadStats{}, fmt.Errorf("lead magnet stats: %w", err)
	}
	st.Waiting = st.Started - st.Received

	rows, err := s.db.Query(`SELECT source, COUNT(*),
			COALESCE(SUM(CASE WHEN delivered_at IS NOT NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN was_member = 0 AND delivered_at IS NOT NULL THEN 1 ELSE 0 END), 0)
		FROM lead_magnet_claims WHERE slug = ? GROUP BY source ORDER BY COUNT(*) DESC, source ASC`, slug)
	if err != nil {
		return LeadStats{}, fmt.Errorf("lead magnet stats by source: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r LeadSourceStats
		if err := rows.Scan(&r.Source, &r.Started, &r.Received, &r.SubscribedForIt); err != nil {
			return LeadStats{}, fmt.Errorf("lead magnet stats by source: %w", err)
		}
		st.BySource = append(st.BySource, r)
	}
	return st, rows.Err()
}

// SetEconomySetting upserts one economy_settings value (the admin panel writes the same
// table). Used for the channel the subscription gate checks.
func (s *Store) SetEconomySetting(key, value string, at time.Time) error {
	_, err := s.db.Exec(`INSERT INTO economy_settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = ?, updated_at = ?`,
		key, value, at.UTC().Format(time.RFC3339), value, at.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("set economy setting %s: %w", key, err)
	}
	return nil
}
