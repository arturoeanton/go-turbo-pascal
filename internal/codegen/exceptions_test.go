package codegen

import "testing"

func TestTryExceptCatches(t *testing.T) {
	check(t, `program E;
begin
  WriteLn('before');
  try
    WriteLn('in try');
    raise 'boom';
    WriteLn('unreachable');
  except
    WriteLn('caught');
  end;
  WriteLn('after');
end.`, "before\nin try\ncaught\nafter\n")
}

func TestTryExceptNoRaise(t *testing.T) {
	check(t, `program E;
begin
  try
    WriteLn('ok');
  except
    WriteLn('should not run');
  end;
  WriteLn('done');
end.`, "ok\ndone\n")
}

func TestTryFinallyAlwaysRuns(t *testing.T) {
	// finally runs on the normal path...
	check(t, `program E;
begin
  try
    WriteLn('body');
  finally
    WriteLn('cleanup');
  end;
  WriteLn('after');
end.`, "body\ncleanup\nafter\n")
}

func TestTryFinallyWithRaise(t *testing.T) {
	// ...and on the exception path (then the exception propagates to except).
	check(t, `program E;
begin
  try
    try
      raise 'x';
    finally
      WriteLn('cleanup');
    end;
  except
    WriteLn('caught');
  end;
end.`, "cleanup\ncaught\n")
}

func TestExceptionPropagatesAcrossCalls(t *testing.T) {
	check(t, `program E;
procedure Deep;
begin
  raise 'fail';
end;
procedure Mid;
begin
  Deep;
end;
begin
  try
    Mid;
  except
    WriteLn('handled');
  end;
end.`, "handled\n")
}

func TestUnhandledRaiseIsError(t *testing.T) {
	prog, err := Compile(`program E; begin raise 'oops'; end.`, "e.pas")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	_, _, runErr := Run(prog, nil, "")
	if runErr == nil {
		t.Fatal("expected a runtime error for an unhandled exception")
	}
}
