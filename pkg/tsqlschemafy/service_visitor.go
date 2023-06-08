package tsqlschemafy

import (
	"strings"

	"github.com/imiller31/protoc-gen-tsql/proto-ext-tsql/tsql_options"
	pgs "github.com/lyft/protoc-gen-star/v2"
)

type TsqlServiceVisitor struct {
	pgs.Visitor
	tnp *TableAndProcs

	buildCtx pgs.BuildContext
}

func initTsqlServiceVisitor(ctx pgs.BuildContext, tnp *TableAndProcs) pgs.Visitor {
	v := TsqlServiceVisitor{
		buildCtx: ctx,
		tnp:      tnp,
	}
	v.Visitor = pgs.PassThroughVisitor(&v)
	return v
}

func (v TsqlServiceVisitor) writeStoredProcedureSubNode() pgs.Visitor {
	return initTsqlServiceVisitor(v.buildCtx, v.tnp)
}

func (v TsqlServiceVisitor) VisitService(s pgs.Service) (pgs.Visitor, error) {
	v.buildCtx.Logf("Service: %s", s.Name())
	return v.writeStoredProcedureSubNode(), nil
}

func (v TsqlServiceVisitor) VisitMethod(s pgs.Method) (pgs.Visitor, error) {
	var generateStoredProcedure bool
	if _, err := s.Extension(tsql_options.E_TsqlGenerateSp, &generateStoredProcedure); err != nil {
		v.buildCtx.Logf("Error getting extension: %s", err)
	}
	if !generateStoredProcedure {
		return v.writeStoredProcedureSubNode(), nil
	}
	sp := &StoredProcedure{Parameters: make(map[string]*Column)}
	returnMsg := s.Output()

	sp.TableName = returnMsg.Name().String()
	sp.StoredProcedureType = getProcType(s.Name().String())
	v.tnp.Procs[s.Name().String()] = sp

	inputMsg := s.Input()
	for _, field := range inputMsg.Fields() {
		if v.tnp.Table.Columns[field.Name().String()] != nil {
			sp.Parameters[field.Name().String()] = v.tnp.Table.Columns[field.Name().String()]
		}
	}

	if len(sp.Parameters) != len(inputMsg.Fields()) {
		v.buildCtx.Failf("Error: %s", "Not all input fields are mapped to table columns")
	}

	v.buildCtx.Logf("StoredProcedure: %s", v.tnp.Procs[s.Name().String()].TableName)
	return v.writeStoredProcedureSubNode(), nil
}

func getProcType(methodName string) string {
	if strings.Contains(strings.ToLower(methodName), "get") {
		return "GET"
	}
	if strings.Contains(strings.ToLower(methodName), "put") {
		return "PUT"
	}
	return ""
}
