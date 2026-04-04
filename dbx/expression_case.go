package dbx

import "github.com/DaiYuANg/arcgo/collectionx"

type Aggregate[T any] struct {
	Function AggregateFunction
	Expr     scalarExpression
	Distinct bool
	star     bool
}

type CaseBuilder[T any] struct {
	branches collectionx.List[caseWhenBranch]
}

type CaseExpression[T any] struct {
	Branches collectionx.List[caseWhenBranch]
	Else     any
}

type aliasedSelectItem struct {
	Item  SelectItem
	Alias string
}

func (excludedColumnOperand[T]) expressionNode() {}
func (Aggregate[T]) expressionNode()             {}
func (Aggregate[T]) selectItemNode()             {}
func (CaseExpression[T]) expressionNode()        {}
func (CaseExpression[T]) selectItemNode()        {}
func (aliasedSelectItem) selectItemNode()        {}

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
	b.branches = mergeList(b.branches, collectionx.NewList(caseWhenBranch{Predicate: predicate, Value: value}))
	return b
}

func (b *CaseBuilder[T]) Else(value any) CaseExpression[T] {
	if b == nil {
		return CaseExpression[T]{Else: value}
	}
	return CaseExpression[T]{
		Branches: b.branches.Clone(),
		Else:     value,
	}
}

func (b *CaseBuilder[T]) End() CaseExpression[T] {
	if b == nil {
		return CaseExpression[T]{}
	}
	return CaseExpression[T]{
		Branches: b.branches.Clone(),
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
