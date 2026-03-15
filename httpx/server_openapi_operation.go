package httpx

import (
	"github.com/DaiYuANg/archgo/collectionx/list"
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
	if len(lo.Filter(op.Parameters, func(existing *huma.Param, _ int) bool {
		return existing != nil && existing.Name == param.Name && existing.In == param.In
	})) > 0 {
		return
	}
	op.Parameters = append(op.Parameters, cloneParam(param))
}
