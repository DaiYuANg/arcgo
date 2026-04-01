package httpx

import (
	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

func forEachOperation(doc *huma.OpenAPI, fn func(*huma.Operation)) {
	if doc == nil || fn == nil {
		return
	}
	lo.ForEach(lo.Entries(doc.Paths), func(entry lo.Entry[string, *huma.PathItem], _ int) {
		if entry.Value == nil {
			return
		}
		list.NewList(
			entry.Value.Get, entry.Value.Put, entry.Value.Post, entry.Value.Delete,
			entry.Value.Options, entry.Value.Head, entry.Value.Patch, entry.Value.Trace,
		).Range(func(_ int, op *huma.Operation) bool {
			if op != nil {
				fn(op)
			}
			return true
		})
	})
}

func appendOperationParameter(op *huma.Operation, param *huma.Param) {
	if op == nil || param == nil {
		return
	}
	if lo.SomeBy(op.Parameters, func(existing *huma.Param) bool {
		return existing != nil && existing.Name == param.Name && existing.In == param.In
	}) {
		return
	}
	op.Parameters = lo.Concat(op.Parameters, []*huma.Param{cloneParam(param)})
}
