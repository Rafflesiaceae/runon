package shell_quote

var ( // bash-5.0
	/* Default set of characters that should be backslash-quoted in strings */
	bashBstab = [256]int{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 1, 1, 0, 0, 0, 0, 0, /* TAB, NL */
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,

		1, 1, 1, 0, 1, 0, 1, 1, /* SPACE, !, DQUOTE, DOL, AMP, SQUOTE */
		1, 1, 1, 0, 1, 0, 0, 0, /* LPAR, RPAR, STAR, COMMA */
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 1, 1, 0, 1, 1, /* SEMI, LESSTHAN, GREATERTHAN, QUEST */

		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 1, 1, 1, 1, 0, /* LBRACK, BS, RBRACK, CARAT */

		1, 0, 0, 0, 0, 0, 0, 0, /* BACKQ */
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 1, 1, 1, 0, 0, /* LBRACE, BAR, RBRACE */

		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,

		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,

		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,

		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
)

// case 'q':		/* print with shell quoting */
//   {
// char *p, *xp;
// int r;

// r = 0;
// p = getstr ();
// if (p && *p == 0)	/* XXX - getstr never returns null */
//   xp = savestring ("''");
// else if (ansic_shouldquote (p))
//   xp = ansic_quote (p, 0, (int *)0);
// else
//   xp = sh_backslash_quote (p, 0, 3);
// if (xp)
//   {
//     /* Use printstr to get fieldwidth and precision right. */
//     r = printstr (start, xp, strlen (xp), fieldwidth, precision);
//     if (r < 0)
//       {
// 	sh_wrerror ();
// 	clearerr (stdout);
//       }
//     free (xp);
//   }

// if (r < 0)
//   PRETURN (EXECUTION_FAILURE);
// break;
//   }

// AnsicShouldQuote returns true if passed string needs to be quoted
func AnsicShouldQuote(in string) bool {
	// @TODO
	return true
}

// ShBackslashQuote
func ShBackslashQuote(in string) string {
	result := ""
	backslash_table := bashBstab

	if in == "" {
		return "''"
	}

	for _, c := range in {
		// println(c)
		if c >= 0 && c <= 127 && backslash_table[c] == 1 {
			result += "\\"
			result += string(c)
			continue
		}

		result += string(c)
	}
	// @TODO
	return result
}

func Quote(in string) string {
	if in == "" {
		return "''"
	}

	if AnsicShouldQuote(in) {

	} else {

	}

	return ""
}
