package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

func ImportJSONToMySQL(dir string, target *MySQLRepository) error {
	for _, name := range dataFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("JSON import requires %s: %w", name, err)
		}
	}
	source, err := NewJSONRepository(dir)
	if err != nil {
		return err
	}
	var store Store
	if err := source.View(func(current *Store) error { store = *current; return nil }); err != nil {
		return err
	}
	// Demo staff passwords are plaintext; live staff must be provisioned separately.
	store.StaffUsers = nil
	for i := range store.AuditLogs {
		store.AuditLogs[i].Summary = maskPhoneText(store.AuditLogs[i].Summary)
		store.AuditLogs[i].Before = maskAuditMap(store.AuditLogs[i].Before)
		store.AuditLogs[i].After = maskAuditMap(store.AuditLogs[i].After)
	}
	var occupied bool
	if err := target.View(func(current *Store) error {
		for _, table := range mysqlTables {
			if reflect.ValueOf(table.get(current)).Len() > 0 {
				occupied = true
				break
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if occupied {
		return errors.New("MySQL target is not empty; import refused")
	}
	return target.Replace(store)
}
