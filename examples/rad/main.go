// Command rad is a small visual workflow builder ("RAD") on top of pkg/vmpas.
//
//	go run .          # in this directory, then open http://localhost:8080
//
// Drag boxes onto a canvas to compose a flow. Each box shows the Pascal it
// generates; a "Custom Pascal" box lets you write your own (edited with a
// Pascal-aware editor). "Run" compiles the flow and executes it on the embedded
// engine; an "Approval" box calls Suspend, so the run pauses, its state is
// persisted (durable execution) and the UI offers Approve / Reject, which
// resumes it — in a fresh engine, as a real service would after a human
// decision. Each executed box reports back live via a bound Trace() callback.
//
// Everything (saved flows, run history, paused states) is stored in SQLite via
// the pure-Go modernc.org/sqlite driver. This program lives in its own Go module
// so that driver never enters the engine's dependency-free import tree.
package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "modernc.org/sqlite"

	"github.com/arturoeanton/go-turbo-pascal/pkg/vmpas"
)

//go:embed index.html
var assets embed.FS

var db *sql.DB

func initDB(path string) {
	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	const schema = `
CREATE TABLE IF NOT EXISTS flows  (name TEXT PRIMARY KEY, graph TEXT, updated TEXT);
CREATE TABLE IF NOT EXISTS runs   (id INTEGER PRIMARY KEY AUTOINCREMENT, flow TEXT, status TEXT, outcome TEXT, output TEXT, trace TEXT, created TEXT);
CREATE TABLE IF NOT EXISTS paused (id TEXT PRIMARY KEY, program TEXT, amount REAL, tag TEXT, output TEXT, state BLOB, trace TEXT, created TEXT);`
	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}
	// Demo data for the example "DB query" component (vmpas Db* via UseDB).
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS users(id INTEGER PRIMARY KEY, name TEXT)`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO users(id,name) VALUES (1,'alice'),(2,'bob'),(3,'carol')`)
}

// caps: deterministic (reproducible persisted state) and bounded (the program is
// drawn in the browser, i.e. untrusted, so it runs under a sandbox).
func caps() vmpas.Capabilities {
	c := vmpas.Sandboxed()
	c.Deterministic, c.Seed = true, 1
	c.MaxDuration = 2 * time.Second
	// This is a local demo: allow the example DB and HTTP boxes to work. DB is
	// wired to the local SQLite via UseDB; HTTP can reach the bundled /demo/api.
	c.Database, c.Network = true, true
	return c
}

type resp struct {
	Status  string   `json:"status"` // done | paused | error
	Output  string   `json:"output"`
	Outcome string   `json:"outcome,omitempty"`
	Tag     string   `json:"tag,omitempty"`
	ID      string   `json:"id,omitempty"`
	Trace   []string `json:"trace"`
	Error   string   `json:"error,omitempty"`
}

// execute runs (st==nil) or resumes a flow, recording the executed boxes via a
// bound Trace() callback the composed program calls at each box.
func execute(program string, amount float64, approved bool, st *vmpas.State) (*vmpas.State, resp) {
	eng := vmpas.NewWith(caps())
	eng.UseDB(vmpas.WrapSQLDB(db)) // example "DB query" boxes run against the local SQLite
	a, ap := amount, approved
	_ = eng.Var("amount", &a)
	_ = eng.Var("approved", &ap)
	var trace []string
	_ = eng.Process("Trace", func(id string) { trace = append(trace, id) })

	var state *vmpas.State
	var err error
	if st == nil {
		state, err = eng.RunDurable(program)
	} else {
		state, err = eng.ResumeDurable(program, st)
	}
	if err != nil {
		return nil, resp{Status: "error", Output: eng.Output(), Trace: trace, Error: err.Error()}
	}
	if state != nil {
		return state, resp{Status: "paused", Output: state.Output, Tag: state.Tag, Trace: trace}
	}
	var outcome string
	_ = eng.Get("outcome", &outcome)
	return nil, resp{Status: "done", Output: eng.Output(), Outcome: outcome, Trace: trace}
}

func logRun(flow string, r resp) {
	tr, _ := json.Marshal(r.Trace)
	_, _ = db.Exec(`INSERT INTO runs(flow,status,outcome,output,trace,created) VALUES(?,?,?,?,?,?)`,
		flow, r.Status, r.Outcome, r.Output, string(tr), time.Now().Format(time.RFC3339))
}

func main() {
	initDB("rad.db")
	http.Handle("/", http.FileServer(http.FS(assets)))

	// Tiny bundled API for the example "HTTP fetch" component (offline-friendly).
	http.HandleFunc("/demo/api", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"vmpas","version":"1.4","items":[1,2,3]}`))
	})

	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Program string  `json:"program"`
			Amount  float64 `json:"amount"`
			Flow    string  `json:"flow"`
		}
		if !decode(w, r, &req) {
			return
		}
		state, out := execute(req.Program, req.Amount, false, nil)
		if state != nil {
			id := "wf-" + strconv.FormatInt(time.Now().UnixNano(), 36)
			tr, _ := json.Marshal(out.Trace)
			_, _ = db.Exec(`INSERT INTO paused(id,program,amount,tag,output,state,trace,created) VALUES(?,?,?,?,?,?,?,?)`,
				id, req.Program, req.Amount, state.Tag, state.Output, state.Data, string(tr), time.Now().Format(time.RFC3339))
			out.ID = id
		}
		logRun(req.Flow, out)
		writeJSON(w, out)
	})

	http.HandleFunc("/api/resume", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID       string `json:"id"`
			Approved bool   `json:"approved"`
		}
		if !decode(w, r, &req) {
			return
		}
		var program, tag, output, traceJSON string
		var amount float64
		var data []byte
		row := db.QueryRow(`SELECT program,amount,tag,output,state,trace FROM paused WHERE id=?`, req.ID)
		if err := row.Scan(&program, &amount, &tag, &output, &data, &traceJSON); err != nil {
			writeJSON(w, resp{Status: "error", Error: "unknown or expired workflow id"})
			return
		}
		_, _ = db.Exec(`DELETE FROM paused WHERE id=?`, req.ID)

		state, out := execute(program, amount, req.Approved, &vmpas.State{Tag: tag, Data: data, Output: output})
		// Carry forward the boxes traced before the pause so the UI shows the whole path.
		var before []string
		_ = json.Unmarshal([]byte(traceJSON), &before)
		out.Trace = append(before, out.Trace...)
		if state != nil {
			out.ID = req.ID // (a multi-approval flow could pause again)
		}
		logRun("", out)
		writeJSON(w, out)
	})

	http.HandleFunc("/api/flows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req struct{ Name, Graph string }
			if !decode(w, r, &req) {
				return
			}
			_, err := db.Exec(`INSERT INTO flows(name,graph,updated) VALUES(?,?,?)
				ON CONFLICT(name) DO UPDATE SET graph=excluded.graph, updated=excluded.updated`,
				req.Name, req.Graph, time.Now().Format(time.RFC3339))
			writeJSON(w, resp{Status: ok(err), Error: errStr(err)})
			return
		}
		rows, _ := db.Query(`SELECT name,updated FROM flows ORDER BY updated DESC`)
		defer rows.Close()
		list := []map[string]string{}
		for rows.Next() {
			var n, u string
			_ = rows.Scan(&n, &u)
			list = append(list, map[string]string{"name": n, "updated": u})
		}
		writeListJSON(w, list)
	})

	http.HandleFunc("/api/flow", func(w http.ResponseWriter, r *http.Request) {
		var graph string
		err := db.QueryRow(`SELECT graph FROM flows WHERE name=?`, r.URL.Query().Get("name")).Scan(&graph)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(graph))
	})

	http.HandleFunc("/api/runs", func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.Query(`SELECT flow,status,outcome,created FROM runs ORDER BY id DESC LIMIT 20`)
		defer rows.Close()
		list := []map[string]string{}
		for rows.Next() {
			var f, s, o, c string
			_ = rows.Scan(&f, &s, &o, &c)
			list = append(list, map[string]string{"flow": f, "status": s, "outcome": o, "created": c})
		}
		writeListJSON(w, list)
	})

	log.Println("RAD workflow demo on http://localhost:8080  (SQLite: rad.db)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, resp{Status: "error", Error: err.Error()})
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, v resp) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func writeListJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func ok(err error) string {
	if err != nil {
		return "error"
	}
	return "done"
}
func errStr(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
