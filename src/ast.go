package src

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

type expression struct {
	literal *token
	binary  *binaryExpression
	kind    expressionKind
}

type columnDefinition struct {
	name     token
	dataType token
}

type CreateTableStatement struct {
	name token
	cols *[]*columnDefinition
}

type SelectStatement struct {
	item  *[]*selectItem
	from  *token
	where *expression
}

type selectItem struct {
	exp      *expression
	asterisk bool
	as       *token
}
