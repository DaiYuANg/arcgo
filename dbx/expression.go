package dbx

import "github.com/samber/lo"

type Expression interface {
	expressionNode()
}

type scalarExpression interface {
	Expression
	operandRenderer
}

type Predicate interface {
	Expression
	predicateNode()
}

type Condition = Predicate

type SelectItem interface {
	selectItemNode()
}

type Assignment interface {
	assignmentNode()
}

type Order interface {
	orderNode()
}

type ComparisonOperator string

type LogicalOperator string

type JoinType string
type AggregateFunction string

const (
	OpEq    ComparisonOperator = "="
	OpNe    ComparisonOperator = "<>"
	OpGt    ComparisonOperator = ">"
	OpGe    ComparisonOperator = ">="
	OpLt    ComparisonOperator = "<"
	OpLe    ComparisonOperator = "<="
	OpIn    ComparisonOperator = "IN"
	OpLike  ComparisonOperator = "LIKE"
	OpIs    ComparisonOperator = "IS"
	OpIsNot ComparisonOperator = "IS NOT"
)

const (
	LogicalAnd LogicalOperator = "AND"
	LogicalOr  LogicalOperator = "OR"
)

const (
	InnerJoin JoinType = "INNER"
	LeftJoin  JoinType = "LEFT"
	RightJoin JoinType = "RIGHT"
)

const (
	AggCount AggregateFunction = "COUNT"
	AggSum   AggregateFunction = "SUM"
	AggAvg   AggregateFunction = "AVG"
	AggMin   AggregateFunction = "MIN"
	AggMax   AggregateFunction = "MAX"
)

type caseWhenBranch struct {
	Predicate Predicate
	Value     any
}

type valueOperand[T any] struct {
	Value T
}

type columnOperand[T any] struct {
	Column typedColumn[T]
}

type excludedColumnOperand[T any] struct {
	Column ColumnMeta
}

type comparisonPredicate struct {
	Left  scalarExpression
	Op    ComparisonOperator
	Right any
}

func (comparisonPredicate) expressionNode() {}
func (comparisonPredicate) predicateNode()  {}

type logicalPredicate struct {
	Op         LogicalOperator
	Predicates []Predicate
}

func (logicalPredicate) expressionNode() {}
func (logicalPredicate) predicateNode()  {}

type notPredicate struct {
	Predicate Predicate
}

func (notPredicate) expressionNode() {}
func (notPredicate) predicateNode()  {}

type existsPredicate struct {
	Query *SelectQuery
}

func (existsPredicate) expressionNode() {}
func (existsPredicate) predicateNode()  {}

type columnAssignment[E any, T any] struct {
	Column Column[E, T]
	Value  any
}

func (columnAssignment[E, T]) assignmentNode() {}

type columnOrder[E any, T any] struct {
	Column     Column[E, T]
	Descending bool
}

func (columnOrder[E, T]) orderNode() {}

type expressionOrder struct {
	Expr       scalarExpression
	Descending bool
}

func (expressionOrder) orderNode() {}

type Aggregate[T any] struct {
	Function AggregateFunction
	Expr     scalarExpression
	Distinct bool
	star     bool
}

type CaseBuilder[T any] struct {
	branches []caseWhenBranch
}

type CaseExpression[T any] struct {
	Branches []caseWhenBranch
	Else     any
}

func (excludedColumnOperand[T]) expressionNode() {}
func (Aggregate[T]) expressionNode()             {}
func (Aggregate[T]) selectItemNode()             {}
func (CaseExpression[T]) expressionNode()        {}
func (CaseExpression[T]) selectItemNode()        {}

type aliasedSelectItem struct {
	Item  SelectItem
	Alias string
}

func (aliasedSelectItem) selectItemNode() {}

func And(predicates ...Predicate) Predicate {
	items := compactPredicates(predicates)
	if len(items) == 1 {
		return items[0]
	}
	return logicalPredicate{Op: LogicalAnd, Predicates: items}
}

func Or(predicates ...Predicate) Predicate {
	items := compactPredicates(predicates)
	if len(items) == 1 {
		return items[0]
	}
	return logicalPredicate{Op: LogicalOr, Predicates: items}
}

func Not(predicate Predicate) Predicate {
	return notPredicate{Predicate: predicate}
}

func Like[E any](column Column[E, string], pattern string) Predicate {
	return comparisonPredicate{
		Left:  column,
		Op:    OpLike,
		Right: valueOperand[string]{Value: pattern},
	}
}

func Exists(query *SelectQuery) Predicate {
	return existsPredicate{Query: query}
}

func CaseWhen[T any](predicate Predicate, value any) *CaseBuilder[T] {
	return (&CaseBuilder[T]{}).When(predicate, value)
}

func CountAll() Aggregate[int64] {
	return Aggregate[int64]{Function: AggCount, star: true}
}

func Count[E any, T any](expr Column[E, T]) Aggregate[int64] {
	return Aggregate[int64]{Function: AggCount, Expr: expr}
}

func CountDistinct[E any, T any](expr Column[E, T]) Aggregate[int64] {
	return Aggregate[int64]{Function: AggCount, Expr: expr, Distinct: true}
}

func Sum[E any, T any](expr Column[E, T]) Aggregate[T] {
	return Aggregate[T]{Function: AggSum, Expr: expr}
}

func Avg[E any, T any](expr Column[E, T]) Aggregate[float64] {
	return Aggregate[float64]{Function: AggAvg, Expr: expr}
}

func Min[E any, T any](expr Column[E, T]) Aggregate[T] {
	return Aggregate[T]{Function: AggMin, Expr: expr}
}

func Max[E any, T any](expr Column[E, T]) Aggregate[T] {
	return Aggregate[T]{Function: AggMax, Expr: expr}
}

func (b *CaseBuilder[T]) When(predicate Predicate, value any) *CaseBuilder[T] {
	if b == nil {
		b = &CaseBuilder[T]{}
	}
	b.branches = append(b.branches, caseWhenBranch{Predicate: predicate, Value: value})
	return b
}

func (b *CaseBuilder[T]) Else(value any) CaseExpression[T] {
	if b == nil {
		return CaseExpression[T]{Else: value}
	}
	return CaseExpression[T]{
		Branches: append([]caseWhenBranch(nil), b.branches...),
		Else:     value,
	}
}

func (b *CaseBuilder[T]) End() CaseExpression[T] {
	if b == nil {
		return CaseExpression[T]{}
	}
	return CaseExpression[T]{
		Branches: append([]caseWhenBranch(nil), b.branches...),
	}
}

func (a Aggregate[T]) As(alias string) SelectItem {
	return aliasedSelectItem{Item: a, Alias: alias}
}

func (c CaseExpression[T]) As(alias string) SelectItem {
	return aliasedSelectItem{Item: c, Alias: alias}
}

func (a Aggregate[T]) Eq(value T) Predicate {
	return comparisonPredicate{
		Left:  a,
		Op:    OpEq,
		Right: valueOperand[T]{Value: value},
	}
}

func (a Aggregate[T]) Ne(value T) Predicate {
	return comparisonPredicate{
		Left:  a,
		Op:    OpNe,
		Right: valueOperand[T]{Value: value},
	}
}

func (a Aggregate[T]) Gt(value T) Predicate {
	return comparisonPredicate{
		Left:  a,
		Op:    OpGt,
		Right: valueOperand[T]{Value: value},
	}
}

func (a Aggregate[T]) Ge(value T) Predicate {
	return comparisonPredicate{
		Left:  a,
		Op:    OpGe,
		Right: valueOperand[T]{Value: value},
	}
}

func (a Aggregate[T]) Lt(value T) Predicate {
	return comparisonPredicate{
		Left:  a,
		Op:    OpLt,
		Right: valueOperand[T]{Value: value},
	}
}

func (a Aggregate[T]) Le(value T) Predicate {
	return comparisonPredicate{
		Left:  a,
		Op:    OpLe,
		Right: valueOperand[T]{Value: value},
	}
}

func (a Aggregate[T]) Asc() Order {
	return expressionOrder{Expr: a}
}

func (a Aggregate[T]) Desc() Order {
	return expressionOrder{Expr: a, Descending: true}
}

func (c CaseExpression[T]) Eq(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpEq,
		Right: valueOperand[T]{Value: value},
	}
}

func (c CaseExpression[T]) Ne(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpNe,
		Right: valueOperand[T]{Value: value},
	}
}

func (c CaseExpression[T]) Gt(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpGt,
		Right: valueOperand[T]{Value: value},
	}
}

func (c CaseExpression[T]) Ge(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpGe,
		Right: valueOperand[T]{Value: value},
	}
}

func (c CaseExpression[T]) Lt(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpLt,
		Right: valueOperand[T]{Value: value},
	}
}

func (c CaseExpression[T]) Le(value T) Predicate {
	return comparisonPredicate{
		Left:  c,
		Op:    OpLe,
		Right: valueOperand[T]{Value: value},
	}
}

func (c CaseExpression[T]) Asc() Order {
	return expressionOrder{Expr: c}
}

func (c CaseExpression[T]) Desc() Order {
	return expressionOrder{Expr: c, Descending: true}
}

func compactPredicates(predicates []Predicate) []Predicate {
	return lo.Filter(predicates, func(predicate Predicate, _ int) bool {
		return predicate != nil
	})
}

func compactExpressions(expressions []Expression) []Expression {
	return lo.Filter(expressions, func(expression Expression, _ int) bool {
		return expression != nil
	})
}
