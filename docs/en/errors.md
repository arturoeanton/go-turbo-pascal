# BPGo Errors

The BPGo diagnostic catalog mirrors the Turbo Pascal 7 / Borland
Pascal 7 error numbers. The full list is defined in
`internal/diagnostics/diagnostics.go`; this document summarises
the categories.

## Compile errors (1-99)

| Code | Name                           | Notes |
|------|--------------------------------|-------|
| 1    | Unexpected symbol              | Unexpected token in a declaration or statement |
| 2    | Identifier expected            | Reserved word used where an identifier is required |
| 3    | Unknown identifier             | The name is not declared in the current scope |
| 4    | Duplicate identifier           | The name is already declared in this scope |
| 5    | Syntax error                   | Generic syntax error |
| 6    | Error in real constant         | Real literal is malformed |
| 7    | Error in integer constant      | Integer literal is malformed or out of range |
| 8    | String constant exceeds line   | Use `+` concatenation to span lines |
| 9    | Unterminated string            | Missing closing `'` |
| 10   | Expected closing quote         | Same as 9 in strict mode |
| 11   | Expected `=`                    | `=` required in const or type declarations |
| 12   | Expected `:=`                   | `:=` required for assignment |
| 13   | Type identifier expected       | A type name was expected |
| 14   | Expected `of`                   | `of` required after `array`/`case`/`set`/`file` |
| 15   | Expected `.`                    | A `.` was expected (e.g. end of unit) |
| 16   | Too many nested procedures     | Reduce nesting or use units |
| 17   | Bad type                       | Type is not valid in this context |
| 18   | Expected `END`                 | The current block must be closed with `END` |
| 26   | Type mismatch                  | Source and destination types are not compatible |
| 27   | Invalid subrange base type    | Subrange base must be ordinal |
| 28   | Lower bound > upper bound     | Swap the bounds |
| 29   | Ordinal expected              | An ordinal type is required |
| 30   | Integer constant expected     | Provide an integer constant |
| 31   | Constant expected              | Provide a constant expression |
| 32   | Integer or real expected      | Provide a numeric constant |
| 33   | Pointer type expected         | Use `^T` |
| 34   | Invalid function result type  | Use a scalar, pointer or string type |
| 35   | Label identifier expected     | Provide a numeric label |
| 36   | BEGIN expected                 | Start the block with BEGIN |
| 37   | Statement part too large      | Split into smaller procedures |
| 38   | Expected DO                    | Add DO |
| 39   | Expected THEN                  | Add THEN |
| 40   | Too many variables            | Reduce variable count or split unit |
| 41   | Undefined type                | Declare the type |
| 42   | File not allowed here         | Files have restrictions in this context |
| 43   | String length mismatch        | Source and destination strings differ in declared length |
| 44   | String constant expected      | Use a string literal |
| 45   | Integer or real variable expected | Provide a numeric variable |
| 46   | Ordinal variable expected     | Provide an ordinal variable |
| 47   | Character expression expected | Provide a Char-compatible expression |
| 48   | Structured variable expected  | Provide a record/array/file |
| 49   | Constant expression expected  | Use a constant |
| 50   | Integer expression expected   | Use an integer expression |
| 51   | Boolean expression expected   | Use a boolean expression |
| 52   | Operand types do not match    | Operator is not defined for these types |
| 53   | Field identifier expected     | Use a record field name |
| 54   | Object file too large         | Reduce code or split the unit |
| 55   | Undefined external            | Provide the external symbol or library |
| 56   | Invalid object file record    | OMF record is not supported |
| 57   | Code segment too large        | Code cannot exceed 64KB without overlays |
| 58   | Data segment too large        | Data cannot exceed 64KB |
| 84   | Unit name mismatch            | Unit identifier does not match filename |
| 85   | Unit version mismatch         | Recompile the unit |
| 86   | Duplicate unit name           | A unit appears twice in uses |
| 87   | Unit cycle detected           | Remove the circular uses |
| 88   | Unit not found                | Add the unit path or create the unit |

## Runtime errors (1-255)

Runtime errors are reported as integer codes by the System unit
runtime (`RunError`) and the IDE message loop.

| Code  | Name                              |
|-------|-----------------------------------|
| 1     | Invalid function number           |
| 2     | File not found                    |
| 3     | Path not found                    |
| 4     | Too many open files               |
| 5     | File access denied                |
| 6     | Invalid file handle               |
| 12    | Invalid file access code          |
| 15    | Invalid drive number              |
| 16    | Cannot remove current directory   |
| 17    | Not same device                   |
| 18    | No more files                     |
| 100   | Disk read error                   |
| 101   | Disk write error                  |
| 102   | File not assigned                 |
| 103   | File not open                     |
| 104   | File not open for input           |
| 105   | File not open for output          |
| 106   | Invalid numeric format            |
| 150   | Division by zero                  |
| 151   | Range check error                 |
| 152   | Stack overflow                    |
| 153   | Heap overflow                     |
| 154   | Invalid pointer operation         |
| 155   | Floating point overflow           |
| 156   | Floating point division by zero   |
| 157   | Invalid floating point operation |
| 158   | Floating point underflow          |
| 159   | Integer overflow                  |
| 160   | Invalid variant operation         |
| 161   | Invalid variant typecast          |
| 162   | Dispatch error                    |
| 200   | Division by zero (delay loop)     |
| 201   | Range check                       |
| 202   | Stack overflow                    |
| 203   | Heap overflow                     |
| 204   | Invalid pointer                   |
| 205   | Floating point overflow           |
| 206   | Floating point underflow          |
| 207   | Invalid 8087 opcode               |

## I/O errors

| Code  | Name                  |
|-------|-----------------------|
| 2     | File not found        |
| 3     | Path not found        |
| 5     | Access denied        |
| 32    | Sharing violation     |
| 100   | Disk read error       |
| 101   | Disk write error      |

## Graph errors

| Code  | Name                              |
|-------|-----------------------------------|
| 0     | No error                          |
| -1    | Graphics not initialized          |
| -2    | Graphics hardware not detected    |
| -3    | Driver file not found             |
| -4    | Invalid driver                    |
| -5    | Not enough memory to load driver  |
| -6    | Not enough memory to scan fill   |
| -7    | Not enough memory to flood fill  |
| -8    | Font file not found               |
| -9    | Invalid font                      |
| -10   | Invalid mode                      |
| -11   | Invalid fill                      |
| -12   | Palette index out of range        |
| -13   | Invalid image buffer              |
| -14   | Out of memory                     |
| -15   | Invalid line style                |
| -16   | Out of viewport                   |
| -17   | Invalid viewport                  |

## Overlay errors

| Code  | Name                      |
|-------|---------------------------|
| 0     | OK                        |
| -1    | Overlay error             |
| -2    | Overlay file not found    |
| -3    | Out of memory             |
| -4    | Overlay read error        |

## Debug errors

| Code  | Name                          |
|-------|-------------------------------|
| 1     | No source for address         |
| 2     | Invalid breakpoint            |
| 3     | Symbol not found              |
| 4     | Process not running           |
