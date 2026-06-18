# `match`, sum types and `Option` (modern mode)

These are **modern language extensions**: they are not part of Turbo
Pascal 7. They are enabled only with the `{$MODE BPGO}` directive at the start of
the source. Without it, the compiler is strict TP7 and `match`, `Some`, etc.
remain ordinary identifiers (full compatibility).

```pascal
{$MODE BPGO}
program Ejemplo;
begin
  { ... }
end.
```

## 1. Sum types (ADTs)

A **sum type** (or algebraic type) is a type that can be one of several
*variants*, and each variant can carry data (a payload). It is declared as an
extended enumeration: each variant is a name with zero or more types between
parentheses.

```pascal
type
  TShape = (Circle(Integer), Rect(Integer, Integer), Empty);
```

Each variant is a **constructor**: calling it builds a value of that type.

```pascal
var s: TShape;
begin
  s := Rect(3, 4);   { variant with payload }
  s := Circle(10);
  s := Empty;        { variant without payload: used without parentheses }
end;
```

At runtime, a sum value is a tagged record (it carries the variant name and its
fields); you do not need to know that representation: it is inspected with
`match`.

## 2. `Option`: `Some` and `None`

`Option` comes built in and is the **honest** way to model "there may be no
value" (instead of returning `nil` and risking an invalid access):

```pascal
function Buscar(n: Integer): Integer;   { "returns" an Option }
begin
  if n > 0 then
    Buscar := Some(n * 10)   { there is a value }
  else
    Buscar := None;          { there is no value }
end;
```

- `Some(x)` wraps a present value.
- `None` represents the absence.

The consumer is **obligated to consider both cases** with `match`, so there are
no careless `nil` dereferences.

## 3. `match` as a statement

`match` inspects a value and executes the arm whose pattern matches:

```pascal
match Buscar(5) of
  Some(v) => WriteLn('encontrado: ', v);   { v is bound to the payload }
  None    => WriteLn('nada');
end;
```

General form:

```
match Expresion of
  Patron1 => Sentencia1;
  Patron2 => Sentencia2;
  else      Sentencia;     { optional }
end;
```

The expression is evaluated **only once**. Each pattern is tried in order; the
first match executes its statement and the rest are skipped.

## 4. `match` as an expression

The most powerful form: `match` **returns a value**, so it is used on the
right-hand side of an assignment or as an argument.

```pascal
function Area(s: TShape): Integer;
begin
  Area := match s of
    Circle(r)  => r * r * 3;
    Rect(w, h) => w * h;
    Empty      => 0;
  end;
end;
```

In the expression form each arm produces a value and the `else` is written
`else => valor`:

```pascal
nombre := match n of
  1 => 'uno';
  2 => 'dos';
  else => 'otro';
end;
```

## 5. Patterns

Within a `match` these patterns are allowed:

### Constructor with binding (destructuring)

Binds the payload fields to new variables, visible in that arm:

```pascal
match s of
  Rect(w, h) => WriteLn('área = ', w * h);   { w and h bound to the payload }
  Circle(r)  => WriteLn('radio = ', r);
end;
```

### Constant / enum

A name that is not a constructor is compared by equality (constants, enumeration
values):

```pascal
type TColor = (Red, Green, Blue);
...
match c of
  Red   => WriteLn('rojo');
  Green => WriteLn('verde');
  Blue  => WriteLn('azul');
end;
```

### Literal

Integers, characters and strings:

```pascal
match n of
  0 => WriteLn('cero');
  1 => WriteLn('uno');
end;
```

### Or-patterns (alternatives)

Several alternatives separated by commas share one arm:

```pascal
match n of
  1, 3, 5 => WriteLn('impar');
  2, 4    => WriteLn('par');
end;
```

### Wildcard `_` and `else`

`_` matches anything; `else` fulfills the same role as a final arm:

```pascal
match n of
  1 => WriteLn('uno');
  _ => WriteLn('cualquier otro');
end;
```

## 6. Guards (`when`)

An arm can add an extra condition with `when`: it only matches if the pattern
**and** the condition hold.

```pascal
match i of
  0            => WriteLn('cero');
  _ when i > 0 => WriteLn('positivo');
  _            => WriteLn('negativo');
end;
```

A wildcard with a guard is **not** terminal: if the condition fails, the
following arms are tried.

## 7. Non-exhaustive match

If no pattern matches and there is no `else`, the `match` **raises a runtime
error** (it does not continue silently). This avoids bugs from forgotten cases:

```pascal
match 7 of
  1 => WriteLn('uno');
end;
{ no arm matches and there is no else -> runtime error "match: no matching arm" }
```

To avoid it, cover all the cases or add an `else`.

## 8. Complete example

```pascal
{$MODE BPGO}
program Formas;
type
  TShape = (Circle(Integer), Rect(Integer, Integer), Empty);

function Describir(s: TShape): string;
begin
  Describir := match s of
    Circle(r) when r > 0 => 'círculo de radio ' + IntToStr(r);
    Circle(r)            => 'círculo degenerado';
    Rect(w, h)           => 'rect ' + IntToStr(w) + 'x' + IntToStr(h);
    Empty                => 'vacío';
  end;
end;

var s: TShape;
begin
  s := Rect(3, 4);
  WriteLn(Describir(s));   { rect 3x4 }
  s := Circle(5);
  WriteLn(Describir(s));   { círculo de radio 5 }
  s := Empty;
  WriteLn(Describir(s));   { vacío }
end.
```

## Limitations (honest)

- **Nested patterns** are not supported yet: `Some(Rect(w, h))` (a pattern
  inside another). Only a single level is destructured.
- **Static exhaustiveness**: it is not verified at compile time that all
  variants are covered (the engine is dynamically typed). The runtime error
  for non-exhaustiveness partially compensates.
- Or-patterns do not bind variables (they are used with literals/constants, not
  with constructors that carry a payload).

## See also

- [TP7 compatibility and modern extensions](compatibility.md)
- [Embedding Pascal in Go (`vmpas`)](vmpas.md)
