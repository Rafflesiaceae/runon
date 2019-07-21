package shell_quote

import (
	"fmt"
	sh "shell_quote/sh"
	"testing"
)

func assertEqualToBashPrintfQ(t *testing.T, testStr string) {
	theirs, theirsErr, err := sh.CaptureExec("bash", "-lc", fmt.Sprintf("printf %%q '%s'", testStr))
	if err != 0 {
		t.Fatal(theirsErr)
	}

	ours := ShBackslashQuote(testStr)
	if ours != theirs {
		t.Fatalf("ours != theirs\nours: %s\ntheirs: %s", ours, theirs)
	}
}

func TestShBackslashQuote(t *testing.T) {
	assertEqualToBashPrintfQ(t, "qwe && asd | jkasd -- asdads | asd")
	assertEqualToBashPrintfQ(t, "echo asd && echo bsd")
	assertEqualToBashPrintfQ(t, "")
	assertEqualToBashPrintfQ(t, "BLUE=$((echo asd) | cat); echo ${BLUE}")
	assertEqualToBashPrintfQ(t, "echo \"echo \\\"echo bsd\\\"\"")
}
