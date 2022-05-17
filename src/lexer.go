package src

import (
	"fmt"
	"strings"
)

func lex(source string) ([]*token, error) {
	tokens := []*token{}
	cur := cursor{}
lex:
	for cur.Pointer < uint(len(source)) {
		lexers := []Lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
		for _, l := range lexers {
			if token, newCursor, ok := l(source, cur); ok {
				cur = newCursor
				if token != nil {
					tokens = append(tokens, token)
				}

				continue lex
			}
		}

		// Informing any syntax errors
		hint := ""
		if len(tokens) > 0 {
			hint = " after " + tokens[len(tokens)-1].Value
		}
		return nil, fmt.Errorf("Unable to lex tokens%s, at %d:%d", hint, cur.Loc.line, cur.Loc.col)
	}
	return tokens, nil
}

func lexNumeric(source string, ic cursor) (*token, cursor, bool) {
	cur := ic

	foundPeriod := false
	foundExpMarker := false

	for ; cur.Pointer < uint(len(source)); cur.Pointer++ {
		c := source[cur.Pointer]
		cur.Loc.col++

		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExpmarker := c == 'e'

		// starting with a digit or period
		if cur.Pointer == ic.Pointer {
			if !isDigit && !isPeriod {
				return nil, ic, false
			}

			foundPeriod = isPeriod
			continue
		}

		if isPeriod {
			if foundPeriod {
				return nil, ic, false
			}

			foundPeriod = true
			continue
		}

		if isExpmarker {
			if foundExpMarker {
				return nil, ic, false
			}

			// No periods allowed after expMarker
			foundPeriod = true
			foundExpMarker = true

			if cur.Pointer == uint(len(source)-1) {
				return nil, ic, false
			}

			cNext := source[cur.Pointer+1]
			if cNext == '-' || cNext == '+' {
				cur.Pointer++
				cur.Loc.col++
			}

			continue
		}

		if !isDigit {
			break
		}
	}

	if cur.Pointer == ic.Pointer {
		return nil, ic, false
	}

	return &token{
		Value: source[ic.Pointer:cur.Pointer],
		Loc:   ic.Loc,
		Kind:  NumericKind,
	}, cur, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*token, cursor, bool) {
	cur := ic

	if len(source[cur.Pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.Pointer] != delimiter {
		return nil, ic, false
	}

	cur.Loc.col++
	cur.Pointer++

	var value []byte
	for ; cur.Pointer < uint(len(source)); cur.Pointer++ {
		c := source[cur.Pointer]

		if c == delimiter {
			// SQL escapes are via double characters, not backslash.
			if cur.Pointer+1 >= uint(len(source)) || source[cur.Pointer+1] != delimiter {
				cur.Pointer++ // TODO: fix this duplicated code
				cur.Loc.col++
				return &token{
					Value: string(value),
					Loc:   ic.Loc,
					Kind:  StringKind,
				}, cur, true
			} else {
				value = append(value, delimiter)
				cur.Pointer++
				cur.Loc.col++
			}
		}

		value = append(value, c)
		cur.Loc.col++
	}

	return nil, ic, false
}

func lexString(source string, ic cursor) (*token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	c := source[ic.Pointer]
	cur := ic

	cur.Pointer++
	cur.Loc.col++

	// Syntax that should be thrown away
	switch c {
	case '\n':
		cur.Loc.line++
		cur.Loc.col = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true
	}

	symbols := []symbol{
		Comma,
		LeftParen,
		RightParen,
		SemiColon,
		Asterisk,
	}

	var options []string
	for _, s := range symbols {
		options = append(options, string(s))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.Pointer = ic.Pointer + uint(len(match))
	cur.Loc.col = ic.Loc.col + uint(len(match))

	return &token{
		Value: match,
		Loc:   ic.Loc,
		Kind:  SymbolKind,
	}, cur, true
}

func lexKeyword(source string, ic cursor) (*token, cursor, bool) {
	cur := ic
	keywords := []keyword{
		Select,
		From,
		As,
		Table,
		Create,
		Insert,
		Into,
		Values,
		Int,
		Text,
	}

	var options []string
	for _, k := range keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}
	cur.Pointer = ic.Pointer + uint(len(match))
	cur.Loc.col = ic.Loc.col + uint(len(match))

	return &token{
		Value: match,
		Kind:  KeywordKind,
		Loc:   ic.Loc,
	}, cur, true
}

// longestMatch iter through a source string starting at the given cursor to find
// the longest matching among the provided options.
func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for cur.Pointer < uint(len(source)) {
		value = append(value, strings.ToLower(string(source[cur.Pointer]))...)
		cur.Pointer++

	match:
		for i, option := range options {
			for _, skip := range skipList {
				if i == skip {
					continue match
				}
			}

			if option == string(value) {
				skipList = append(skipList, i)
				if len(option) > len(match) {
					match = option
				}

				continue
			}

			sharesPrefix := string(value) == option[:cur.Pointer-ic.Pointer]
			tooLong := len(value) > len(option)
			if tooLong || !sharesPrefix {
				skipList = append(skipList, i)
			}
		}

		if len(skipList) == len(options) {
			break
		}
	}

	return match
}

func lexIdentifier(source string, ic cursor) (*token, cursor, bool) {
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		return token, newCursor, true
	}

	cur := ic

	c := source[cur.Pointer]
	isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	if !isAlphabetical {
		return nil, ic, false
	}

	cur.Pointer++
	cur.Loc.col++

	value := []byte{c}
	for ; cur.Pointer < uint(len(source)); cur.Pointer++ {
		c = source[cur.Pointer]

		isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumeric := c >= '0' && c <= '9'
		if isAlphabetical || isNumeric || c == '$' || c == '_' {
			value = append(value, c)
			cur.Loc.col++
			continue
		}

		break
	}

	if len(value) == 0 {
		return nil, ic, false
	}

	return &token{
		Value: strings.ToLower(string(value)),
		Loc:   ic.Loc,
		Kind:  IdentifierKind,
	}, cur, true

}
