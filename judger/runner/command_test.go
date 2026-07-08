package runner

import "testing"

func TestParseCommandUsesPlainFields(t *testing.T) {
	bin, args, err := parseCommand("  python3   main.py  ")
	if err != nil {
		t.Fatal(err)
	}
	if bin != "python3" || len(args) != 1 || args[0] != "main.py" {
		t.Fatalf("command = %q %#v", bin, args)
	}
	if _, _, err := parseCommand(`python3 "main.py"`); err == nil {
		t.Fatal("quoted command was accepted")
	}
}
