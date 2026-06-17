program CrtDemo;
{ Unit Crt: limpiar pantalla, posicionar el cursor y colorear (ANSI).
  Ejecutar: go run ./cmd/pasrun examples/pascal/crt_demo.pas }

uses Crt;

begin
  ClrScr;
  TextColor(14); { amarillo }
  GotoXY(5, 3);
  WriteLn('BPGo Turbo Pascal 7 - unit Crt');
  TextColor(10); { verde claro }
  GotoXY(5, 5);
  WriteLn('Texto en (5,5) con color');
  NormVideo;
end.
