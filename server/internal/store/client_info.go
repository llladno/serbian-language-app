package store

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	clientVersionRE  = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){0,2}$`)
	clientPlatformRE = regexp.MustCompile(`^[a-z0-9_]{1,24}$`)
)

// ClientInfo is the Telegram app a person last opened the Mini App in, as the app reported it.
type ClientInfo struct {
	Version  string // Telegram.WebApp.version, e.g. "10.1"; "" until reported
	Platform string // Telegram.WebApp.platform, e.g. "ios"; "" until reported
	SeenAt   string // RFC 3339 UTC of the last report
}

// NormalizeClientInfo cleans what the browser sent and says whether it is usable. The values come from the client, so
// anything that is not a plain dotted version or a short lowercase platform name is dropped rather than stored.
func NormalizeClientInfo(version, platform string) (string, string, bool) {
	version = strings.TrimSpace(version)
	platform = strings.ToLower(strings.TrimSpace(platform))
	if !clientVersionRE.MatchString(version) {
		return "", "", false
	}
	if platform != "" && !clientPlatformRE.MatchString(platform) {
		platform = ""
	}
	return version, platform, true
}

// NoteClientInfo remembers the Telegram app version and platform a person's Mini App just reported. A row that already
// holds the same version and platform is not written again, so calling it on every Mini App open costs no write; only
// the seen-at stamp is refreshed, and at most once a day.
func (s *Store) NoteClientInfo(userID, version, platform string) error {
	version, platform, ok := NormalizeClientInfo(version, platform)
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	dayAgo := now.Add(-24 * time.Hour).Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE users SET tg_client_version = ?, tg_client_platform = ?, tg_client_seen_at = ?
		WHERE id = ? AND (COALESCE(tg_client_version,'') <> ? OR COALESCE(tg_client_platform,'') <> ?
			OR COALESCE(tg_client_seen_at,'') < ?)`,
		version, platform, now.Format(time.RFC3339), userID, version, platform, dayAgo); err != nil {
		return fmt.Errorf("note client info: %w", err)
	}
	return nil
}

// ClientInfoOf returns what was last reported for a person; the zero value when nothing was.
func (s *Store) ClientInfoOf(userID string) (ClientInfo, error) {
	var ci ClientInfo
	err := s.db.QueryRow(`SELECT COALESCE(tg_client_version,''), COALESCE(tg_client_platform,''), COALESCE(tg_client_seen_at,'')
		FROM users WHERE id = ?`, userID).Scan(&ci.Version, &ci.Platform, &ci.SeenAt)
	if err != nil {
		return ClientInfo{}, fmt.Errorf("client info: %w", err)
	}
	return ci, nil
}
