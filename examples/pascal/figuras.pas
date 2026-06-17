program Figuras;
{ OOP estilo TP7: objetos, herencia, métodos virtuales e inherited.
  Ejecutar: go run ./cmd/pasrun examples/pascal/figuras.pas }

type
  TFigura = object
    Nombre: Integer;
    constructor Init(N: Integer);
    function Area: Integer;
  end;

  TRectangulo = object(TFigura)
    Ancho, Alto: Integer;
    constructor Init(W, H: Integer);
    function Area: Integer;
  end;

constructor TFigura.Init(N: Integer);
begin
  Nombre := N;
end;

function TFigura.Area: Integer;
begin
  Area := 0;
end;

constructor TRectangulo.Init(W, H: Integer);
begin
  inherited Init(1);
  Ancho := W;
  Alto := H;
end;

function TRectangulo.Area: Integer;
begin
  Area := Ancho * Alto;
end;

var
  r: TRectangulo;
begin
  r.Init(4, 5);
  WriteLn('Area del rectangulo: ', r.Area);
end.
