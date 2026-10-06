package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/economy"
)

var (
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrAlreadyOwned       = errors.New("already owned")
	ErrProductUnavailable = errors.New("product unavailable")
	ErrPromoInvalid       = errors.New("promo code invalid")
	ErrNoEntitlement      = errors.New("no entitlement")
)

const productCols = `id, kind, ref, title, description, price, discount_percent,
	discount_from, discount_to, grant_qty, active, sort_order`

func scanProduct(sc interface{ Scan(...any) error }) (economy.Product, error) {
	var p economy.Product
	var active int
	err := sc.Scan(&p.ID, &p.Kind, &p.Ref, &p.Title, &p.Description, &p.Price,
		&p.DiscountPercent, &p.DiscountFrom, &p.DiscountTo, &p.GrantQty, &active, &p.SortOrder)
	p.Active = active == 1
	return p, err
}

// ListProducts returns the catalogue in display order.
func (s *Store) ListProducts(activeOnly bool) ([]economy.Product, error) {
	q := `SELECT ` + productCols + ` FROM products`
	if activeOnly {
		q += ` WHERE active = 1`
	}
	q += ` ORDER BY sort_order ASC, id ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var out []economy.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Entitlements returns what the user owns, keyed "kind:ref". A consumable that
// has been spent down to nothing keeps its row, so a zero here means "had some
// once", which is why callers must compare against 0 rather than test presence.
func (u *UserStore) Entitlements() (map[string]int64, error) {
	rows, err := u.db.Query(`SELECT kind, ref, qty FROM user_entitlements WHERE user_id = ?`, u.user)
	if err != nil {
		return nil, fmt.Errorf("entitlements: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var kind, ref string
		var qty int64
		if err := rows.Scan(&kind, &ref, &qty); err != nil {
			return nil, fmt.Errorf("scan entitlement: %w", err)
		}
		out[kind+":"+ref] = qty
	}
	return out, rows.Err()
}

// purchaseBalanceHook, when non-nil, runs inside Purchase between reading the
// balance and deciding whether it covers the price. It exists so a test can
// park one transaction precisely in the window the user row lock closes, and
// prove the lock is load-bearing rather than hope two goroutines interleave in
// a sub-millisecond gap — which, measured, they do not. Always nil in
// production; only the concurrency test writes it.
var purchaseBalanceHook func()

// PurchaseResult reports what was actually charged.
type PurchaseResult struct {
	LedgerID int64
	Paid     int64
	Applied  string // "sale", "promo" or ""
	// Replayed means this call matched an earlier purchase's idempotency key:
	// nothing was charged or granted this time, and LedgerID and Paid come
	// from the original. Applied is empty on a replay, since the ledger does
	// not record which discount was used.
	Replayed bool
}

// Purchase charges the effective price and grants the product, atomically.
//
// idemKey makes a retry safe. The same key from the same user for the same
// product returns the original purchase with Replayed set instead of charging
// again, which is what a client that lost the response needs: buying three
// repairs is three purchases, but one purchase arriving three times is one. An
// empty key means "this is definitely a new purchase" and gets a generated,
// unique one — convenient for tests and for callers that have no request id,
// but it gives up retry protection, so the HTTP layer should always pass a key.
//
// The user's row is locked first so two concurrent purchases cannot both see
// a sufficient balance and both debit it. On SQLite writers are serialised
// anyway, so the lock is a no-op there.
func (u *UserStore) Purchase(productID int64, promoCode, idemKey string, now time.Time) (PurchaseResult, error) {
	tx, err := u.db.Begin()
	if err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: %w", err)
	}
	defer tx.Rollback()

	if err := lockUserTx(tx, u.user); err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: %w", err)
	}

	p, err := scanProduct(tx.QueryRow(`SELECT `+productCols+` FROM products WHERE id = ?`, productID))
	if errors.Is(err, sql.ErrNoRows) {
		return PurchaseResult{}, ErrProductUnavailable
	}
	if err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: load product: %w", err)
	}
	if !p.Active {
		return PurchaseResult{}, ErrProductUnavailable
	}

	key := idemKey
	if key == "" {
		key = strconv.FormatInt(now.UnixNano(), 10)
	}
	ledgerKey := "purchase:" + u.user + ":" + strconv.FormatInt(p.ID, 10) + ":" + key

	// A replay is settled before anything else is checked. The original
	// purchase already succeeded, so re-running the ownership and funds checks
	// would reject the retry of a permanent product as ErrAlreadyOwned — the
	// user would see a failure for a purchase that worked.
	if idemKey != "" {
		var id, amount int64
		err := tx.QueryRow(`SELECT id, amount FROM currency_ledger WHERE idempotency_key = ?`, ledgerKey).
			Scan(&id, &amount)
		switch {
		case err == nil:
			return PurchaseResult{LedgerID: id, Paid: -amount, Replayed: true}, nil
		case !errors.Is(err, sql.ErrNoRows):
			return PurchaseResult{}, fmt.Errorf("purchase: check replay: %w", err)
		}
	}

	if p.Permanent() {
		var owned int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM user_entitlements WHERE user_id = ? AND kind = ? AND ref = ?`,
			u.user, p.Kind, p.Ref).Scan(&owned); err != nil {
			return PurchaseResult{}, fmt.Errorf("purchase: check ownership: %w", err)
		}
		if owned > 0 {
			return PurchaseResult{}, ErrAlreadyOwned
		}
	}

	promoID, promoPercent, err := validatePromoTx(tx, u.user, promoCode, p.ID, now)
	if err != nil {
		return PurchaseResult{}, err
	}

	paid, applied := economy.EffectivePrice(p.Price, p.SalePercent(now), promoPercent)

	bal, err := balanceTx(tx, u.user)
	if err != nil {
		return PurchaseResult{}, err
	}
	if purchaseBalanceHook != nil {
		purchaseBalanceHook()
	}
	if bal < paid {
		return PurchaseResult{}, ErrInsufficientFunds
	}

	// A free purchase writes no ledger row — the ledger records movements of
	// currency, and nothing moved — so LedgerID stays 0 and any redemption
	// below records 0 too. promo_redemptions.ledger_id has no foreign key, so
	// 0 is readable as "this redemption cost nothing".
	var ledgerID int64
	if paid > 0 {
		ledgerID, err = addLedgerEntryTx(tx, LedgerEntry{
			UserID:         u.user,
			Amount:         -paid,
			Kind:           "purchase",
			Ref:            strconv.FormatInt(p.ID, 10),
			IdempotencyKey: ledgerKey,
		}, now)
		if err != nil {
			return PurchaseResult{}, err
		}
	}

	grant := int64(p.GrantQty)
	if p.Permanent() {
		grant = 1
	}
	iso := now.UTC().Format(time.RFC3339)
	if _, err := tx.Exec(`INSERT INTO user_entitlements (user_id, kind, ref, qty, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id, kind, ref) DO UPDATE SET
			qty = user_entitlements.qty + ?, updated_at = ?`,
		u.user, p.Kind, p.Ref, grant, iso, grant, iso); err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: grant: %w", err)
	}

	// Only a code that actually won the comparison is spent. Recording a
	// redemption for a code that lost to a better sale would burn a
	// one-per-user code the buyer never got anything for.
	if promoID != 0 && applied == "promo" {
		if _, err := tx.Exec(`INSERT INTO promo_redemptions (promo_code_id, user_id, product_id, ledger_id, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			promoID, u.user, p.ID, ledgerID, iso); err != nil {
			return PurchaseResult{}, fmt.Errorf("purchase: record redemption: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return PurchaseResult{}, fmt.Errorf("purchase: commit: %w", err)
	}
	return PurchaseResult{LedgerID: ledgerID, Paid: paid, Applied: applied}, nil
}

// validatePromoTx resolves a code to its id and percent, or returns
// ErrPromoInvalid. An empty code is not an error — it means no code.
//
// Codes are matched upper-cased and trimmed, so the admin panel must store
// them upper-cased: promo_codes.code is uniquely indexed and compared exactly,
// so a code saved as "osen" could never be redeemed.
func validatePromoTx(tx *dbtx, userID, code string, productID int64, now time.Time) (int64, int, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, 0, nil
	}
	var (
		id                          int64
		percent, maxRedeem, maxUser int
		scope, from, to             string
		active                      int
	)
	err := tx.QueryRow(`SELECT id, discount_percent, scope, valid_from, valid_to,
		max_redemptions, max_per_user, active FROM promo_codes WHERE code = ?`, code).
		Scan(&id, &percent, &scope, &from, &to, &maxRedeem, &maxUser, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrPromoInvalid
	}
	if err != nil {
		return 0, 0, fmt.Errorf("promo code: %w", err)
	}
	if active != 1 {
		return 0, 0, ErrPromoInvalid
	}
	day := now.Format("2006-01-02")
	if (from != "" && day < from) || (to != "" && day > to) {
		return 0, 0, ErrPromoInvalid
	}
	if scope == "products" {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM promo_code_products WHERE promo_code_id = ? AND product_id = ?`,
			id, productID).Scan(&n); err != nil {
			return 0, 0, fmt.Errorf("promo scope: %w", err)
		}
		if n == 0 {
			return 0, 0, ErrPromoInvalid
		}
	}
	if maxRedeem > 0 {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM promo_redemptions WHERE promo_code_id = ?`, id).Scan(&n); err != nil {
			return 0, 0, fmt.Errorf("promo redemptions: %w", err)
		}
		if n >= maxRedeem {
			return 0, 0, ErrPromoInvalid
		}
	}
	if maxUser > 0 {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM promo_redemptions WHERE promo_code_id = ? AND user_id = ?`,
			id, userID).Scan(&n); err != nil {
			return 0, 0, fmt.Errorf("promo redemptions per user: %w", err)
		}
		if n >= maxUser {
			return 0, 0, ErrPromoInvalid
		}
	}
	return id, percent, nil
}

// SpendStreakRepair consumes one streak_repair unit and repairs the day. The
// two halves commit together: a repair that fails its window check must not
// eat the consumable.
func (u *UserStore) SpendStreakRepair(day string, now time.Time) error {
	tx, err := u.db.Begin()
	if err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}
	defer tx.Rollback()

	if err := lockUserTx(tx, u.user); err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}

	res, err := tx.Exec(`UPDATE user_entitlements SET qty = qty - 1, updated_at = ?
		WHERE user_id = ? AND kind = ? AND ref = 'streak_repair' AND qty > 0`,
		now.UTC().Format(time.RFC3339), u.user, economy.ProductConsumable)
	if err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("spend repair: %w", err)
	}
	if n == 0 {
		return ErrNoEntitlement
	}

	if err := repairStreakTx(tx, u.user, day, now); err != nil {
		return err // the rollback puts the consumable back
	}
	return tx.Commit()
}
