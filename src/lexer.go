package src

import (
	"fmt"
	"log"
	"strings"
)

func lex(source string) ([]*token, error) {
	tokens := []*token{}
	cur := cursor{}
lex:
	for cur.pointer < uint(len(source)) {
		lexers := []lexer{lexKeyword, lexSymbol, lexString, lexNumeric, lexIdentifier}
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
			hint = " after " + tokens[len(tokens)-1].value
		}
		for _, t := range tokens {
			log.Println(t.value)
		}
		return nil, fmt.Errorf("Unable to lex tokens%s, at %d:%d", hint, cur.loc.line, cur.loc.col)
	}
	return tokens, nil
}

func lexNumeric(source string, ic cursor) (*token, cursor, bool) {
	cur := ic

	foundPeriod := false
	foundExpMarker := false

	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]
		cur.loc.col++

		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExpmarker := c == 'e'

		// starting with a digit or period
		if cur.pointer == ic.pointer {
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

			if cur.pointer == uint(len(source)-1) {
				return nil, ic, false
			}

			cNext := source[cur.pointer+1]
			if cNext == '-' || cNext == '+' {
				cur.pointer++
				cur.loc.col++
			}

			continue
		}

		if !isDigit {
			break
		}
	}

	if cur.pointer == ic.pointer {
		return nil, ic, false
	}

	return &token{
		value: source[ic.pointer:cur.pointer],
		loc:   ic.loc,
		kind:  NumericKind,
	}, cur, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*token, cursor, bool) {
	cur := ic

	if len(source[cur.pointer:]) == 0 {
		return nil, ic, false
	}

	if source[cur.pointer] != delimiter {
		return nil, ic, false
	}

	cur.loc.col++
	cur.pointer++

	var value []byte
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c := source[cur.pointer]

		if c == delimiter {
			// SQL escapes are via double characters, not backslash.
			if cur.pointer+1 >= uint(len(source)) || source[cur.pointer+1] != delimiter {
				cur.pointer++ // TODO: fix this duplicated code
				cur.loc.col++
				return &token{
					value: string(value),
					loc:   ic.loc,
					kind:  StringKind,
				}, cur, true
			} else {
				value = append(value, delimiter)
				cur.pointer++
				cur.loc.col++
			}
		}

		value = append(value, c)
		cur.loc.col++
	}

	return nil, ic, false
}

func lexString(source string, ic cursor) (*token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lexSymbol(source string, ic cursor) (*token, cursor, bool) {
	c := source[ic.pointer]
	cur := ic

	cur.pointer++
	cur.loc.col++

	switch c {
	// Syntax that should be thrown away
	case '\n':
		cur.loc.line++
		cur.loc.col = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cur, true
	}

	// Syntax that should be kept
	symbols := []symbol{
		Equal,
		XEqual,
		Concat,
		Plus,
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

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.col = ic.loc.col + uint(len(match))

	return &token{
		value: match,
		loc:   ic.loc,
		kind:  SymbolKind,
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
		Where,
		Insert,
		Into,
		Values,
		Int,
		Text,
		And,
		Or,
		True,
		False,
	}

	var options []string
	for _, k := range keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cur.pointer = ic.pointer + uint(len(match))
	cur.loc.col = ic.loc.col + uint(len(match))

	kind := KeywordKind
	if match == string(True) || match == string(False) {
		kind = BoolKind
	}

	return &token{
		value: match,
		kind:  kind,
		loc:   ic.loc,
	}, cur, true
}

// longestMatch iter through a source string starting at the given cursor to find
// the longest matching among the provided options.
func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skipList []int
	var match string

	cur := ic

	for cur.pointer < uint(len(source)) {
		value = append(value, strings.ToLower(string(source[cur.pointer]))...)
		cur.pointer++

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

			sharesPrefix := string(value) == option[:cur.pointer-ic.pointer]
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

	c := source[cur.pointer]
	isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	if !isAlphabetical {
		return nil, ic, false
	}

	cur.pointer++
	cur.loc.col++

	value := []byte{c}
	for ; cur.pointer < uint(len(source)); cur.pointer++ {
		c = source[cur.pointer]

		isAlphabetical := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumeric := c >= '0' && c <= '9'
		if isAlphabetical || isNumeric || c == '$' || c == '_' {
			value = append(value, c)
			cur.loc.col++
			continue
		}

		break
	}

	if len(value) == 0 {
		return nil, ic, false
	}

	return &token{
		value: strings.ToLower(string(value)),
		loc:   ic.loc,
		kind:  IdentifierKind,
	}, cur, true

}
