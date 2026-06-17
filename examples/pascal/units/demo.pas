program Demo;
{ Usa la unit StrUtil. Ejecutar:
    go run ./cmd/pasrun examples/pascal/units/demo.pas }

uses StrUtil;

begin
  WriteLn('StrUtil version ', Version);
  WriteLn(Repetir('ab', 3));
  WriteLn(Repetir('-', 10));
  WriteLn('Llamadas a Repetir: ', Llamadas);
end.
