package src

type location struct {
	line uint
	col  uint
}

type keyword string

const (
	Select     keyword = "select"
	From       keyword = "from"
	As         keyword = "as"
	Table      keyword = "table"
	Create     keyword = "create"
	Insert     keyword = "insert"
	Into       keyword = "into"
	Values     keyword = "values"
	Int        keyword = "int"
	Text       keyword = "text"
	Where      keyword = "where"
	And        keyword = "and"
	Or         keyword = "or"
	True       keyword = "true"
	False      keyword = "false"
	PrimaryKey keyword = "primary key"
)

func (k keyword) toToken() token {
	return token{
		kind:  KeywordKind,
		value: string(k),
	}
}

type symbol string

const (
	SemiColon      symbol = ";"
	Asterisk       symbol = "*"
	Comma          symbol = ","
	LeftParen      symbol = "("
	RightParen     symbol = ")"
	Equal          symbol = "="
	XEqual         symbol = "!="
	Greater        symbol = ">"
	GreaterOrEqual symbol = ">="
	Less           symbol = "<"
	LessOrEqual           = "<="
	Concat         symbol = "||"
	Plus           symbol = "+"
)

func (s symbol) toToken() token {
	return token{
		kind:  SymbolKind,
		value: string(s),
	}
}

type cursor struct {
	pointer uint
	loc     location
}

// A lexer parses any sql command and turns it into code.
type lexer func(string, cursor) (*token, cursor, bool)
