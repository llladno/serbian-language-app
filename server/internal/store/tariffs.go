package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Tariff is one thing that can be bought for rubles. Prices are kopecks.
type Tariff struct {
	Code              string
	Title             string
	PriceKopecks      int64
	FirstPriceKopecks int64 // 0: no first-purchase price
	OldPriceKopecks   int64 // 0: no struck-through price
	PeriodDays        int   // 0: the purchase never expires
	// BonusCurrency is credited in the in-app currency, once, when the tariff is
	// paid for. 0: no bonus.
	BonusCurrency int64
	Active        bool
}

const (
	TariffLevel = "level"
	TariffYear  = "year"

	// AllAccess is the entitlement a year grants: every level until it ends.
	AllAccessKind = "all_access"
	AllAccessRef  = "year"

	// LedgerTopup is the ledger kind for currency that arrives with money (the
	// admin's transactions screen already labels it «Пополнение»).
	LedgerTopup = "topup"

	PaymentPending   = "pending"
	PaymentSucceeded = "succeeded"
	PaymentCanceled  = "canceled"
)

var ErrTariffUnavailable = errors.New("tariff unavailable")

// ErrNotClaimable: the payment is not a paid level credit of this learner (it does
// not exist, belongs to somebody else, is not paid, is a year, or no level was named).
// ErrCreditSpent: the credit was already spent on another level.
var (
	ErrNotClaimable = errors.New("payment is not an unspent level credit")
	ErrCreditSpent  = errors.New("level credit already spent on another level")
)

// Tariff reads one tariff, active or not.
func (s *Store) Tariff(code string) (Tariff, error) {
	var t Tariff
	var active int
	err := s.db.QueryRow(`SELECT code, title, price_kopecks, first_price_kopecks, old_price_kopecks,
		period_days, bonus_currency, active FROM tariffs WHERE code = ?`, code).
		Scan(&t.Code, &t.Title, &t.PriceKopecks, &t.FirstPriceKopecks, &t.OldPriceKopecks, &t.PeriodDays,
			&t.BonusCurrency, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return Tariff{}, ErrTariffUnavailable
	}
	if err != nil {
		return Tariff{}, fmt.Errorf("tariff %s: %w", code, err)
	}
	t.Active = active == 1
	return t, nil
}

// Payment is one attempt to pay for a tariff.
type Payment struct {
	ID                int64
	UserID            string
	TariffCode        string
	GrantedPhase      string // level orders: the phase this payment opens
	AmountKopecks     int64
	Provider          string
	ProviderPaymentID string
	Status            string
	ReceiptContact    string
	Test              bool
	CreatedAt         time.Time
	PaidAt            time.Time
}

const paymentCols = `id, user_id, tariff_code, granted_phase, amount_kopecks, provider,
	provider_payment_id, status, receipt_contact, test, created_at, paid_at`

func scanPayment(sc interface{ Scan(...any) error }) (Payment, error) {
	var p Payment
	var test int
	var created, paid string
	if err := sc.Scan(&p.ID, &p.UserID, &p.TariffCode, &p.GrantedPhase, &p.AmountKopecks, &p.Provider,
		&p.ProviderPaymentID, &p.Status, &p.ReceiptContact, &test, &created, &paid); err != nil {
		return Payment{}, err
	}
	p.Test = test == 1
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	p.PaidAt, _ = time.Parse(time.RFC3339, paid)
	return p, nil
}

// CreatePayment records a new pending payment and returns its id.
func (s *Store) CreatePayment(p Payment, now time.Time) (int64, error) {
	var id int64
	err := s.db.QueryRow(`INSERT INTO payments (user_id, tariff_code, granted_phase, amount_kopecks,
		provider, provider_payment_id, status, receipt_contact, test, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		p.UserID, p.TariffCode, p.GrantedPhase, p.AmountKopecks, p.Provider, p.ProviderPaymentID,
		PaymentPending, p.ReceiptContact, boolInt(p.Test), now.UTC().Format(time.RFC3339)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create payment: %w", err)
	}
	return id, nil
}

// SetPaymentProviderID links a payment to the provider's id for it, once the
// provider has answered.
func (s *Store) SetPaymentProviderID(id int64, providerID string, test bool) error {
	if _, err := s.db.Exec(`UPDATE payments SET provider_payment_id = ?, test = ? WHERE id = ?`,
		providerID, boolInt(test), id); err != nil {
		return fmt.Errorf("set payment provider id: %w", err)
	}
	return nil
}

// PaymentByID reads one payment.
func (s *Store) PaymentByID(id int64) (Payment, error) {
	p, err := scanPayment(s.db.QueryRow(`SELECT `+paymentCols+` FROM payments WHERE id = ?`, id))
	if err != nil {
		return Payment{}, fmt.Errorf("payment %d: %w", id, err)
	}
	return p, nil
}

// PaymentByProviderID finds the payment a provider notification is about.
func (s *Store) PaymentByProviderID(provider, providerID string) (Payment, error) {
	p, err := scanPayment(s.db.QueryRow(`SELECT `+paymentCols+` FROM payments
		WHERE provider = ? AND provider_payment_id = ?`, provider, providerID))
	if err != nil {
		return Payment{}, fmt.Errorf("payment %s/%s: %w", provider, providerID, err)
	}
	return p, nil
}

// PendingPayments lists userID's payments still waiting for the provider's
// answer, newest first, created since `since`.
func (s *Store) PendingPayments(userID string, since time.Time) ([]Payment, error) {
	rows, err := s.db.Query(`SELECT `+paymentCols+` FROM payments
		WHERE user_id = ? AND status = ? AND provider_payment_id <> '' AND created_at >= ?
		ORDER BY id DESC LIMIT 5`, userID, PaymentPending, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("pending payments: %w", err)
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// HasPaidLevel reports whether userID has ever paid for a level: after the
// first one the first-purchase price is gone.
func (s *Store) HasPaidLevel(userID string) (bool, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM payments WHERE user_id = ? AND tariff_code = ? AND status = ?`,
		userID, TariffLevel, PaymentSucceeded).Scan(&n); err != nil {
		return false, fmt.Errorf("has paid level: %w", err)
	}
	return n > 0, nil
}

// LastLevelPaymentAt is when userID last paid for a level, ok=false when they
// never have. A payment made through a test shop counts like any other: it is the
// same flow, and a tester should meet the limit a customer would.
func (s *Store) LastLevelPaymentAt(userID string) (t time.Time, ok bool, err error) {
	var paid sql.NullString
	if err := s.db.QueryRow(`SELECT MAX(paid_at) FROM payments
		WHERE user_id = ? AND tariff_code = ? AND status = ? AND paid_at <> ''`,
		userID, TariffLevel, PaymentSucceeded).Scan(&paid); err != nil {
		return time.Time{}, false, fmt.Errorf("last level payment: %w", err)
	}
	if !paid.Valid || paid.String == "" {
		return time.Time{}, false, nil
	}
	t, perr := time.Parse(time.RFC3339, paid.String)
	if perr != nil {
		return time.Time{}, false, nil
	}
	return t, true, nil
}

// PaymentBonus is the in-app currency a paid payment credited (the topup entry
// keyed by it), 0 when it credited none.
func (s *Store) PaymentBonus(paymentID int64) (int64, error) {
	var amount int64
	err := s.db.QueryRow(`SELECT amount FROM currency_ledger WHERE idempotency_key = ?`,
		"payment:"+strconv.FormatInt(paymentID, 10)).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("payment bonus: %w", err)
	}
	return amount, nil
}

// CancelPayment marks a pending payment as canceled. It never touches one that
// has already succeeded.
func (s *Store) CancelPayment(id int64) error {
	if _, err := s.db.Exec(`UPDATE payments SET status = ? WHERE id = ? AND status = ?`,
		PaymentCanceled, id, PaymentPending); err != nil {
		return fmt.Errorf("cancel payment: %w", err)
	}
	return nil
}

// CompletePayment turns a pending payment into what was bought, in one
// transaction: the status flips and the entitlement is granted together, or
// neither happens. It reports whether THIS call did it, so a notification and a
// return-page check arriving together grant once. A payment that is not pending
// (already done, or canceled) is left alone.
//
// A level opens its phase for good. A year ends period_days after now, or after
// the end of a year that is still running, so paying early never wastes days.
func (s *Store) CompletePayment(id int64, now time.Time) (granted bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("complete payment: begin: %w", err)
	}
	defer tx.Rollback()

	iso := now.UTC().Format(time.RFC3339)
	res, err := tx.Exec(`UPDATE payments SET status = ?, paid_at = ? WHERE id = ? AND status = ?`,
		PaymentSucceeded, iso, id, PaymentPending)
	if err != nil {
		return false, fmt.Errorf("complete payment: mark: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}

	var userID, code, phase string
	if err := tx.QueryRow(`SELECT user_id, tariff_code, granted_phase FROM payments WHERE id = ?`, id).
		Scan(&userID, &code, &phase); err != nil {
		return false, fmt.Errorf("complete payment: read: %w", err)
	}

	switch code {
	case TariffLevel:
		// An order made without a level is a credit: the learner picks the level
		// after paying (ClaimLevel). Only an order that fixed its level, as the old
		// ones did, opens it here.
		if phase != "" {
			if err := grantLevelTx(tx, userID, phase, iso); err != nil {
				return false, fmt.Errorf("complete payment: grant level: %w", err)
			}
		}
	case TariffYear:
		var days int
		if err := tx.QueryRow(`SELECT period_days FROM tariffs WHERE code = ?`, TariffYear).Scan(&days); err != nil {
			return false, fmt.Errorf("complete payment: year length: %w", err)
		}
		base := now.UTC()
		var cur string
		err := tx.QueryRow(`SELECT expires_at FROM user_entitlements WHERE user_id = ? AND kind = ? AND ref = ?`,
			userID, AllAccessKind, AllAccessRef).Scan(&cur)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("complete payment: read year: %w", err)
		}
		if t, perr := time.Parse(time.RFC3339, cur); perr == nil && t.After(base) {
			base = t
		}
		until := base.AddDate(0, 0, days).UTC().Format(time.RFC3339)
		if _, err := tx.Exec(`INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at, expires_at)
			VALUES (?, ?, ?, 1, ?, ?)
			ON CONFLICT (user_id, kind, ref) DO UPDATE SET qty = 1, updated_at = ?, expires_at = ?`,
			userID, AllAccessKind, AllAccessRef, iso, until, iso, until); err != nil {
			return false, fmt.Errorf("complete payment: grant year: %w", err)
		}
	default:
		return false, fmt.Errorf("complete payment: unknown tariff %q", code)
	}

	// The tariff's bonus currency, in the same transaction as the grant: the
	// learner never has the level without the зёрнышки or the other way round. The
	// key makes a second run of this payment credit nothing (CompletePayment is
	// already idempotent through the status flip; this is the second lock).
	var bonus int64
	var title string
	if err := tx.QueryRow(`SELECT bonus_currency, title FROM tariffs WHERE code = ?`, code).Scan(&bonus, &title); err != nil {
		return false, fmt.Errorf("complete payment: bonus: %w", err)
	}
	if bonus > 0 {
		_, err := addLedgerEntryTx(tx, LedgerEntry{
			UserID: userID, Amount: bonus, Kind: LedgerTopup, Ref: strconv.FormatInt(id, 10),
			IdempotencyKey: "payment:" + strconv.FormatInt(id, 10),
			Comment:        "Бонус за оплату: " + title,
		}, now)
		if err != nil && !errors.Is(err, ErrDuplicateEntry) {
			return false, fmt.Errorf("complete payment: credit bonus: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("complete payment: commit: %w", err)
	}
	return true, nil
}

// grantLevelTx opens one level for good.
func grantLevelTx(tx *dbtx, userID, phase, iso string) error {
	_, err := tx.Exec(`INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at)
		VALUES (?, 'phase_unlock', ?, 1, ?)
		ON CONFLICT (user_id, kind, ref) DO UPDATE SET qty = 1, updated_at = ?`,
		userID, phase, iso, iso)
	return err
}

// UnclaimedLevelPayments are the learner's paid level orders that have not been
// spent on a level yet (the credits), oldest first.
func (s *Store) UnclaimedLevelPayments(userID string) ([]Payment, error) {
	rows, err := s.db.Query(`SELECT `+paymentCols+` FROM payments
		WHERE user_id = ? AND tariff_code = ? AND status = ? AND granted_phase = ''
		ORDER BY paid_at, id`, userID, TariffLevel, PaymentSucceeded)
	if err != nil {
		return nil, fmt.Errorf("unclaimed level payments: %w", err)
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("unclaimed level payments: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ClaimLevel spends a paid level credit on phase: the payment now names the level
// and the learner owns it. It happens once. Asking again for the same level is not
// an error (a second tap); another level on a spent credit is ErrCreditSpent.
// Whether phase is a level that may be opened is the caller's check: the store does
// not know the course.
func (s *Store) ClaimLevel(userID string, paymentID int64, phase string, now time.Time) error {
	if phase == "" {
		return ErrNotClaimable
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("claim level: begin: %w", err)
	}
	defer tx.Rollback()

	var code, status, granted string
	err = tx.QueryRow(`SELECT tariff_code, status, granted_phase FROM payments WHERE id = ? AND user_id = ?`,
		paymentID, userID).Scan(&code, &status, &granted)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotClaimable
	}
	if err != nil {
		return fmt.Errorf("claim level: read: %w", err)
	}
	if code != TariffLevel || status != PaymentSucceeded {
		return ErrNotClaimable
	}
	if granted == phase {
		return nil
	}
	if granted != "" {
		return ErrCreditSpent
	}
	res, err := tx.Exec(`UPDATE payments SET granted_phase = ? WHERE id = ? AND granted_phase = ''`, phase, paymentID)
	if err != nil {
		return fmt.Errorf("claim level: mark: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Somebody spent it between our read and our write. The same level is the
		// same answer as a second tap; another one is a refusal.
		var now string
		if err := tx.QueryRow(`SELECT granted_phase FROM payments WHERE id = ?`, paymentID).Scan(&now); err == nil && now == phase {
			return nil
		}
		return ErrCreditSpent
	}
	if err := grantLevelTx(tx, userID, phase, now.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("claim level: grant: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("claim level: commit: %w", err)
	}
	return nil
}

// AllAccessUntil reports when the learner's running year ends. ok is false when
// there is none, or it has already ended.
func (u *UserStore) AllAccessUntil(now time.Time) (until time.Time, ok bool, err error) {
	var s string
	err = u.db.QueryRow(`SELECT expires_at FROM user_entitlements WHERE user_id = ? AND kind = ? AND ref = ?`,
		u.user, AllAccessKind, AllAccessRef).Scan(&s)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("all access: %w", err)
	}
	t, perr := time.Parse(time.RFC3339, s)
	if perr != nil || !t.After(now) {
		return time.Time{}, false, nil
	}
	return t, true, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
