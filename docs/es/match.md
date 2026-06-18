# `match`, tipos suma y `Option` (modo moderno)

Estas son **extensiones modernas** del lenguaje: no forman parte de Turbo
Pascal 7. Se activan solo con el directivo `{$MODE BPGO}` al inicio del fuente.
Sin él, el compilador es TP7 estricto y `match`, `Some`, etc. siguen siendo
identificadores normales (compatibilidad total).

```pascal
{$MODE BPGO}
program Ejemplo;
begin
  { ... }
end.
```

## 1. Tipos suma (ADTs)

Un **tipo suma** (o tipo algebraico) es un tipo que puede ser una de varias
*variantes*, y cada variante puede llevar datos (payload). Se declara como un
enumerado extendido: cada variante es un nombre con cero o más tipos entre
paréntesis.

```pascal
type
  TShape = (Circle(Integer), Rect(Integer, Integer), Empty);
```

Cada variante es un **constructor**: al llamarlo construye un valor de ese tipo.

```pascal
var s: TShape;
begin
  s := Rect(3, 4);   { variante con payload }
  s := Circle(10);
  s := Empty;        { variante sin payload: se usa sin paréntesis }
end;
```

En runtime, un valor suma es un registro etiquetado (lleva el nombre de la
variante y sus campos); no necesitás conocer esa representación: se inspecciona
con `match`.

## 2. `Option`: `Some` y `None`

`Option` viene integrado y es la forma **honesta** de modelar "puede no haber
valor" (en vez de devolver `nil` y arriesgar un acceso inválido):

```pascal
function Buscar(n: Integer): Integer;   { "devuelve" un Option }
begin
  if n > 0 then
    Buscar := Some(n * 10)   { hay valor }
  else
    Buscar := None;          { no hay valor }
end;
```

- `Some(x)` envuelve un valor presente.
- `None` representa la ausencia.

El consumidor está **obligado a considerar ambos casos** con `match`, así no hay
desreferencias de `nil` por descuido.

## 3. `match` como sentencia

`match` inspecciona un valor y ejecuta la rama (arm) cuyo patrón coincide:

```pascal
match Buscar(5) of
  Some(v) => WriteLn('encontrado: ', v);   { v se liga al payload }
  None    => WriteLn('nada');
end;
```

Forma general:

```
match Expresion of
  Patron1 => Sentencia1;
  Patron2 => Sentencia2;
  else      Sentencia;     { opcional }
end;
```

La expresión se evalúa **una sola vez**. Se prueba cada patrón en orden; la
primera coincidencia ejecuta su sentencia y el resto se omite.

## 4. `match` como expresión

La forma más potente: `match` **devuelve un valor**, así que se usa en el lado
derecho de una asignación o como argumento.

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

En la forma expresión cada rama produce un valor y el `else` se escribe
`else => valor`:

```pascal
nombre := match n of
  1 => 'uno';
  2 => 'dos';
  else => 'otro';
end;
```

## 5. Patrones

Dentro de un `match` se admiten estos patrones:

### Constructor con binding (destructuring)

Liga los campos del payload a variables nuevas, visibles en esa rama:

```pascal
match s of
  Rect(w, h) => WriteLn('área = ', w * h);   { w y h ligados al payload }
  Circle(r)  => WriteLn('radio = ', r);
end;
```

### Constante / enum

Un nombre que no es constructor se compara por igualdad (constantes, valores de
enumerados):

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

Enteros, caracteres y cadenas:

```pascal
match n of
  0 => WriteLn('cero');
  1 => WriteLn('uno');
end;
```

### Or-patterns (alternativas)

Varias alternativas separadas por coma comparten una rama:

```pascal
match n of
  1, 3, 5 => WriteLn('impar');
  2, 4    => WriteLn('par');
end;
```

### Comodín `_` y `else`

`_` coincide con cualquier cosa; `else` cumple el mismo rol como rama final:

```pascal
match n of
  1 => WriteLn('uno');
  _ => WriteLn('cualquier otro');
end;
```

## 6. Guards (`when`)

Una rama puede añadir una condición extra con `when`: solo coincide si el patrón
**y** la condición se cumplen.

```pascal
match i of
  0            => WriteLn('cero');
  _ when i > 0 => WriteLn('positivo');
  _            => WriteLn('negativo');
end;
```

Un comodín con guard **no** es terminal: si la condición falla, se prueban las
ramas siguientes.

## 7. Match no exhaustivo

Si ningún patrón coincide y no hay `else`, el `match` **lanza un error en tiempo
de ejecución** (no continúa en silencio). Esto evita bugs por casos olvidados:

```pascal
match 7 of
  1 => WriteLn('uno');
end;
{ ningún arm coincide y no hay else -> error de runtime "match: no matching arm" }
```

Para evitarlo, cubrí todos los casos o agregá un `else`.

## 8. Ejemplo completo

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

## Limitaciones (honestas)

- **Patrones anidados** no están soportados aún: `Some(Rect(w, h))` (un patrón
  dentro de otro). Se desestructura un solo nivel.
- **Exhaustividad estática**: no se verifica en tiempo de compilación que se
  cubran todas las variantes (el motor es de tipado dinámico). El error de
  runtime por no-exhaustivo compensa parcialmente.
- Las or-patterns no ligan variables (se usan con literales/constantes, no con
  constructores que tengan payload).

## Ver también

- [Compatibilidad TP7 y extensiones modernas](compatibility.md)
- [Embeber Pascal en Go (`vmpas`)](vmpas.md)
