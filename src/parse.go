package src

import (
	"errors"
	"fmt"
)

func Parse(source string) (*ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	a := ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := parseStatements(tokens, cursor, SemiColon.toToken())
		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("Failed to parse, expected statement")
		}
		cursor = newCursor

		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for {
			_, newCursor, ok = parseToken(tokens, cursor, SemiColon.toToken())
			if ok {
				atLeastOneSemicolon = true
				cursor = newCursor
			} else {
				break
			}

		}

		if !atLeastOneSemicolon {
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("Missing semi-colon between statements")
		}
	}

	return &a, nil
}

func parseStatements(tokens []*token, initialCursor uint, delimiter token) (*Statement, uint, bool) {
	cursor := initialCursor

	semiColonToken := SemiColon.toToken()
	slct, newCursor, ok := parseSelectStatement(tokens, cursor, semiColonToken)
	if ok {
		return &Statement{
			Kind:   SelectAstKind,
			Select: slct,
		}, newCursor, true
	}

	inst, newCursor, ok := parseInsertStatement(tokens, cursor, semiColonToken)
	if ok {
		return &Statement{
			Kind:   InsertAstKind,
			Insert: inst,
		}, newCursor, true
	}

	crt, newCursor, ok := parseCreateStatement(tokens, cursor, semiColonToken)
	if ok {
		return &Statement{
			Kind:   CreateAstKind,
			Create: crt,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

func parseInsertStatement(tokens []*token, initialCursor uint, delimiter token) (*InsertStatement, uint, bool) {
	cursor := initialCursor
	var ok bool

	// Insert
	_, cursor, ok = parseToken(tokens, cursor, Insert.toToken())
	if !ok {
		return nil, initialCursor, false
	}

	// Into
	_, cursor, ok = parseToken(tokens, cursor, Into.toToken())
	if !ok {
		helpMessage(tokens, cursor, "Expected into")
		return nil, initialCursor, false
	}

	// Table name
	table, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	// VALUES
	_, cursor, ok = parseToken(tokens, cursor, Values.toToken())
	if !ok {
		helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}

	// Left paren
	_, cursor, ok = parseToken(tokens, cursor, LeftParen.toToken())
	if !ok {
		helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}

	// Expression list
	values, newCursor, ok := parseExpressions(tokens, cursor, []token{RightParen.toToken()})
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	// Right paren
	_, cursor, ok = parseToken(tokens, cursor, RightParen.toToken())
	if !ok {
		helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}

	return &InsertStatement{
		table:  *table,
		values: values,
	}, cursor, true
}

func parseCreateStatement(tokens []*token, initialCursor uint, delimiter token) (*CreateTableStatement, uint, bool) {
	cursor := initialCursor
	var ok bool

	_, cursor, ok = parseToken(tokens, cursor, Create.toToken())
	if !ok {
		return nil, initialCursor, false
	}

	_, cursor, ok = parseToken(tokens, cursor, Table.toToken())
	if !ok {
		return nil, initialCursor, false
	}

	name, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = parseToken(tokens, cursor, LeftParen.toToken())
	if !ok {
		helpMessage(tokens, cursor, "Expected left parenthesis")
		return nil, initialCursor, false
	}

	cols, newCursor, ok := parseColumnDefinitions(tokens, cursor, RightParen.toToken())
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	_, cursor, ok = parseToken(tokens, cursor, RightParen.toToken())
	if !ok {
		helpMessage(tokens, cursor, "Expected right parenthesis")
		return nil, initialCursor, false
	}

	return &CreateTableStatement{
		name: *name,
		cols: cols,
	}, cursor, true
}

func parseColumnDefinitions(tokens []*token, initialCursor uint, delimiter token) (*[]*columnDefinition, uint, bool) {
	cursor := initialCursor

	cds := []*columnDefinition{}
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		if delimiter.equals(current) {
			break
		}

		if len(cds) > 0 {
			var ok bool
			_, cursor, ok = parseToken(tokens, cursor, Comma.toToken())
			if !ok {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		id, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor

		ty, newCursor, ok := parseTokenKind(tokens, cursor, KeywordKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column type")
			return nil, initialCursor, false
		}
		cursor = newCursor

		primaryKey := false
		_, cursor, ok = parseToken(tokens, cursor, PrimaryKey.toToken())
		if ok {
			primaryKey = true
		}

		cds = append(cds, &columnDefinition{
			name:       *id,
			dataType:   *ty,
			primaryKey: primaryKey,
		})
	}

	return &cds, cursor, true
}

func parseSelectStatement(tokens []*token, initialCursor uint, delimiter token) (*SelectStatement, uint, bool) {
	var ok bool
	cursor := initialCursor
	_, cursor, ok = parseToken(tokens, cursor, Select.toToken())
	if !ok {
		return nil, initialCursor, false
	}

	slct := SelectStatement{}

	fromToken := From.toToken()
	item, newCursor, ok := parseSelectItem(tokens, cursor, []token{fromToken, delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = item
	cursor = newCursor

	whereToken := Where.toToken()

	_, cursor, ok = parseToken(tokens, cursor, fromToken)
	if ok {
		from, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM item")
			return nil, initialCursor, false
		}
		slct.from = from
		cursor = newCursor
	}

	_, cursor, ok = parseToken(tokens, cursor, whereToken)
	if ok {
		where, newCursor, ok := parseExpression(tokens, cursor, []token{delimiter}, 0)
		if !ok {
			helpMessage(tokens, cursor, "Expected WHERE conditionals")
			return nil, initialCursor, false
		}
		slct.where = where
		cursor = newCursor
	}

	return &slct, cursor, true
}

func parseSelectItem(tokens []*token, initialCursor uint, delimiters []token) (*[]*selectItem, uint, bool) {
	cursor := initialCursor

	var s []*selectItem
outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		for _, de := range delimiters {
			if de.equals(current) {
				break outer
			}
		}

		var ok bool
		if len(s) > 0 {
			_, cursor, ok = parseToken(tokens, cursor, Comma.toToken())
			if !ok {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		var si selectItem
		_, cursor, ok = parseToken(tokens, cursor, Asterisk.toToken())
		if ok {
			si = selectItem{asterisk: true}
		} else {
			asToken := As.toToken()
			delimiters := append(delimiters, Comma.toToken(), asToken)
			exp, newCursor, ok := parseExpression(tokens, cursor, delimiters, 0)
			if !ok {
				helpMessage(tokens, cursor, "Expected expression")
				return nil, initialCursor, false
			}

			cursor = newCursor
			si.exp = exp

			_, cursor, ok = parseToken(tokens, cursor, asToken)
			if ok {
				ide, newCursor, ok := parseTokenKind(tokens, cursor, IdentifierKind)
				if !ok {
					helpMessage(tokens, cursor, "Expected identifier after AS")
					return nil, initialCursor, false
				}

				cursor = newCursor
				si.as = ide
			}
		}

		s = append(s, &si)
	}

	return &s, cursor, true
}

func parseExpression(tokens []*token, initialCursor uint, delimiters []token, minBp uint) (*expression, uint, bool) {
	cursor := initialCursor

	var exp *expression
	var ok bool
	_, newCursor, ok := parseToken(tokens, cursor, LeftParen.toToken())
	if ok {
		cursor = newCursor
		rightParenToken := RightParen.toToken()

		exp, newCursor, ok = parseExpression(tokens, cursor, append(delimiters, rightParenToken), minBp)
		if !ok {
			helpMessage(tokens, cursor, "Expected expression after opening paren")
			return nil, initialCursor, false
		}
		cursor = newCursor

		_, cursor, ok = parseToken(tokens, cursor, rightParenToken)
		if !ok {
			helpMessage(tokens, cursor, "Expected closing paren")
			return nil, initialCursor, false
		}

	} else {
		exp, cursor, ok = parseLiteralExpression(tokens, cursor)
		if !ok {
			return nil, initialCursor, false
		}
	}

	lastCursor := cursor
outer:
	for cursor < uint(len(tokens)) {
		for _, d := range delimiters {
			_, _, ok = parseToken(tokens, cursor, d)
			if ok {
				break outer
			}
		}

		binOps := []token{
			And.toToken(),
			Or.toToken(),
			Equal.toToken(),
			XEqual.toToken(),
			Comma.toToken(),
			Plus.toToken(),
		}

		var op *token = nil
		for _, bo := range binOps {
			var t *token
			t, cursor, ok = parseToken(tokens, cursor, bo)
			if ok {
				op = t
				break
			}
		}

		if op == nil {
			helpMessage(tokens, cursor, "Expected binary operator")
			return nil, initialCursor, false
		}

		bp := op.bindingPower()
		if bp < minBp {
			cursor = lastCursor
			break
		}

		b, newCursor, ok := parseExpression(tokens, cursor, delimiters, bp)
		if !ok {
			helpMessage(tokens, cursor, "Expected right operand")
			return nil, initialCursor, false
		}

		exp = &expression{
			binary: &binaryExpression{
				*exp,
				*b,
				*op,
			},
			kind: binaryKind,
		}
		cursor = newCursor
		lastCursor = cursor

	}

	return exp, cursor, true
}

func parseLiteralExpression(tokens []*token, initialCursor uint) (*expression, uint, bool) {
	cursor := initialCursor
	kinds := []tokenKind{IdentifierKind, NumericKind, StringKind, BoolKind}
	for _, kind := range kinds {
		t, newCursor, ok := parseTokenKind(tokens, cursor, kind)
		if ok {
			return &expression{
				literal: t,
				kind:    literal,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

func parseExpressions(tokens []*token, initialCursor uint, delimiters []token) (*[]*expression, uint, bool) {
	cursor := initialCursor

	exps := []*expression{}
outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		for _, delimiter := range delimiters {
			if delimiter.equals(current) {
				break outer
			}
		}

		var ok bool
		if len(exps) > 0 {
			_, cursor, ok = parseToken(tokens, cursor, Comma.toToken())
			if !ok {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		exp, newCursor, ok := parseExpression(tokens, cursor, []token{Comma.toToken(), RightParen.toToken()}, 0)
		if !ok {
			helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		cursor = newCursor

		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

func parseToken(tokens []*token, initialCursor uint, t token) (*token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	if p := tokens[cursor]; t.equals(p) {
		return p, cursor + 1, true
	}

	return nil, initialCursor, false
}

func parseTokenKind(tokens []*token, initialCursor uint, kind tokenKind) (*token, uint, bool) {
	cursor := initialCursor
	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]
	if current.kind == kind {
		return current, cursor + 1, true
	}

	return nil, initialCursor, false
}

func helpMessage(tokens []*token, cursor uint, msg string) {
	var c *token
	if cursor+1 < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor]
	}

	fmt.Printf("[%d,%d]: %s, near: %s\n", c.loc.line, c.loc.col, msg, c.value)
}
