# API de unidades de BPGo

BPGo incluye una implementación de sala limpia (clean-room) de cada unidad
estándar de TP7 / BP7. El manifiesto en `compat/spec/units/<name>.json` lista
cada símbolo público. Este documento ofrece una breve descripción general de cada
unidad y un puntero a la implementación en Go.

| Unidad        | Paquete Go                          | Descripción |
|---------------|-------------------------------------|-------------|
| System       | `internal/rtl/system`              | Unidad implícita. Memoria, E/S, cadenas, matemáticas, ordinales, ayudantes de conjuntos, control |
| Crt          | `internal/rtl/crt`                 | Pantalla en modo texto, teclado, sonido, ventanas |
| Dos          | `internal/rtl/dos`                 | Servicios DOS: fecha/hora, entorno, archivos, despacho de interrupciones |
| Strings      | `internal/rtl/strings`             | Ayudantes de PChar (StrCat, StrComp, ...) |
| WinDos       | `internal/rtl/windos`              | Variantes con PChar de los servicios Dos |
| Printer      | `internal/rtl/printer`             | Variable de archivo Lst, en memoria o respaldada por archivo |
| Graph        | `internal/rtl/graph`               | Gráficos BGI: dibujo, viewports, paleta, páginas |
| Graph3       | `internal/rtl/graph3`              | Shim de graph de TP3 |
| Turbo3       | `internal/rtl/turbo3`              | Variables de archivo de TP3 (Kbd, Lst, Con, Aux, AuxIn) |
| Overlay      | `internal/rtl/overlay`             | Administrador de overlays |
| Objects      | `internal/tv/objects`              | TV: TObject, TStream, TCollection |
| Drivers      | `internal/tv/drivers`              | TV: eventos, ratón, pantalla |
| Views        | `internal/tv/views`                | TV: TView, TGroup, TFrame, TScrollBar, TInputLine, TButton, ... |
| Menus        | `internal/tv/menus`                | TV: TMenu, TMenuBar, TMenuBox, TStatusLine |
| Dialogs      | `internal/tv/dialogs`              | TV: TWindow, TDialog |
| App          | `internal/tv/app`                  | TV: TProgram, TApplication, TDesktop |
| HistList     | `internal/tv/histlist`             | TV: THistory, THistList |
| MsgBox       | `internal/tv/msgbox`               | TV: MessageBox, InputBox |
| StdDlg       | `internal/tv/stddlg`               | TV: diálogos de abrir/guardar archivo |
| Editors      | `internal/tv/editors`              | TV: TEditor, TFileEditor, TEditWindow |
| Validate     | `internal/tv/validate`             | TV: TValidator, TRangeValidator, TFilterValidator |
| ColorSel     | `internal/tv/colorsel`             | TV: TColorSelector, TColorDisplay |
| Outline      | `internal/tv/outline`              | TV: TOutline (vista de árbol) |
| Memory       | `internal/tv/memory`               | TV: envoltorio del administrador de memoria |

## Dónde encontrar cada símbolo

Cada símbolo público está declarado en el correspondiente
`compat/spec/units/<name>.json`. Cada entrada tiene:

- `name`: el identificador Pascal
- `kind`: `procedure`, `function`, `variable`, `type`, `constant`
- `signature`: la firma Pascal
- `status`: `implemented` (el paquete BPGo exporta el símbolo)
- `tests`: un grupo lógico de pruebas (p. ej. `system_mem`, `crt_screen`)

Los archivos de prueba de Go en `internal/rtl/<name>/<name>_test.go` cubren
cada símbolo implementado.
