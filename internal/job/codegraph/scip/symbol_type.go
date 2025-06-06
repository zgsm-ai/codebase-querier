package scip

import (
	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// SymbolType 定义
type SymbolType string

// buildSymbol 核心映射函数：根据 SCIP 枚举值返回分类类型
func buildSymbol(occ *scip.Occurrence, path string) *codegraphpb.Symbol {

	s := &codegraphpb.Symbol{
		Identifier: occ.Symbol,
		Path:       path,
		Role:       getSymbolRoleFromOccurrence(occ),
		Range:      occ.Range,
	}

	parsedSymbol, err := scip.ParseSymbol(occ.Symbol)
	if err != nil {
		logx.Debugf("parser symbol error: %v", err)
		return s
	}
	descriptors := parsedSymbol.Descriptors
	if len(descriptors) == 0 {
		return s
	} // 需要 判断 descriptors的长度； 1是包；2是类、接口、函数；3是接口方法
	// 对go来说，descriptors 对于包只有一个元素；对于非包，有两个以上的元素，其中一个是包；

	s.Namespace = descriptors[0].Name
	s.Type = int32(descriptors[len(descriptors)-1].Suffix)
	s.Name = descriptors[len(descriptors)-1].Name
	return s
}

// 0: "UnspecifiedSuffix"
// 1: "Namespace" / "Package"
// 2: "Type"
// 3: "Term"
// 4: "Method"
// 5: "TypeParameter"
// 6: "Parameter"
// 7: "Meta"
// 8: "Local"
// 9: "Macro"
