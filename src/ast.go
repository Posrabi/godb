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
	Select *selectStatement
	Create *createTableStatement
	Insert *insertStatement
	Kind   astKind
}

type insertStatement struct {
	table  token
	values *[]*expression
}

type expressionKind uint

const (
	literal expressionKind = iota
)

type expression struct {
	literal *token
	kind    expressionKind
}

type columnDefinition struct {
	name     token
	dataType token
}

type createTableStatement struct {
	name token
	cols *[]*columnDefinition
}

type selectStatement struct {
	item []*expression
	from token
}
