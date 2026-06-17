program List;
type
  PNode = ^TNode;
  TNode = record
    Value: Integer;
    Next: PNode;
  end;
var
  Head: PNode;
begin
  Head := nil;
end.
