package app

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed migrations/001_init.sql
var migrations embed.FS

type MySQLRepository struct{ DB *sql.DB }

func OpenMySQL(dsn string) (*MySQLRepository, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("MYSQL_DSN is required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &MySQLRepository{DB: db}, nil
}

func MigrateMySQL(ctx context.Context, db *sql.DB) error {
	raw, err := migrations.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	for _, statement := range strings.Split(string(raw), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("schema migration: %w", err)
		}
	}
	rule, err := json.Marshal(SeedStore().RuleConfig)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "INSERT IGNORE INTO rule_config (id, payload) VALUES (1, ?)", string(rule))
	if err != nil {
		return err
	}
	return nil
}

func CheckMySQLSchema(ctx context.Context, db *sql.DB) error {
	var version int
	return db.QueryRowContext(ctx, "SELECT version FROM schema_migrations WHERE version=1").Scan(&version)
}

var mysqlTables = []struct {
	name string
	get  func(*Store) any
	set  func(*Store, []json.RawMessage) error
	keys func(json.RawMessage) ([]any, error)
	cols string
}{
	{"members", func(s *Store) any { return s.Members }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.Members) }, memberKeys, "id, openid, phone"},
	{"staff_users", func(s *Store) any { return s.StaffUsers }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.StaffUsers) }, staffKeys, "id, username, password_hash"},
	{"orders", func(s *Store) any { return s.Orders }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.Orders) }, orderKeys, "id, member_id, status"},
	{"coupon_templates", func(s *Store) any { return s.CouponTemplates }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.CouponTemplates) }, idKeys, "id"},
	{"coupon_instances", func(s *Store) any { return s.Coupons }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.Coupons) }, couponKeys, "id, member_id, status"},
	{"wallet_accounts", func(s *Store) any { return s.WalletAccounts }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.WalletAccounts) }, walletKeys, "member_id"},
	{"wallet_ledgers", func(s *Store) any { return s.WalletLedgers }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.WalletLedgers) }, ledgerKeys, "id, member_id, order_id"},
	{"referral_relations", func(s *Store) any { return s.ReferralRelations }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.ReferralRelations) }, relationKeys, "id, referrer_member_id, referred_member_id"},
	{"referral_rewards", func(s *Store) any { return s.ReferralRewards }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.ReferralRewards) }, rewardKeys, "id, order_id"},
	{"audit_logs", func(s *Store) any { return s.AuditLogs }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.AuditLogs) }, auditKeys, "id, operator_staff_id, action, created_at"},
	{"notifications", func(s *Store) any { return s.Notifications }, func(s *Store, rows []json.RawMessage) error { return decodeRows(rows, &s.Notifications) }, notificationKeys, "id, member_id"},
}

func decodeRows[T any](rows []json.RawMessage, dest *[]T) error {
	*dest = make([]T, 0, len(rows))
	for _, row := range rows {
		var item T
		if err := json.Unmarshal(row, &item); err != nil {
			return err
		}
		*dest = append(*dest, item)
	}
	return nil
}

func extract(raw json.RawMessage, fields ...string) ([]any, error) {
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	values := make([]any, 0, len(fields))
	for _, field := range fields {
		v, _ := item[field].(string)
		if v == "" && (field == "phone" || field == "order_id") {
			values = append(values, nil)
		} else {
			values = append(values, v)
		}
	}
	return values, nil
}
func idKeys(r json.RawMessage) ([]any, error)     { return extract(r, "id") }
func memberKeys(r json.RawMessage) ([]any, error) { return extract(r, "id", "openid", "phone") }
func staffKeys(r json.RawMessage) ([]any, error)  { return extract(r, "id", "username", "password") }
func orderKeys(r json.RawMessage) ([]any, error)  { return extract(r, "id", "member_id", "status") }
func couponKeys(r json.RawMessage) ([]any, error) { return extract(r, "id", "member_id", "status") }
func walletKeys(r json.RawMessage) ([]any, error) { return extract(r, "member_id") }
func ledgerKeys(r json.RawMessage) ([]any, error) { return extract(r, "id", "member_id", "order_id") }
func relationKeys(r json.RawMessage) ([]any, error) {
	return extract(r, "id", "referrer_member_id", "referred_member_id")
}
func rewardKeys(r json.RawMessage) ([]any, error) { return extract(r, "id", "order_id") }
func auditKeys(r json.RawMessage) ([]any, error) {
	return extract(r, "id", "operator_staff_id", "action", "created_at")
}
func notificationKeys(r json.RawMessage) ([]any, error) { return extract(r, "id", "member_id") }

func (r *MySQLRepository) View(fn func(*Store) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store, err := loadStore(ctx, tx)
	if err != nil {
		return err
	}
	if err := fn(&store); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLRepository) Update(fn func(*Store) error) error {
	return r.updateWithTx(fn, nil)
}

func (r *MySQLRepository) updateWithTx(fn func(*Store) error, after func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := r.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var revision int64
	if err := tx.QueryRowContext(ctx, "SELECT revision FROM store_revision WHERE id=1 FOR UPDATE").Scan(&revision); err != nil {
		return err
	}
	store, err := loadStore(ctx, tx)
	if err != nil {
		return err
	}
	before, err := cloneStore(store)
	if err != nil {
		return err
	}
	if err := fn(&store); err != nil {
		return err
	}
	if err := saveStore(ctx, tx, &before, &store); err != nil {
		return err
	}
	if after != nil {
		if err := after(ctx, tx); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE store_revision SET revision=? WHERE id=1", revision+1); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLRepository) Replace(store Store) error {
	return r.Update(func(current *Store) error { *current = store; return nil })
}

func loadStore(ctx context.Context, tx *sql.Tx) (Store, error) {
	var store Store
	for _, table := range mysqlTables {
		rows, err := tx.QueryContext(ctx, "SELECT payload FROM "+table.name+" ORDER BY "+strings.Split(table.cols, ",")[0])
		if err != nil {
			return store, err
		}
		var raw []json.RawMessage
		for rows.Next() {
			var item []byte
			if err := rows.Scan(&item); err != nil {
				rows.Close()
				return store, err
			}
			raw = append(raw, append([]byte(nil), item...))
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return store, err
		}
		if err := table.set(&store, raw); err != nil {
			return store, err
		}
	}
	var raw []byte
	err := tx.QueryRowContext(ctx, "SELECT payload FROM rule_config WHERE id=1").Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &store.RuleConfig)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return store, err
	}
	return store, nil
}

func cloneStore(store Store) (Store, error) {
	raw, err := json.Marshal(store)
	if err != nil {
		return Store{}, err
	}
	var clone Store
	return clone, json.Unmarshal(raw, &clone)
}

func tableRows(value any, keyFn func(json.RawMessage) ([]any, error)) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	result := make(map[string]json.RawMessage, len(items))
	for _, item := range items {
		keys, err := keyFn(item)
		if err != nil {
			return nil, err
		}
		id, ok := keys[0].(string)
		if !ok || id == "" {
			return nil, errors.New("row has no primary key")
		}
		if _, exists := result[id]; exists {
			return nil, fmt.Errorf("duplicate primary key %s", id)
		}
		result[id] = item
	}
	return result, nil
}

func saveStore(ctx context.Context, tx *sql.Tx, before, store *Store) error {
	for _, table := range mysqlTables {
		oldRows, err := tableRows(table.get(before), table.keys)
		if err != nil {
			return err
		}
		newRows, err := tableRows(table.get(store), table.keys)
		if err != nil {
			return err
		}
		columns := strings.Split(table.cols, ", ")
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(columns)+1), ",")
		updates := make([]string, 0, len(columns))
		for _, col := range columns[1:] {
			updates = append(updates, col+"=?")
		}
		updates = append(updates, "payload=?")
		for id := range oldRows {
			if _, exists := newRows[id]; !exists {
				if _, err := tx.ExecContext(ctx, "DELETE FROM "+table.name+" WHERE "+columns[0]+"=?", id); err != nil {
					return err
				}
			}
		}
		for id, row := range newRows {
			old, exists := oldRows[id]
			if exists && reflect.DeepEqual(old, row) {
				continue
			}
			keys, err := table.keys(row)
			if err != nil {
				return err
			}
			var query string
			var args []any
			if exists {
				query = "UPDATE " + table.name + " SET " + strings.Join(updates, ", ") + " WHERE " + columns[0] + "=?"
				args = append(append([]any(nil), keys[1:]...), string(row), id)
			} else {
				query = "INSERT INTO " + table.name + " (" + table.cols + ", payload) VALUES (" + placeholders + ")"
				args = append(keys, string(row))
			}
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return fmt.Errorf("save %s: %w", table.name, err)
			}
		}
	}
	if !reflect.DeepEqual(before.RuleConfig, store.RuleConfig) {
		raw, err := json.Marshal(store.RuleConfig)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO rule_config (id, payload) VALUES (1, ?) ON DUPLICATE KEY UPDATE payload=VALUES(payload)", string(raw))
		if err != nil {
			return err
		}
	}
	return nil
}
