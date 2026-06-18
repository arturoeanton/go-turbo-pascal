package vmpas

import "database/sql"

// SQLDB is the minimal database surface the Db* host builtins use. The host
// supplies an implementation (bringing its own driver), so pkg/vmpas stays
// free of external dependencies. WrapSQLDB adapts a standard *sql.DB.
type SQLDB interface {
	// Exec runs a statement and returns the number of affected rows.
	Exec(query string, args ...any) (int64, error)
	// Query runs a query and returns a row cursor.
	Query(query string, args ...any) (SQLRows, error)
}

// SQLRows is a forward-only row cursor over a query result.
type SQLRows interface {
	Columns() ([]string, error)
	Next() bool                  // advance; false at end
	Scan(dest ...any) error       // scan the current row
	Close() error
}

// WrapSQLDB adapts a standard library *sql.DB to the SQLDB interface. It imports
// only database/sql (standard library), so the zero-external-dependency
// guarantee of pkg/vmpas is preserved; the concrete driver is registered by the
// host program.
func WrapSQLDB(db *sql.DB) SQLDB { return sqlDBAdapter{db} }

type sqlDBAdapter struct{ db *sql.DB }

func (a sqlDBAdapter) Exec(query string, args ...any) (int64, error) {
	res, err := a.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (a sqlDBAdapter) Query(query string, args ...any) (SQLRows, error) {
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil // *sql.Rows satisfies SQLRows
}
