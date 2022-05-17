package src

type location struct {
	line uint
	col  uint
}

type keyword string

const (
	Select keyword = "select"
	From   keyword = "from"
	As     keyword = "as"
	Table  keyword = "table"
	Create keyword = "create"
	Insert keyword = "insert"
	Into   keyword = "into"
	Values keyword = "values"
	Int    keyword = "int"
	Text   keyword = "text"
)

type symbol string

const (
	SemiColon  symbol = ";"
	Asterisk   symbol = "*"
	Comma      symbol = ","
	LeftParen         = "("
	RightParen        = ")"
)

type tokenKind uint

const (
	KeywordKind tokenKind = iota
	SymbolKind
	IdentifierKind
	StringKind
	NumericKind
)

type token struct {
	Value string
	Kind  tokenKind
	Loc   location
}

type cursor struct {
	Pointer uint
	Loc     location
}

func (t *token) equals(other *token) bool {
	return t.Value == other.Value && t.Kind == other.Kind
}

// A lexer parses any sql command and turns it into code.
type Lexer func(string, cursor) (*token, cursor, bool)
