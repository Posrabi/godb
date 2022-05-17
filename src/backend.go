package src

import "errors"

type columnType uint

const (
	TextType columnType = iota
	IntType
)

type Cell interface {
	AsText() string
	AsInt() int32
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
)

type Backend interface {
	CreateTable(*createTableStatement) error
	Insert(*insertStatement) error
	Select(*selectStatement)
}
