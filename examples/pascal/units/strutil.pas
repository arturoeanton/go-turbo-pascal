unit StrUtil;
{ Unit de ejemplo: declara en la interface, implementa abajo, e inicializa
  un contador. Usada por demo.pas. }

interface

const
  Version = 1;

function Repetir(s: String; n: Integer): String;

var
  Llamadas: Integer;

implementation

function Repetir(s: String; n: Integer): String;
var
  i: Integer;
  r: String;
begin
  r := '';
  for i := 1 to n do
    r := r + s;
  Llamadas := Llamadas + 1;
  Repetir := r;
end;

initialization
  Llamadas := 0;
end.
