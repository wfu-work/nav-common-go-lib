package dbs

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	commonconfigs "github.com/wfu-work/nav-common-go-lib/configs"
	"gorm.io/gorm"
)

func TestSQLiteFileURI(t *testing.T) {
	for _, tc := range []struct {
		name, path, wantPath string
	}{
		{"windows", "D:/GNSS 数据/project #1/100% ready/nav.db", "/D:/GNSS 数据/project #1/100% ready/nav.db"},
		{"unix", "/var/lib/GNSS 数据/project #1/100% ready/nav.db", "/var/lib/GNSS 数据/project #1/100% ready/nav.db"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := url.Parse(sqliteFileURI(tc.path))
			if err != nil {
				t.Fatal(err)
			}
			if uri.Scheme != "file" || uri.Host != "" || uri.RawQuery != "" || uri.Fragment != "" {
				t.Fatalf("filesystem path became URI metadata: %+v", uri)
			}
			if uri.Path != tc.wantPath {
				t.Fatalf("URI path = %q, want %q", uri.Path, tc.wantPath)
			}
		})
	}
}

func TestGormSQLiteFile(t *testing.T) {
	for _, name := range []string{"absolute", "relative"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if name == "relative" {
				t.Chdir(root)
				root = "."
			}
			configuration := commonconfigs.Sqlite{GeneralDB: commonconfigs.GeneralDB{
				Path:   filepath.Join(root, "GNSS 数据 #1", "100% ready & nested"),
				Dbname: "hipnaos", LogMode: "silent",
			}}
			original := configuration
			want := filepath.Join(configuration.Path, "hipnaos.db")
			db, err := GormSqliteByConfig(configuration)
			if err != nil {
				t.Fatal(err)
			}
			pool, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = pool.Close() })
			if pool.Stats().MaxOpenConnections != 1 {
				t.Fatal("SQLite default connection limit changed")
			}
			var sequence int
			var name, actual string
			if err = pool.QueryRow("PRAGMA database_list").Scan(&sequence, &name, &actual); err != nil {
				t.Fatal(err)
			}
			wantFile, err := os.Stat(want)
			if err != nil {
				t.Fatal(err)
			}
			actualFile, err := os.Stat(actual)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(actualFile, wantFile) {
				t.Fatalf("opened %q instead of configured file %q", actual, want)
			}
			var journal string
			var timeout int
			if err = pool.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
				t.Fatalf("WAL not enabled: %q, %v", journal, err)
			}
			if err = pool.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil || timeout != 5000 {
				t.Fatalf("busy timeout changed: %d, %v", timeout, err)
			}
			if err = db.Exec("CREATE TABLE startup_probe (value TEXT)").Error; err != nil {
				t.Fatal(err)
			}
			if err = db.Exec("INSERT INTO startup_probe VALUES (?)", "sqlite path works").Error; err != nil {
				t.Fatal(err)
			}
			if err = pool.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := GormSqliteByConfig(configuration)
			if err != nil {
				t.Fatal(err)
			}
			reopenedPool, err := reopened.DB()
			if err != nil {
				t.Fatal(err)
			}
			defer reopenedPool.Close()
			var value string
			if err = reopenedPool.QueryRow("SELECT value FROM startup_probe").Scan(&value); err != nil || value != "sqlite path works" {
				t.Fatalf("database was not persisted at its configured path: %q, %v", value, err)
			}
			if configuration != original {
				t.Fatal("opening SQLite changed the persistent configuration")
			}
			if _, err = os.Stat(want); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGormSQLiteMemoryAndURI(t *testing.T) {
	open := func(name string) *gorm.DB {
		t.Helper()
		db, err := GormSqliteByConfig(commonconfigs.Sqlite{GeneralDB: commonconfigs.GeneralDB{
			Dbname: name, LogMode: "silent", MaxOpenConns: 2,
		}})
		if err != nil {
			t.Fatal(err)
		}
		pool, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = pool.Close() })
		if pool.Stats().MaxOpenConnections != 2 {
			t.Fatal("explicit connection limit was not retained")
		}
		return db
	}

	first, second := open(":memory:"), open(":memory:")
	if err := first.Exec("CREATE TABLE private_probe (value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	var count int
	if err := second.Raw("SELECT count(*) FROM sqlite_master WHERE name='private_probe'").Scan(&count).Error; err != nil || count != 0 {
		t.Fatalf("separate in-memory databases unexpectedly shared state: %d, %v", count, err)
	}

	uri := "file:" + url.PathEscape(t.Name()) + "?mode=memory&cache=shared&_foreign_keys=on"
	shared, peer := open(uri), open(uri)
	if err := shared.Exec("CREATE TABLE shared_probe (value TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := peer.Raw("SELECT count(*) FROM sqlite_master WHERE name='shared_probe'").Scan(&count).Error; err != nil || count != 1 {
		t.Fatalf("explicit shared-memory URI was not retained: %d, %v", count, err)
	}
	var foreignKeys int
	if err := peer.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil || foreignKeys != 1 {
		t.Fatalf("explicit URI query options were lost: %d, %v", foreignKeys, err)
	}
}
