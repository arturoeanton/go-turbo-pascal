# BPGo Compiler Directives

BPGo recognises the standard Turbo Pascal 7 / Borland Pascal 7
compiler directives. The full list is defined in
`compat/spec/directives.json`; this document summarises the
semantics implemented in the vm backend.

## Switch directives

| Directive | Default | Description                          |
|-----------|---------|--------------------------------------|
| `{$A+}`   | `{$A-}` | Align record fields on word boundary |
| `{$A-}`   |         | Pack fields tightly                  |
| `{$B+}`   | `{$B-}` | Boolean evaluation: full (always in vm) |
| `{$B-}`   |         | Short-circuit evaluation              |
| `{$D+}`   | `{$D-}` | Emit debug information                |
| `{$D-}`   |         | No debug information                 |
| `{$E+}`   | `{$E-}` | Use 8087 emulation                   |
| `{$E-}`   |         | No emulation                          |
| `{$F+}`   | `{$F-}` | Force far calls                      |
| `{$F-}`   |         | Allow near calls                     |
| `{$G+}`   | `{$G-}` | Use 80286 instructions               |
| `{$G-}`   |         | 8086 instructions only               |
| `{$I+}`   | `{$I-}` | Enable I/O checking                  |
| `{$I-}`   |         | No I/O checking                      |
| `{$N+}`   | `{$N-}` | Use numeric coprocessor              |
| `{$N-}`   |         | No numeric coprocessor               |
| `{$Q+}`   | `{$Q-}` | Enable integer overflow checking     |
| `{$Q-}`   |         | No overflow checking                 |
| `{$R+}`   | `{$R-}` | Enable range checking                |
| `{$R-}`   |         | No range checking                    |
| `{$S+}`   | `{$S-}` | Enable stack checking                |
| `{$S-}`   |         | No stack checking                    |
| `{$V+}`   | `{$V-}` | Strict var-string checking           |
| `{$V-}`   |         | Relaxed var-string checking          |
| `{$X+}`   | `{$X-}` | Enable extended syntax (function calls in expressions) |
| `{$X-}`   |         | Disable extended syntax             |

## Parametric directives

| Directive              | Description                       |
|------------------------|-----------------------------------|
| `{$I filename}`        | Include a file inline             |
| `{$L filename.obj}`    | Link an OMF .obj file             |
| `{$M stack,heapmin,heapmax}` | Set memory sizes in paragraphs |
| `{$O unitname}`        | Mark a unit as overlay             |

## Conditional compilation

| Directive            | Description                  |
|----------------------|------------------------------|
| `{$DEFINE name}`     | Define a symbol              |
| `{$UNDEF name}`      | Undefine a symbol            |
| `{$IFDEF name}`      | Compile next code if defined |
| `{$IFNDEF name}`     | Compile next code if not defined |
| `{$IFOPT switch}`    | Compile next code if switch is on |
| `{$ELSE}`            | Else branch of conditional  |
| `{$ENDIF}`           | End conditional block       |

## Notes on BPGo implementation

- The lexer silently discards `{$...}` and `(*$...*)` directives
  during tokenisation. The conformance harness verifies that the
  directive names are recognised in `compat/spec/directives.json`.
- The sem analyser and the vm backend do not yet honour these
  switches; the directives are recorded in the source map for IDE
  display but otherwise treated as a no-op.
- `{$I file.inc}` and `{$L file.obj}` are not yet wired to the
  include / OMF search paths.
