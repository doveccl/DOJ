package judger

import "testing"

func TestCompareDefaultIgnoresTrailingSpaceAndBlankLines(t *testing.T) {
	verdict, message := Compare(ModeDefault, []byte("1 2  \n3\n\n"), []byte("1 2\n3   \n"))
	if verdict != VerdictAccepted {
		t.Fatalf("verdict = %s, message = %q", verdict, message)
	}
}

func TestCompareStrictReportsPresentationError(t *testing.T) {
	verdict, message := Compare(ModeStrict, []byte("1 2\n"), []byte("1 2  \n\n"))
	if verdict != VerdictPresentationError {
		t.Fatalf("verdict = %s, message = %q", verdict, message)
	}
}

func TestCompareWrongAnswer(t *testing.T) {
	verdict, message := Compare(ModeDefault, []byte("42\n"), []byte("43\n"))
	if verdict != VerdictWrongAnswer {
		t.Fatalf("verdict = %s, message = %q", verdict, message)
	}
}
