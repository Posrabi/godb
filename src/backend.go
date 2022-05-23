package src

import "errors"

type columnType uint

const (
	TextType columnType = iota
	IntType
	BoolType
)

type Cell interface {
	AsText() string
	AsInt() int32
	AsBool() bool
}

type Results struct {
	Columns []struct {
		Type columnType
		Name string
	}

	Rows [][]Cell
}

var (
	TableDoesNotExists = errors.New("Table does not exist")
	ColumnDoesNotExist = errors.New("Column does not exist")
	InvalidSelectItem  = errors.New("Select item is not valid")
	InvalidDatatype    = errors.New("Invalid datatype")
	MissingValues      = errors.New("Missing values")
	InvalidCell        = errors.New("Cell is invalid")
	InvalidOperands    = errors.New("Operands are invalid")
)

type Backend interface {
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
}
