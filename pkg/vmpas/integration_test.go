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

func TestHttpAllVerbsAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo the method and the Authorization header so the test can assert both.
		fmt.Fprintf(w, "%s auth=%s", r.Method, r.Header.Get("Authorization"))
	}))
	defer srv.Close()

	e := NewWith(Capabilities{Network: true})
	url := srv.URL
	var put, del, patch, custom string
	e.Var("url", &url)
	e.Var("put", &put)
	e.Var("del", &del)
	e.Var("patch", &patch)
	e.Var("custom", &custom)
	if err := e.Run(`
begin
  HttpSetHeader('Authorization', 'Bearer tok');
  put    := HttpPut(url, 'application/json', '{}');
  del    := HttpDelete(url);
  patch  := HttpPatch(url, 'application/json', '{}');
  custom := HttpRequest('OPTIONS', url, '', '');
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if put != "PUT auth=Bearer tok" {
		t.Fatalf("PUT = %q", put)
	}
	if del != "DELETE auth=Bearer tok" {
		t.Fatalf("DELETE = %q", del)
	}
	if patch != "PATCH auth=Bearer tok" {
		t.Fatalf("PATCH = %q", patch)
	}
	if custom != "OPTIONS auth=Bearer tok" {
		t.Fatalf("HttpRequest OPTIONS = %q", custom)
	}
}

func TestJsonAccessors(t *testing.T) {
	e := New() // JSON needs no capability
	var name string
	var count int
	var ok bool
	var n int
	e.Var("name", &name)
	e.Var("count", &count)
	e.Var("ok", &ok)
	e.Var("n", &n)
	if err := e.Run(`
begin
  name  := JsonStr('{"user":{"name":"alice"},"items":[10,20,30],"active":true,"count":7}', 'user.name');
  count := JsonInt('{"count":7}', 'count');
  ok    := JsonBool('{"active":true}', 'active');
  n     := JsonLen('{"items":[10,20,30]}', 'items');
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if name != "alice" {
		t.Fatalf("JsonStr = %q", name)
	}
	if count != 7 {
		t.Fatalf("JsonInt = %d", count)
	}
	if !ok {
		t.Fatalf("JsonBool = %v", ok)
	}
	if n != 3 {
		t.Fatalf("JsonLen = %d", n)
	}
}

func TestJsonArrayIndexAndValid(t *testing.T) {
	e := New()
	var id int
	var valid, bad bool
	e.Var("id", &id)
	e.Var("valid", &valid)
	e.Var("bad", &bad)
	if err := e.Run(`
begin
  id    := JsonInt('{"items":[{"id":1},{"id":42}]}', 'items.1.id');
  valid := JsonValid('{"a":1}');
  bad   := JsonValid('{not json');
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if id != 42 {
		t.Fatalf("nested array id = %d, want 42", id)
	}
	if !valid || bad {
		t.Fatalf("JsonValid: valid=%v bad=%v", valid, bad)
	}
}

func TestJsonBuild(t *testing.T) {
	e := New()
	// Build a nested document, then read fields back (round-trip).
	var name string
	var age int
	var active bool
	var nested string
	e.Var("name", &name)
	e.Var("age", &age)
	e.Var("active", &active)
	e.Var("nested", &nested)
	if err := e.Run(`program T;
var doc, name, nested: string; age: Integer; active: Boolean;
begin
  doc := '{}';
  doc := JsonSetStr(doc, 'user.name', 'alice');
  doc := JsonSetInt(doc, 'user.age', 30);
  doc := JsonSetBool(doc, 'user.active', true);
  doc := JsonSetInt(doc, 'tags.0', 7);
  doc := JsonSetInt(doc, 'tags.1', 9);
  name   := JsonStr(doc, 'user.name');
  age    := JsonInt(doc, 'user.age');
  active := JsonBool(doc, 'user.active');
  nested := JsonInt(doc, 'tags.1');  { array index round-trip via string }
end.`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if name != "alice" || age != 30 || !active {
		t.Fatalf("round-trip: name=%q age=%d active=%v", name, age, active)
	}
	if nested != "9" {
		t.Fatalf("tags.1 = %q, want 9", nested)
	}
}

func TestJsonEscape(t *testing.T) {
	e := New()
	var out string
	e.Var("out", &out)
	if err := e.Run(`out := JsonEscape('he said "hi"')`); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != `"he said \"hi\""` {
		t.Fatalf("JsonEscape = %q", out)
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
