package src

import (
	"errors"
	"fmt"
)

func tokenFromKeyword(k keyword) token {
	return token{
		Kind:  KeywordKind,
		Value: string(k),
	}
}

func tokenFromSymbol(s symbol) token {
	return token{
		Kind:  KeywordKind,
		Value: string(s),
	}
}

func expectToken(tokens []*token, cursor uint, t token) bool {
	if cursor >= uint(len(tokens)) {
		return false
	}

	return t.equals(tokens[cursor])
}

func helpMessage(tokens []*token, cursor uint, msg string) {
	var c *token
	if cursor < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor-1]
	}

	fmt.Printf("[%d,%d]: %s, got: %s\n", c.Loc.line, c.Loc.col, msg, c.Value)
}

func parse(source string) (*ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	a := ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := parseStatements(tokens, cursor, tokenFromSymbol(SemiColon))
		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("Failed to parse, expected statement")
		}

		cursor = newCursor

		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for expectToken(tokens, cursor, tokenFromSymbol(SemiColon)) {
			cursor++
			atLeastOneSemicolon = true
		}

		if !atLeastOneSemicolon {
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("Missing semi-colon between statements")
		}
	}

	return &a, nil
}

func parseStatements(tokens []*token, initialCursor uint, delimiter token) (*statement, uint, bool) {
	cursor := initialCursor

	semiColonToken := tokenFromSymbol(SemiColon)
	slct, newCursor, ok := parseSelectStatement(tokens, cursor, semiColonToken)
	if ok {
		return &statement{
			Kind:   SelectAstKind,
			Select: slct,
		}, newCursor, true
	}

	inst, newCursor, ok := parseInsertStatement(tokens, cursor, semiColonToken)
	if ok {
		return &statement{
			Kind:   InsertAstKind,
			Insert: inst,
		}, newCursor, true
	}

	crt, newCursor, ok := parseCreateStatement(tokens, cursor, semiColonToken)
	if ok {
		return &statement{
			Kind:   CreateAstKind,
			Create: crt,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

func parseSelectStatement(tokens []*token, initialCursor uint, delimiter token) (*selectStatement, uint, bool) {
	cursor := initialCursor
	if !expectToken(tokens, cursor, tokenFromKeyword(Select)) {
		return nil, initialCursor, false
	}
	cursor++

	slct := selectStatement{}

	exps, newCursor, ok := parseExpressions(tokens, cursor, []token{tokenFromKeyword(From), delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = *exps
	cursor = newCursor

	if expectToken(tokens, cursor, tokenFromKeyword(From)) {
		cursor++

		from, newCursor, ok := parseToken(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM token")
			return nil, initialCursor, false
		}

		slct.from = *from
		cursor = newCursor
	}

	return &slct, cursor, true
}

func parseToken(tokens []*token, initialCursor uint, kind tokenKind) (*token, uint, bool) {
	cursor := initialCursor
	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]
	if current.Kind == kind {
		return current, cursor + 1, true
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

		if len(exps) > 0 {
			if !expectToken(tokens, cursor, tokenFromSymbol(Comma)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}

			cursor++
		}

		exp, newCursor, ok := parseExpression(tokens, cursor, tokenFromSymbol(Comma))
		if !ok {
			helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		cursor = newCursor

		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

func parseExpression(tokens []*token, initialCursor uint, _ token) (*expression, uint, bool) {
	cursor := initialCursor

	kinds := []tokenKind{IdentifierKind, NumericKind, StringKind}
	for _, kind := range kinds {
		t, newCursor, ok := parseToken(tokens, cursor, kind)
		if ok {
			return &expression{
				literal: t,
				kind:    literal,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}

func parseInsertStatement(tokens []*token, initialCursor uint, delimiter token) (*insertStatement, uint, bool) {
	cursor := initialCursor

	// Insert
	if !expectToken(tokens, cursor, tokenFromKeyword(Insert)) {
		return nil, initialCursor, false
	}
	cursor++

	// Into
	if !expectToken(tokens, cursor, tokenFromKeyword(Into)) {
		helpMessage(tokens, cursor, "Expected into")
		return nil, initialCursor, false
	}
	cursor++

	// Table name
	table, newCursor, ok := parseToken(tokens, cursor, IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	// VALUES
	if !expectToken(tokens, cursor, tokenFromKeyword(Values)) {
		helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}
	cursor++

	// Left paren
	if !expectToken(tokens, cursor, tokenFromSymbol(LeftParen)) {
		helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}
	cursor++

	// Expression list
	values, newCursor, ok := parseExpressions(tokens, cursor, []token{tokenFromSymbol(RightParen)})
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	// Right paren
	if !expectToken(tokens, cursor, tokenFromSymbol(RightParen)) {
		helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}
	cursor++

	return &insertStatement{
		table:  *table,
		values: values,
	}, cursor, true
}

func parseCreateStatement(tokens []*token, initialCursor uint, delimiter token) (*createStatement, uint, bool) {
	cursor := initialCursor

	if !expectToken(tokens, cursor, tokenFromKeyword(Create)) {
		return nil, initialCursor, false
	}
	cursor++

	if !expectToken(tokens, cursor, tokenFromKeyword(Table)) {
		return nil, initialCursor, false
	}
	cursor++

	name, newCursor, ok := parseToken(tokens, cursor, IdentifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(LeftParen)) {
		helpMessage(tokens, cursor, "Expected left parenthesis")
		return nil, initialCursor, false
	}
	cursor++

	cols, newCursor, ok := parseColumnDefinitions(tokens, cursor, tokenFromSymbol(RightParen))
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(RightParen)) {
		helpMessage(tokens, cursor, "Expected right parenthesis")
		return nil, initialCursor, false
	}
	cursor++

	return &createStatement{
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
			if !expectToken(tokens, cursor, tokenFromSymbol(Comma)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
			cursor++
		}

		id, newCursor, ok := parseToken(tokens, cursor, IdentifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}

		cursor = newCursor

		ty, newCursor, ok := parseToken(tokens, cursor, KeywordKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column type")
			return nil, initialCursor, false
		}
		cursor = newCursor

		cds = append(cds, &columnDefinition{
			name:     *id,
			dataType: *ty,
		})
	}

	return &cds, cursor, true
}
