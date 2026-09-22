package store

import (
	"fmt"
	"strings"
	"time"
)

// Attribution is the first-touch UTM combination captured from the browser.
// Every field is optional; a zero value means no UTM parameters were present.
// See docs/superpowers/specs/2026-09-22-utm-link-tracking-design.md.
type Attribution struct {
	UtmSource, UtmMedium, UtmCampaign, UtmContent string
}

func (a Attribution) empty() bool {
	return a.UtmSource == "" && a.UtmMedium == "" && a.UtmCampaign == "" && a.UtmContent == ""
}

// maxUTMLen caps each UTM field before it reaches the database — the fields
// ultimately come from a public, unauthenticated endpoint (track-visit) and
// must never let a caller stuff an unbounded string into a row.
const maxUTMLen = 100

func clampUTM(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > maxUTMLen {
		return string(r[:maxUTMLen])
	}
	return s
}

func (a Attribution) clamped() Attribution {
	return Attribution{
		UtmSource:   clampUTM(a.UtmSource),
		UtmMedium:   clampUTM(a.UtmMedium),
		UtmCampaign: clampUTM(a.UtmCampaign),
		UtmContent:  clampUTM(a.UtmContent),
	}
}

// SetUserAttribution records userID's first-touch UTM attribution. A no-op
// when attr is empty. The WHERE guard makes a second call for the same user
// a no-op too, so attribution can never be overwritten by a later visit or
// login — call this every time a new-or-existing account resolves, not just
// on the very first one.
func (s *Store) SetUserAttribution(userID string, attr Attribution) error {
	attr = attr.clamped()
	if attr.empty() {
		return nil
	}
	if _, err := s.db.Exec(`UPDATE users SET utm_source = ?, utm_medium = ?, utm_campaign = ?, utm_content = ?
		WHERE id = ? AND utm_source IS NULL`,
		nullIf(attr.UtmSource), nullIf(attr.UtmMedium), nullIf(attr.UtmCampaign), nullIf(attr.UtmContent), userID); err != nil {
		return fmt.Errorf("set user attribution: %w", err)
	}
	return nil
}

// RecordLinkVisit logs one anonymous page view carrying UTM parameters — the
// raw counter behind the admin's "переходы" stat. A no-op when attr is empty.
func (s *Store) RecordLinkVisit(attr Attribution, at time.Time) error {
	attr = attr.clamped()
	if attr.empty() {
		return nil
	}
	if _, err := s.db.Exec(`INSERT INTO link_visits (utm_source, utm_medium, utm_campaign, utm_content, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		nullIf(attr.UtmSource), nullIf(attr.UtmMedium), nullIf(attr.UtmCampaign), nullIf(attr.UtmContent),
		at.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("record link visit: %w", err)
	}
	return nil
}

// DebugQuery runs a read-only, single-column SELECT and returns the values
// as strings. Used by tests in other packages that need to assert on raw
// rows without a purpose-built getter (link_visits has no application-level
// reader — the admin panel queries it directly over SQL).
func (s *Store) DebugQuery(query string) ([]string, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
