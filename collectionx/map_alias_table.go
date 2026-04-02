package collectionx

import "github.com/DaiYuANg/arcgo/collectionx/mapping"

type tableFluent[R comparable, C comparable, V any] interface {
	WhereRows(predicate func(rowKey R, row map[C]V) bool) *mapping.Table[R, C, V]
	RejectRows(predicate func(rowKey R, row map[C]V) bool) *mapping.Table[R, C, V]
	WhereCells(predicate func(rowKey R, columnKey C, value V) bool) *mapping.Table[R, C, V]
	RejectCells(predicate func(rowKey R, columnKey C, value V) bool) *mapping.Table[R, C, V]
	EachRow(fn func(rowKey R, row map[C]V)) *mapping.Table[R, C, V]
	EachCell(fn func(rowKey R, columnKey C, value V)) *mapping.Table[R, C, V]
	FirstCellWhere(predicate func(rowKey R, columnKey C, value V) bool) (R, C, V, bool)
	AnyCellMatch(predicate func(rowKey R, columnKey C, value V) bool) bool
	AllCellsMatch(predicate func(rowKey R, columnKey C, value V) bool) bool
}
