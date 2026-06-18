// Ejemplo: consumir una API HTTP y una base SQL desde Pascal embebido.
//
// Ejecutar con:
//
//	go run ./examples/integration
//
// Es autocontenido y offline: levanta un servidor HTTP local y usa una base
// SQL en memoria (implementando la interfaz vmpas.SQLDB), así no requiere red
// externa ni drivers de terceros. Muestra las capacidades Network y Database
// del sandbox y la API Http*/Db* del motor.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

func main() {
	// --- Servidor HTTP local (simula una API) ---
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	// --- Base SQL en memoria (el host provee la implementación) ---
	db := newMemDB([]string{"id", "name"}, [][]any{
		{int64(1), "alice"},
		{int64(2), "bob"},
	})

	eng := vmpas.NewWith(vmpas.Capabilities{Network: true, Database: true})
	eng.UseDB(db)

	url := srv.URL
	if err := eng.Var("url", &url); err != nil {
		panic(err)
	}

	script := `
begin
  { Consumir la API }
  WriteLn('GET ', url);
  WriteLn('respuesta: ', HttpGet(url));
  WriteLn('status: ', HttpLastStatus());

  { Recorrer una consulta SQL }
  WriteLn('usuarios:');
  if DbOpen('SELECT id, name FROM users') then
    while not DbEof() do
    begin
      WriteLn('  ', DbFieldInt(0), ' -> ', DbFieldStr(1));
      DbNext;
    end;
  DbClose;
end.`

	if err := eng.Run(script); err != nil {
		panic(err)
	}
	fmt.Print(eng.Output())
}

// --- Base SQL en memoria que satisface vmpas.SQLDB ---

type memDB struct {
	cols []string
	data [][]any
}

func newMemDB(cols []string, data [][]any) *memDB { return &memDB{cols: cols, data: data} }

func (d *memDB) Exec(query string, args ...any) (int64, error) { return int64(len(d.data)), nil }

func (d *memDB) Query(query string, args ...any) (vmpas.SQLRows, error) {
	return &memRows{cols: d.cols, data: d.data}, nil
}

type memRows struct {
	cols []string
	data [][]any
	pos  int
}

func (r *memRows) Columns() ([]string, error) { return r.cols, nil }
func (r *memRows) Next() bool {
	if r.pos < len(r.data) {
		r.pos++
		return true
	}
	return false
}
func (r *memRows) Scan(dest ...any) error {
	row := r.data[r.pos-1]
	for i := range dest {
		*(dest[i].(*any)) = row[i]
	}
	return nil
}
func (r *memRows) Close() error { return nil }
