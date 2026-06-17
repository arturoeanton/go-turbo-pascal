program ForwardTest;
procedure P; forward;
procedure Q; begin P; end;
procedure P; begin end;
begin
  Q;
end.
