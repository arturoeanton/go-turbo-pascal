program Listas;
{ Lista enlazada simple con punteros y records. Ejecutar:
    go run ./cmd/pasrun examples/pascal/listas.pas }

type
  PNodo = ^TNodo;
  TNodo = record
    Valor: Integer;
    Sig: PNodo;
  end;

var
  cabeza, p: PNodo;
  i: Integer;

begin
  cabeza := nil;
  { Insertar 5..1 al frente, queda 1 2 3 4 5 }
  for i := 5 downto 1 do
  begin
    New(p);
    p^.Valor := i;
    p^.Sig := cabeza;
    cabeza := p;
  end;

  Write('Lista: ');
  p := cabeza;
  while p <> nil do
  begin
    Write(p^.Valor, ' ');
    p := p^.Sig;
  end;
  WriteLn;
end.
