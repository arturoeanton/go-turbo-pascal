# Compiler directives

go-turbo-pascal recognises the standard Turbo Pascal 7 / Borland Pascal 7
compiler directives. The full list is defined in
`compat/spec/directives.json`; this document summarises the
semantics on the bytecode VM backend.

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

## Implementation notes

These directives are recognised for source compatibility, but most have no
runtime effect on the bytecode backend — the VM is not a DOS/8086 target, so
switches about segments, coprocessors, far calls and memory paragraphs are
parsed and ignored:

- The lexer accepts `{$...}` and `(*$...*)` directives during tokenisation and
  records them in the source map (for IDE display). Switch directives such as
  `{$R+}`/`{$I+}` are treated as no-ops on the VM backend.
- `{$I file.inc}` (include) and `{$L file.obj}` (OMF linking) are **not** wired
  to the VM backend; OMF/8086 is a legacy/experimental path (see the
  [compatibility matrix](compatibility.md)).
