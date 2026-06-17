# BPGo Units API

BPGo ships a clean-room implementation of every standard TP7 / BP7
unit. The manifest in `compat/spec/units/<name>.json` lists every
public symbol. This document provides a short overview of each
unit and a pointer to the Go implementation.

| Unit          | Go package                          | Description |
|---------------|-------------------------------------|-------------|
| System       | `internal/rtl/system`              | Implicit unit. Memory, I/O, strings, math, ordinals, set helpers, control |
| Crt          | `internal/rtl/crt`                 | Text-mode screen, keyboard, sound, windows |
| Dos          | `internal/rtl/dos`                 | DOS services: date/time, env, files, interrupt dispatch |
| Strings      | `internal/rtl/strings`             | PChar helpers (StrCat, StrComp, ...) |
| WinDos       | `internal/rtl/windos`              | PChar flavours of Dos services |
| Printer      | `internal/rtl/printer`             | Lst file variable, in-memory or file-backed |
| Graph        | `internal/rtl/graph`               | BGI graphics: drawing, viewports, palette, pages |
| Graph3       | `internal/rtl/graph3`              | TP3 graph shim |
| Turbo3       | `internal/rtl/turbo3`              | TP3 file variables (Kbd, Lst, Con, Aux, AuxIn) |
| Overlay      | `internal/rtl/overlay`             | Overlay manager |
| Objects      | `internal/tv/objects`              | TV: TObject, TStream, TCollection |
| Drivers      | `internal/tv/drivers`              | TV: events, mouse, screen |
| Views        | `internal/tv/views`                | TV: TView, TGroup, TFrame, TScrollBar, TInputLine, TButton, ... |
| Menus        | `internal/tv/menus`                | TV: TMenu, TMenuBar, TMenuBox, TStatusLine |
| Dialogs      | `internal/tv/dialogs`              | TV: TWindow, TDialog |
| App          | `internal/tv/app`                  | TV: TProgram, TApplication, TDesktop |
| HistList     | `internal/tv/histlist`             | TV: THistory, THistList |
| MsgBox       | `internal/tv/msgbox`               | TV: MessageBox, InputBox |
| StdDlg       | `internal/tv/stddlg`               | TV: file open/save dialogs |
| Editors      | `internal/tv/editors`              | TV: TEditor, TFileEditor, TEditWindow |
| Validate     | `internal/tv/validate`             | TV: TValidator, TRangeValidator, TFilterValidator |
| ColorSel     | `internal/tv/colorsel`             | TV: TColorSelector, TColorDisplay |
| Outline      | `internal/tv/outline`              | TV: TOutline (tree view) |
| Memory       | `internal/tv/memory`               | TV: memory manager wrapper |

## Where to find each symbol

Every public symbol is declared in the corresponding
`compat/spec/units/<name>.json`. Each entry has:

- `name`: the Pascal identifier
- `kind`: `procedure`, `function`, `variable`, `type`, `constant`
- `signature`: the Pascal signature
- `status`: `implemented` (the BPGo package exports the symbol)
- `tests`: a logical test bucket (e.g. `system_mem`, `crt_screen`)

The Go test files under `internal/rtl/<name>/<name>_test.go` cover
every implemented symbol.
