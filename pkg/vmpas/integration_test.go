package vmpas

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- HTTP -------------------------------------------------------------------

func TestHttpGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	}))
	defer srv.Close()

	e := NewWith(Capabilities{Network: true})
	url := srv.URL
	var body string
	if err := e.Var("url", &url); err != nil {
		t.Fatal(err)
	}
	if err := e.Var("body", &body); err != nil {
		t.Fatal(err)
	}
	if err := e.Run(`body := HttpGet(url)`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if body != "hello world" {
		t.Fatalf("HttpGet body = %q, want %q", body, "hello world")
	}
}

func TestHttpPostAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			fmt.Fprintf(w, "echo:%s", b)
			return
		}
		http.Error(w, "no", http.StatusNotFound)
	}))
	defer srv.Close()

	e := NewWith(Capabilities{Network: true})
	url := srv.URL
	var body string
	var status int
	e.Var("url", &url)
	e.Var("body", &body)
	e.Var("status", &status)
	if err := e.Run(`
begin
  body := HttpPost(url, 'text/plain', 'ping');
  status := HttpLastStatus();
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if body != "echo:ping" {
		t.Fatalf("HttpPost body = %q", body)
	}
	if status != 200 {
		t.Fatalf("HttpLastStatus = %d, want 200", status)
	}
}

func TestHttpBlockedByDefault(t *testing.T) {
	if err := New().Run(`program T; var s: string; begin s := HttpGet('http://x'); end.`); err == nil {
		t.Fatal("expected HTTP to be blocked under the default sandbox")
	}
}

// --- DB ---------------------------------------------------------------------

type fakeDB struct {
	cols     []string
	data     [][]any
	affected int64
	lastSQL  string
}

func (d *fakeDB) Exec(q string, args ...any) (int64, error) {
	d.lastSQL = q
	return d.affected, nil
}

func (d *fakeDB) Query(q string, args ...any) (SQLRows, error) {
	d.lastSQL = q
	return &fakeRows{cols: d.cols, data: d.data}, nil
}

type fakeRows struct {
	cols []string
	data [][]any
	pos  int
}

func (r *fakeRows) Columns() ([]string, error) { return r.cols, nil }
func (r *fakeRows) Next() bool {
	if r.pos < len(r.data) {
		r.pos++
		return true
	}
	return false
}
func (r *fakeRows) Scan(dest ...any) error {
	row := r.data[r.pos-1]
	for i := range dest {
		*(dest[i].(*any)) = row[i]
	}
	return nil
}
func (r *fakeRows) Close() error { return nil }

func TestDbQueryCursor(t *testing.T) {
	db := &fakeDB{
		cols: []string{"id", "name"},
		data: [][]any{
			{int64(1), "alice"},
			{int64(2), "bob"},
		},
	}
	e := NewWith(Capabilities{Database: true})
	e.UseDB(db)

	var names string
	var sumIDs int
	e.Var("names", &names)
	e.Var("sumIDs", &sumIDs)
	if err := e.Run(`
begin
  names := '';
  sumIDs := 0;
  if DbOpen('SELECT id, name FROM users') then
    while not DbEof() do
    begin
      sumIDs := sumIDs + DbFieldInt(0);
      names := names + DbFieldStr(1) + ';';
      DbNext;
    end;
  DbClose;
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if names != "alice;bob;" {
		t.Fatalf("names = %q, want %q", names, "alice;bob;")
	}
	if sumIDs != 3 {
		t.Fatalf("sumIDs = %d, want 3", sumIDs)
	}
}

func TestDbExec(t *testing.T) {
	db := &fakeDB{affected: 5}
	e := NewWith(Capabilities{Database: true})
	e.UseDB(db)

	var n int
	e.Var("n", &n)
	if err := e.Run(`n := DbExec('DELETE FROM logs')`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if n != 5 {
		t.Fatalf("DbExec affected = %d, want 5", n)
	}
	if db.lastSQL != "DELETE FROM logs" {
		t.Fatalf("lastSQL = %q", db.lastSQL)
	}
}

func TestDbBlockedByDefault(t *testing.T) {
	if err := New().Run(`program T; var n: Integer; begin n := DbExec('x'); end.`); err == nil {
		t.Fatal("expected DB to be blocked under the default sandbox")
	}
}
