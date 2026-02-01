// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// package forgery provides integration helpers for Forgejo
package forgery

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"testing"

	"forgejo.org/cmd"
	"forgejo.org/models/db"
	"forgejo.org/models/system"
	"forgejo.org/modules/auth/password/hash"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/setting/config"
	"forgejo.org/modules/translation"
	"forgejo.org/routers"
)

// Instance represents a test Forgejo instance
type Instance struct {
	Server *httptest.Server
	URL    url.URL
}

// Session returns an anonymous (unauthenticated) session.
func (i Instance) Session() Session {
	client := *i.Server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return Session{
		Client: client,
		URL:    i.URL,
	}
}

var rwlock sync.RWMutex

var cleanups []func() // nil means not initialized

// WrapTestMain called with m.Run within TestMain ensures that the temporary files
// of the shared forgejo instance are cleaned up after the tests.
// If the test binary is called from a git hook, the main app will be run instead.
func WrapTestMain(run func() int) {
	if gd := os.Getenv("GIT_DIR"); gd != "" {
		// log.Println("testf: GIT_DIR is set: running as app", os.Args)
		app := cmd.NewMainApp("test-version", "testf")
		if err := cmd.RunMainApp(app, os.Args...); err != nil {
			log.Fatal(err) // should never happen since RunMainApp exits on error
		}
		os.Exit(0)
	}

	rwlock.Lock()
	if cleanups != nil {
		panic("TestMainCleanup should not be called multiple times")
	}
	cleanups = []func(){}
	rwlock.Unlock()

	// note that panic in m.Run cannot be caught
	// (remember to cleanup your tmp folder regularly...)
	// https://go.dev/issue/37206
	_ = run() // no need to call os.Exit

	rwlock.Lock()
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
	rwlock.Unlock()
}

// SharedInstance returns a shared forgejo instance.
// t.Parallel() should be called before, so that other test can run at the same time.
// A read-lock will be held until the test is done (cleaned-up).
// Be sure to only make contained changes (with dedicated users/orgs/repos), to not disturb other parallel running tests.
func SharedInstance(t testing.TB) Instance {
	t.Helper()

	fgi, err := instance()
	if err != nil {
		t.Fatal(err)
	}

	rwlock.RLock()
	t.Cleanup(rwlock.RUnlock)
	return fgi
}

var instance = sync.OnceValues(func() (Instance, error) {
	rwlock.Lock()
	defer rwlock.Unlock()
	if cleanups == nil {
		return Instance{}, errors.New("TestMainCleanup has not been called in TestMain")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cleanups = append(cleanups, cancel)

	var err error
	storageRoot := os.Getenv("TEST_STORAGE_ROOT")
	if storageRoot != "" {
		storageRoot, err = filepath.Abs(storageRoot)
		if err != nil {
			return Instance{}, fmt.Errorf("TEST_STORAGE_ROOT.Abs: %w", err)
		}
		if err := os.RemoveAll(storageRoot); err != nil {
			log.Println("could not cleanup TEST_STORAGE_ROOT", storageRoot, err)
		}
		if err := os.MkdirAll(storageRoot, 0o700); err != nil {
			return Instance{}, fmt.Errorf("TEST_STORAGE_ROOT.MkdirAll: %w", err)
		}
		// no cleanup, so that the folder can be inspected after the test
	} else {
		storageRoot, err = os.MkdirTemp("", "forgejo-test-")
		if err != nil {
			return Instance{}, fmt.Errorf("storageRoot: %w", err)
		}
		cleanups = append(cleanups, func() {
			if err := os.RemoveAll(storageRoot); err != nil {
				log.Println("could not cleanup", storageRoot, err)
			}
		})
	}

	setting.CustomConf = filepath.Join(storageRoot, "forgejo.ini")
	setting.CfgProvider, err = setting.NewConfigProviderFromFile(setting.CustomConf)
	if err != nil {
		return Instance{}, fmt.Errorf("setting.NewConfigProviderFromFile: %w", err)
	}

	rootFolder, err := findGoModFolder()
	if err != nil {
		return Instance{}, err
	}

	s := httptest.NewUnstartedServer(nil)
	{
		// setting
		setting.StaticRootPath = rootFolder
		// TODO: support minio storage
		setting.AppWorkPath = storageRoot

		_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "true")
		setting.CfgProvider.Section("git").Key("HOME_PATH").SetValue(filepath.Join(storageRoot, "git-home"))

		host, port, err := net.SplitHostPort(s.Listener.Addr().String())
		if err != nil {
			return Instance{}, err
		}
		setting.CfgProvider.Section("server").Key("HTTP_PORT").SetValue(port)
		setting.CfgProvider.Section("server").Key("HTTP_ADDR").SetValue(host)
		setting.CfgProvider.Section("security").Key("INSTALL_LOCK").SetValue("true")

		// use sqlite :memory: by default
		setting.CfgProvider.Section("database").Key("DB_TYPE").SetValue(cmp.Or(os.Getenv("TEST_DB_TYPE"), "sqlite3"))
		setting.CfgProvider.Section("database").Key("PATH").SetValue(cmp.Or(os.Getenv("TEST_DB_PATH"), ":memory:"))

		setting.CfgProvider.Section("database").Key("HOST").SetValue(os.Getenv("TEST_DB_HOST"))
		setting.CfgProvider.Section("database").Key("NAME").SetValue(os.Getenv("TEST_DB_NAME"))
		setting.CfgProvider.Section("database").Key("USER").SetValue(os.Getenv("TEST_DB_USER"))
		setting.CfgProvider.Section("database").Key("PASSWD").SetValue(os.Getenv("TEST_DB_PASSWD"))
		setting.CfgProvider.Section("database").Key("SCHEMA").SetValue(os.Getenv("TEST_DB_SCHEMA"))

		// ensure git push updates are immediately seen (repo.IsEmpty for instance)
		setting.CfgProvider.Section("queue.push_update").Key("TYPE").SetValue("immediate")

		// register the dummy hash algorithm function used in the test fixtures
		_ = hash.Register("dummy", hash.NewDummyHasher)
		setting.PasswordHashAlgo, _ = hash.SetDefaultPasswordHashAlgorithm("dummy")

		// TODO: faster oauth init with saved key
		// oauth2.Init
		// actions.InitOIDC

		setting.LoadCommonSettings()
		setting.LoadDBSetting()

		if err := setting.CfgProvider.Save(); err != nil {
			return Instance{}, err
		}

		// buf, _ := os.ReadFile(setting.CustomConf)
		// fmt.Println(string(buf))
	}

	setting.IsProd = true // disable hot-reloading of html templates and translations (hacky)
	{
		// i18n
		setting.Names = []string{"english"}
		setting.Langs = []string{"en-US"}
		translation.InitLocales(ctx)
	}
	{
		setting.Database.LogSQL = true
		setting.IsInTesting = true
		if err := db.InitEngine(ctx); err != nil {
			return Instance{}, fmt.Errorf("db.InitEngine(%s, %s): %w", setting.Database.Type, setting.Database.Host, err)
		}

		// require.NoError(t, gitea_migrations.Migrate(masterEngine)) // requires git, likely not needed
		if err := db.SyncAllTables(); err != nil {
			return Instance{}, fmt.Errorf("db.SyncAllTables(): %w", err)
		}
	}
	{
		masterEngine, err := db.GetMasterEngine(db.GetEngine(db.DefaultContext))
		if err != nil {
			return Instance{}, fmt.Errorf("db.GetMasterEngine(%s, %s): %w", setting.Database.Type, setting.Database.Host, err)
		}
		cleanups = append(cleanups, func() {
			masterEngine.Close()
		})

		// Ideally there should be no global fixtures.
		// Let's see how far we can get without them...
		// if err := loadFixtures(masterEngine, rootFolder); err != nil {
		// 	return Instance{}, err
		// }
	}
	{
		// misc
		config.SetDynGetter(system.NewDatabaseDynKeyGetter())
		routers.InitWebInstalled(ctx)
	}

	s.Config.Handler = routers.NormalRoutes()
	s.Start()
	cleanups = append(cleanups, s.Close)

	u, err := url.Parse(s.URL)
	if err != nil {
		return Instance{}, fmt.Errorf("url.Parse(%s): %w", s.URL, err)
	}

	if os.Getenv("TEST_WAIT_BEFORE_CLEANUP") != "" {
		cleanups = append(cleanups, func() {
			log.Println("Test instance kept active for debugging (hit ctrl+c to cleanup and stop)", s.URL)
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			<-ctx.Done()
			stop()
		})
	}

	return Instance{
		Server: s,
		URL:    *u,
	}, nil
})

func findGoModFolder() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	searchDir := wd
	for searchDir != "" {
		if _, err := os.Stat(filepath.Join(searchDir, "go.mod")); err == nil {
			return searchDir, nil
		}
		if dir := filepath.Dir(searchDir); dir == searchDir {
			searchDir = "" // reaches the root of filesystem
		} else {
			searchDir = dir
		}
	}
	return "", fmt.Errorf("The tests should run in a Forgejo repository; could not find the 'go.mod' file within %s", wd)
}

/*
func loadFixtures(eng *xorm.Engine, rootFolder string) error {
	const fixtureExtension = ".yml"
	files, err := filepath.Glob(filepath.Join(rootFolder, "models", "fixtures", "*"+fixtureExtension))
	if err != nil {
		return fmt.Errorf("filepath.Glob(models/fixtures): %w", err)
	}
	slices.SortFunc(files, func(v1, v2 string) int {
		return db.TableNameInsertionOrderSortFunc(
			filepath.Base(strings.TrimSuffix(v1, fixtureExtension)),
			filepath.Base(strings.TrimSuffix(v2, fixtureExtension)),
		)
	})
	for _, f := range files {
		if err := insertFixture(eng, f); err != nil {
			return fmt.Errorf("insertFixture(%s): %w", f, err)
		}
	}
	if err := adjustAutoIncr(eng); err != nil {
		return err
	}
	return nil
}

func insertFixture(eng db.Engine, fixturePath string) error {
	f, err := os.Open(fixturePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var records []map[string]any
	err = yaml.NewDecoder(f).Decode(&records)
	if err != nil {
		return fmt.Errorf("yaml.Decode: %w", err)
	}

	table := filepath.Base(strings.TrimSuffix(f.Name(), filepath.Ext(f.Name())))
	for r, record := range records {
		columns := []string{}
		sqlValues := []string{}
		values := []any{}
		i := 1

		for key, value := range record {
			columns = append(columns, dbQuoteKeyword(key))

			switch v := value.(type) {
			case string:
				// Try to decode hex.
				if hexData, ok := strings.CutPrefix(v, "0x"); ok {
					value, err = hex.DecodeString(hexData)
					if err != nil {
						return fmt.Errorf("[%d] %s: 0x: %w", r, key, err)
					}
				}
			case []any:
				// Decode array.
				var bytes []byte
				bytes, err = json.Marshal(v)
				if err != nil {
					return fmt.Errorf("[%d] %s: json.Marshal: %w", r, key, err)
				}
				value = string(bytes)
			}

			values = append(values, value)

			sqlValues = append(sqlValues, dbPlaceholder(i))
			i++
		}
		statement := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			dbQuoteKeyword(table),
			strings.Join(columns, ", "),
			strings.Join(sqlValues, ", "))
		if _, err := eng.Exec(append([]any{statement}, values...)...,
		); err != nil {
			return fmt.Errorf("[%d]: insert: %w", r, err)
		}
	}
	return nil
}

func dbQuoteKeyword(keyword string) string {
	switch setting.Database.Type {
	case "sqlite3":
		return `"` + keyword + `"`
	case "mysql":
		return "`" + keyword + "`"
	case "postgres":
		parts := strings.Split(keyword, ".")
		for i, p := range parts {
			parts[i] = `"` + p + `"`
		}
		return strings.Join(parts, ".")
	default:
		return "invalid"
	}
}

// placeholder returns the placeholder string.
func dbPlaceholder(index int) string {
	if setting.Database.Type.IsPostgreSQL() {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func adjustAutoIncr(eng *xorm.Engine) error {
	if !setting.Database.Type.IsPostgreSQL() {
		return nil
	}
	// only needed for postgres
	results, err := eng.QueryString(`SELECT 'SELECT SETVAL(' ||
		quote_literal(quote_ident(PGT.schemaname) || '.' || quote_ident(S.relname)) ||
		', COALESCE(MAX(' ||quote_ident(C.attname)|| '), 1) ) FROM ' ||
		quote_ident(PGT.schemaname)|| '.'||quote_ident(T.relname)|| ';'
	 FROM pg_class AS S,
	      pg_depend AS D,
	      pg_class AS T,
	      pg_attribute AS C,
	      pg_tables AS PGT
	 WHERE S.relkind = 'S'
	     AND S.oid = D.objid
	     AND D.refobjid = T.oid
	     AND D.refobjid = C.attrelid
	     AND D.refobjsubid = C.attnum
	     AND T.relname = PGT.tablename
	 ORDER BY S.relname;`)
	if err != nil {
		return fmt.Errorf("Failed to generate sequence update: %w", err)
	}
	for _, r := range results {
		for _, value := range r {
			_, err = eng.Exec(value)
			if err != nil {
				return fmt.Errorf("Failed to update sequence %s: %w", value, err)
			}
		}
	}
	return nil
}
*/
