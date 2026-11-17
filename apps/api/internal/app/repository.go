package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	Members           []Member              `json:"members"`
	StaffUsers        []StaffAccount        `json:"staff_users"`
	Orders            []BookingOrder        `json:"orders"`
	CouponTemplates   []CouponTemplate      `json:"coupon_templates"`
	Coupons           []Coupon              `json:"coupon_instances"`
	WalletAccounts    []WalletAccountRecord `json:"wallet_accounts"`
	WalletLedgers     []WalletLedger        `json:"wallet_ledgers"`
	ReferralRelations []ReferralRelation    `json:"referral_relations"`
	ReferralRewards   []ReferralReward      `json:"referral_rewards"`
	AuditLogs         []AuditLog            `json:"audit_logs"`
	Notifications     []Notification        `json:"notifications"`
	RuleConfig        RuleConfig            `json:"rule_config"`
}

type Repository interface {
	View(func(*Store) error) error
	Update(func(*Store) error) error
	Replace(Store) error
}

type JSONRepository struct {
	dir   string
	mu    sync.Mutex
	store Store
}

func NewJSONRepository(dir string) (*JSONRepository, error) {
	repo := &JSONRepository{dir: dir}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := repo.loadOrSeed(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *JSONRepository) View(fn func(*Store) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(&r.store)
}

func (r *JSONRepository) Update(fn func(*Store) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := fn(&r.store); err != nil {
		return err
	}
	return r.writeAllLocked()
}

func (r *JSONRepository) Replace(store Store) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store = store
	return r.writeAllLocked()
}

func (r *JSONRepository) loadOrSeed() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.allFilesExistLocked() {
		r.store = SeedStore()
		return r.writeAllLocked()
	}

	readers := []struct {
		name string
		dest interface{}
	}{
		{"members.json", &r.store.Members},
		{"staff_users.json", &r.store.StaffUsers},
		{"orders.json", &r.store.Orders},
		{"coupon_templates.json", &r.store.CouponTemplates},
		{"coupon_instances.json", &r.store.Coupons},
		{"wallet_accounts.json", &r.store.WalletAccounts},
		{"wallet_ledgers.json", &r.store.WalletLedgers},
		{"referral_relations.json", &r.store.ReferralRelations},
		{"referral_rewards.json", &r.store.ReferralRewards},
		{"audit_logs.json", &r.store.AuditLogs},
		{"notifications.json", &r.store.Notifications},
		{"rule_config.json", &r.store.RuleConfig},
	}

	for _, reader := range readers {
		raw, err := os.ReadFile(filepath.Join(r.dir, reader.name))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, reader.dest); err != nil {
			return err
		}
	}
	return nil
}

func (r *JSONRepository) allFilesExistLocked() bool {
	for _, name := range dataFileNames {
		if _, err := os.Stat(filepath.Join(r.dir, name)); err != nil {
			return false
		}
	}
	return true
}

func (r *JSONRepository) writeAllLocked() error {
	writers := []struct {
		name string
		data interface{}
	}{
		{"members.json", r.store.Members},
		{"staff_users.json", r.store.StaffUsers},
		{"orders.json", r.store.Orders},
		{"coupon_templates.json", r.store.CouponTemplates},
		{"coupon_instances.json", r.store.Coupons},
		{"wallet_accounts.json", r.store.WalletAccounts},
		{"wallet_ledgers.json", r.store.WalletLedgers},
		{"referral_relations.json", r.store.ReferralRelations},
		{"referral_rewards.json", r.store.ReferralRewards},
		{"audit_logs.json", r.store.AuditLogs},
		{"notifications.json", r.store.Notifications},
		{"rule_config.json", r.store.RuleConfig},
	}

	for _, writer := range writers {
		if err := writeJSONFile(filepath.Join(r.dir, writer.name), writer.data); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONFile(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return nil
}

var dataFileNames = []string{
	"members.json",
	"staff_users.json",
	"orders.json",
	"coupon_templates.json",
	"coupon_instances.json",
	"wallet_accounts.json",
	"wallet_ledgers.json",
	"referral_relations.json",
	"referral_rewards.json",
	"audit_logs.json",
	"notifications.json",
	"rule_config.json",
}

var errStopIteration = errors.New("stop iteration")
