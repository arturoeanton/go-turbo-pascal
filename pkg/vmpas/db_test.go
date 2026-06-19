package vmpas

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"
)

// A minimal in-memory database/sql driver (standard library only) to exercise
// WrapSQLDB and the *sql.DB adapter without pulling in an external driver.
type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return fakeConn{}, nil }

type fakeConn struct{}

func (fakeConn) Prepare(q string) (driver.Stmt, error) { return fakeStmt{}, nil }
func (fakeConn) Close() error                          { return nil }
func (fakeConn) Begin() (driver.Tx, error)             { return nil, driver.ErrSkip }

type fakeStmt struct{}

func (fakeStmt) Close() error  { return nil }
func (fakeStmt) NumInput() int { return -1 }
func (fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(2), nil
}
func (fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &fakeDriverRows{cols: []string{"x"}, data: [][]driver.Value{{int64(7)}}}, nil
}

type fakeDriverRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *fakeDriverRows) Columns() []string { return r.cols }
func (r *fakeDriverRows) Close() error      { return nil }
func (r *fakeDriverRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

func init() { sql.Register("vmpasfake", fakeDriver{}) }

// TestWrapSQLDB covers WrapSQLDB and the sqlDBAdapter (Exec/Query) over a real
// *sql.DB; the Db* builtins themselves are covered in integration_test.go.
func TestWrapSQLDB(t *testing.T) {
	db, err := sql.Open("vmpasfake", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wrapped := WrapSQLDB(db)

	n, err := wrapped.Exec("INSERT INTO t VALUES (?)", 1)
	if err != nil || n != 2 {
		t.Fatalf("Exec: n=%d err=%v", n, err)
	}

	rows, err := wrapped.Query("SELECT x FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil || len(cols) != 1 || cols[0] != "x" {
		t.Fatalf("Columns: %v err=%v", cols, err)
	}
	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var v any
	if err := rows.Scan(&v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got, ok := v.(int64); !ok || got != 7 {
		t.Fatalf("scanned value = %v (%T), want int64 7", v, v)
	}
}
