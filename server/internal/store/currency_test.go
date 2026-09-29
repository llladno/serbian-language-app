package store

import "testing"

func TestMigration013CreatesSchemaAndSeeds(t *testing.T) {
	s := newStore(t)

	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM economy_settings`).Scan(&n); err != nil {
		t.Fatalf("economy_settings missing: %v", err)
	}
	if n == 0 {
		t.Fatal("economy_settings not seeded")
	}

	var goal string
	if err := s.db.QueryRow(`SELECT value FROM economy_settings WHERE key = ?`, "daily_goal").Scan(&goal); err != nil {
		t.Fatalf("daily_goal: %v", err)
	}
	if goal != "10" {
		t.Fatalf("daily_goal = %q, want \"10\"", goal)
	}

	// Every new table must exist and be empty.
	for _, tbl := range []string{
		"currency_ledger", "user_daily_activity", "streak_repairs", "user_answer_streak",
		"quests", "quest_claims", "products", "promo_codes", "promo_code_products",
		"promo_redemptions", "user_entitlements",
	} {
		if _, err := s.db.Exec(`SELECT 1 FROM ` + tbl + ` WHERE 1 = 0`); err != nil {
			t.Fatalf("table %s: %v", tbl, err)
		}
	}

	// users.timezone exists with the documented default.
	id, err := s.CreateUser("Тест")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	var tz string
	if err := s.db.QueryRow(`SELECT timezone FROM users WHERE id = ?`, id).Scan(&tz); err != nil {
		t.Fatalf("timezone column: %v", err)
	}
	if tz != "Europe/Belgrade" {
		t.Fatalf("timezone = %q, want Europe/Belgrade", tz)
	}
}
