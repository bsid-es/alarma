package sqlite_test

import (
	"testing"
	"testing/fstest"

	asqlite "bsid.es/alarma/sqlite"
	"bsid.es/alarma/sqlite/migration"
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

func TestMigrate(t *testing.T) {
	conn := mustOpenConn(t)
	defer mustCloseConn(t, conn)

	// Empty filesystems don't trigger migration.
	fsys := make(fstest.MapFS, 3)
	if err := asqlite.Migrate(conn, fsys); err != nil {
		t.Fatal(err)
	}
	assertVersion(t, conn, 0)

	// First script triggers migration.
	fsys["0000.sql"] = &fstest.MapFile{
		Data: []byte("create table t1 (a text);"),
	}
	if err := asqlite.Migrate(conn, fsys); err != nil {
		t.Fatal(err)
	}
	assertVersion(t, conn, 1)
	assertTableExists(t, conn, "t1")

	// Second script also triggers migration.
	fsys["0001.sql"] = &fstest.MapFile{
		Data: []byte("create table t2 (a text);"),
	}
	if err := asqlite.Migrate(conn, fsys); err != nil {
		t.Fatal(err)
	}
	assertVersion(t, conn, 2)
	assertTableExists(t, conn, "t2")

	// Non-SQL scripts don't trigger migration.
	fsys["0002.txt"] = &fstest.MapFile{
		Data: []byte("create table t3 (a text);"),
	}
	if err := asqlite.Migrate(conn, fsys); err != nil {
		t.Fatal(err)
	}
	assertVersion(t, conn, 2)
	assertTableDoesNotExist(t, conn, "t3")
}

func TestMigrateScripts(t *testing.T) {
	conn := mustOpenConn(t)
	defer mustCloseConn(t, conn)
	if err := asqlite.Migrate(conn, migration.Scripts); err != nil {
		t.Errorf("unexpected error\n%v", err)
	}
}

func mustOpenConn(tb testing.TB) *sqlite.Conn {
	tb.Helper()
	conn, err := sqlite.OpenConn(":memory:", 0)
	if err != nil {
		tb.Fatal(err)
	}
	return conn
}

func mustCloseConn(tb testing.TB, conn *sqlite.Conn) {
	tb.Helper()
	if err := conn.Close(); err != nil {
		tb.Fatal(err)
	}
}

func assertVersion(tb testing.TB, conn *sqlite.Conn, want int) {
	tb.Helper()
	var got int
	if err := sqlitex.Exec(conn, "pragma user_version", func(stmt *sqlite.Stmt) error {
		got = stmt.ColumnInt(0)
		return nil
	}); err != nil {
		tb.Fatal(err)
	} else if got != want {
		tb.Errorf("wrong version\ngot:  %d\nwant: %d", got, want)
	}
}

func assertTableExists(tb testing.TB, conn *sqlite.Conn, table string) {
	tb.Helper()
	if !tableExists(tb, conn, table) {
		tb.Fatalf("expected table %s", table)
	}
}

func assertTableDoesNotExist(tb testing.TB, conn *sqlite.Conn, table string) {
	tb.Helper()
	if tableExists(tb, conn, table) {
		tb.Fatalf("unexpected table %s", table)
	}
}

func tableExists(tb testing.TB, conn *sqlite.Conn, table string) bool {
	tb.Helper()
	var exists int
	err := sqlitex.Exec(
		conn,
		"select count(*) from sqlite_master where type='table' and name=?",
		func(stmt *sqlite.Stmt) error {
			exists = stmt.ColumnInt(0)
			return nil
		},
		table,
	)
	if err != nil {
		tb.Fatal(err)
	}
	return exists > 0
}
