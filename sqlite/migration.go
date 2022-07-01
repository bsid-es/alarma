package sqlite

import (
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

func Migrate(conn *sqlite.Conn, fsys fs.FS) (err error) {
	release := sqlitex.Save(conn)
	defer release(&err)

	var oldVer int
	if err = sqlitex.ExecTransient(conn, "pragma user_version", func(stmt *sqlite.Stmt) error {
		oldVer = stmt.ColumnInt(0)
		return nil
	}); err != nil {
		return fmt.Errorf("get version: %v", err)
	}

	scripts, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list scripts: %v", err)
	}
	currVer := len(scripts)

	if oldVer >= currVer {
		// There are no scripts to run.
		return nil
	}

	sort.Strings(scripts)
	for _, script := range scripts[oldVer:] {
		buf, err := fs.ReadFile(fsys, script)
		if err != nil {
			return fmt.Errorf("read %s: %v", script, err)
		}
		queries := string(buf)
		for i := 0; queries != ""; i++ {
			stmt, trailingBytes, err := conn.PrepareTransient(queries)
			if err != nil {
				return fmt.Errorf("prepare %s, stmt %d: %v", script, i, err)
			}
			usedBytes := len(queries) - trailingBytes
			queries = queries[usedBytes:]
			_, err = stmt.Step()
			stmt.Finalize()
			if err != nil {
				return fmt.Errorf("execute %s, stmt %d: %v", script, i, err)
			}
			queries = strings.TrimSpace(queries)
		}

	}

	newVer := strconv.Itoa(currVer)
	if err := sqlitex.Exec(conn, "pragma user_version="+newVer, nil); err != nil {
		return fmt.Errorf("set version: %v", err)
	}

	return nil
}
