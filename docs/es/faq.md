# Preguntas frecuentes y resolución de problemas

Respuestas breves a las preguntas que surgen con más frecuencia al embeber Pascal
con `pkg/vmpas` o al ejecutar `.pas` en el motor.

## Embebido

**¿Paso un fragmento o un programa completo?**
Cualquiera de los dos. `Run` acepta un fragmento simple (se envuelve en un
`begin … end.` implícito) o un `program … end.` / `unit … end.` completo. Un
fragmento es cómodo para instrucciones de una línea y reglas; un programa completo
cuando declaras tus propios tipos y rutinas.

**¿Cómo leo un valor de vuelta desde el script?**
Vincula una variable de Go por **puntero** con `Var`; vmpas la siembra en el
script y la copia de vuelta tras la ejecución:

```go
total := 10
eng.Var("total", &total)
eng.Run(`total := total * 2`)
// total == 20
```

Pasa un valor (no un puntero) para solo lectura. Los campos exportados de un
struct se mapean a campos de record por nombre (sin distinguir mayúsculas y
minúsculas); los slices se mapean a arreglos de base 0.

**¿Por qué falla mi script con "unknown identifier" en tiempo de compilación?**
De eso se trata: vmpas verifica los tipos antes de ejecutar. El nombre no está
declarado, o es un builtin restringido por capacidades (p. ej. `HttpGet`) y no
otorgaste la capacidad, por lo que nunca se registró. Otorga la capacidad o
corrige el nombre. Usa `Analyze` para ver qué necesita un script.

**¿Puede el mismo motor ejecutar muchos scripts?**
Sí. Cada `Run` construye una VM nueva y reinicia el estado transitorio, de modo
que las ejecuciones no se filtran entre sí. Para el mismo código ejecutado
repetidamente, `Compile` una sola vez en un `Script` y llama a `Script.Run` —
consulta [vmpas: compila una vez, ejecuta muchas](vmpas.md).

**¿Es vmpas seguro para código no confiable?**
Con la configuración adecuada, sí — consulta [seguridad](seguridad.md). Parte de
`Sandboxed()`, establece `MaxDuration`/`MaxOutput` y mantén acotados los callbacks
de Go registrados. Es un sandbox de lenguaje en proceso, no una jaula del sistema
operativo.

**¿Importar vmpas arrastra dependencias adicionales?**
No. `pkg/vmpas` no tiene dependencias, lo que se hace cumplir mediante
`TestVMPasHasNoExternalDeps`. El soporte de SQL usa el `database/sql` de la
biblioteca estándar de Go; *tú* aportas el driver e inyectas el handle con
`UseDB`, así que ningún driver entra nunca en el árbol de imports de vmpas.

## Lenguaje

**`Do`/`To` como identificadores no se parsean.**
`do` y `to` son palabras reservadas. Para nombrar un método o variable que
colisione, usa un identificador diferente (el lexer no distingue mayúsculas y
minúsculas, así que `Do`/`To` también están reservados).

**`x := inherited Foo + y` no se parsea.**
Limitación conocida: `inherited` funciona como **sentencia** (`inherited Init(a)`)
pero todavía no dentro de una **expresión**. Reescríbelo en dos pasos:

```pascal
tmp := inherited Foo;   { statement-style call into a temporary }
x := tmp + y;
```

**¿Se admite POO?**
Sí: tipos `object`, campos, herencia, **métodos virtuales con despacho
dinámico**, constructores/destructores e `inherited` en forma de sentencia.
Consulta la [matriz de compatibilidad](compatibility.md).

**¿Y `match`, `defer`, canales?**
Esas son extensiones modernas sobre TP7: [match](match.md),
[defer/panic/recover](defer.md), [spawn/canales](concurrency.md).

## Herramientas y errores

**¿Soporte de editores?**
Los servidores LSP (`pls`) y DAP (`pdap`) impulsan VSCode y Zed — diagnósticos,
hover, autocompletado, ir a definición y depuración. Consulta
[editores](editores.md).

**Un código de error en tiempo de ejecución como 200/202/203, ¿qué es?**
Son violaciones de límites: 200 = presupuesto de pasos/tiempo, 202 = profundidad
de llamadas, 203 = heap/salida. Eleva el límite correspondiente de `Capabilities`
si es legítimo, o trátalo como el sandbox haciendo su trabajo. El catálogo de
errores está en [errores](errors.md).

**`bpgo test-compat` solo muestra 2 tests, ¿es esa la cobertura real?**
No. Ese arnés es un stub heredado. La señal autorizada es la suite de tests del
motor: `go test ./...` (más de 600 tests). Consulta la
[matriz de compatibilidad](compatibility.md).

## Rendimiento

**¿Es vmpas más rápido que goja?**
En **memoria**, vmpas asigna mucho menos. En **tiempo** puro, goja es actualmente
~1.6–3.3× más rápido — es un intérprete de JS muy optimizado. vmpas se enfoca en
otra cosa: la verificación de tipos anticipada, el sandbox de capacidades, la
ejecución durable y la ausencia de dependencias. Las cifras y la
metodología están en [estado](estado.md).

**¿Cómo hago rápida la ejecución repetida?**
Usa `Compile` → `Script.Run` (compila una vez, ejecuta muchas) y vincula
variables en lugar de reconstruir la cadena del programa cada vez.
