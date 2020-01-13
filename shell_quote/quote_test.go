package shell_quote

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
)

func assertEqualToBashPrintfQ(t *testing.T, testStr string) {
	var theirs string

	{
		cmd := exec.Command("bash", "-lc", fmt.Sprintf("printf %%q '%s'", testStr))

		var stdoutBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf

		err := cmd.Run()
		if err != nil {
			t.Error(err)
		}

		theirs = string(stdoutBuf.Bytes())
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
