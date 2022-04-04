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

func ShBackslashQuote(in string) string {
	result := ""
	backslash_table := bashBstab

	if in == "" {
		return "''"
	}

	for _, c := range in {
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
