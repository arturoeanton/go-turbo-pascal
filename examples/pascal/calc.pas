program Calc;
{ Lee dos enteros y muestra operaciones con formato de campo.
  Ejecutar:  echo "12 5" | go run ./cmd/pasrun examples/pascal/calc.pas }

var
  a, b: Integer;
  prom: Real;

begin
  Write('Ingrese dos numeros: ');
  ReadLn(a, b);
  WriteLn('Suma:       ', (a + b):6);
  WriteLn('Producto:   ', (a * b):6);
  prom := (a + b) / 2;
  WriteLn('Promedio:   ', prom:6:2);
end.
