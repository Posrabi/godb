package src

import "fmt"

type ast struct {
	Statements []*Statement
}

type astKind uint

const (
	SelectAstKind astKind = iota
	CreateAstKind
	InsertAstKind
)

type Statement struct {
	Select *SelectStatement
	Create *CreateTableStatement
	Insert *InsertStatement
	Kind   astKind
}

type InsertStatement struct {
	table  token
	values *[]*expression
}

type CreateTableStatement struct {
	name token
	cols *[]*columnDefinition
}

type CreateIndexStatement struct {
	table      token
	name       token
	unique     bool
	primaryKey bool
	exp        expression
}

type SelectStatement struct {
	item  *[]*selectItem
	from  *token
	where *expression
}

type expressionKind uint

const (
	literal expressionKind = iota
	binaryKind
)

type binaryExpression struct {
	a  expression
	b  expression
	op token
}

func (be binaryExpression) generateCode() string {
	return fmt.Sprintf("(%s %s %s)", be.a.generateCode(), be.op.value, be.b.generateCode())
}

type expression struct {
	literal *token
	binary  *binaryExpression
	kind    expressionKind
}

func (e expression) generateCode() string {
	switch e.kind {
	case literal:
		switch e.literal.kind {
		case IdentifierKind:
			return fmt.Sprintf("\"%s\"", e.literal.value)
		case StringKind:
			return fmt.Sprintf("%s", e.literal.value)
		default:
			return fmt.Sprintf(e.literal.value)
		}
	case binaryKind:
		return e.binary.generateCode()
	}

	return ""
}

type columnDefinition struct {
	name       token
	dataType   token
	primaryKey bool
}

type selectItem struct {
	exp      *expression
	asterisk bool
	as       *token
}
