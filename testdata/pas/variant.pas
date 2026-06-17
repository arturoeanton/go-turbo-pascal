program VariantRec;
type
  TValue = record
    case Kind: Integer of
      0: (I: Integer);
      1: (S: String);
  end;
var
  V: TValue;
begin
  V.Kind := 0;
  V.I := 42;
end.
