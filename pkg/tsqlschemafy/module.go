package tsqlschemafy

import (
	pgs "github.com/lyft/protoc-gen-star/v2"
)

type TsqlfyModule struct {
	*pgs.ModuleBase

	database *Database
}

func TsqlSchemafy() *TsqlfyModule { return &TsqlfyModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *TsqlfyModule) Name() string { return "TsqlSchemafy" }

func (p *TsqlfyModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	p.database = &Database{}

	var heads []pgs.File
	var visited = make(map[string]bool)

	for _, f := range targets {
		if f.Dependents() != nil {
			heads = append(heads, f)
		}
	}

	for _, h := range heads {
		if visited[h.Name().String()] {
			continue
		}
		p.BuildContext.Logf("Generating schema for %s", h.Name().String())
		tableAndProcs := &TableAndProcs{}
		p.generateTableSchema(h, tableAndProcs)
		for _, d := range h.Dependents() {
			if d.Services() != nil {
				p.generateStoredProcs(d, tableAndProcs)
				visited[d.Name().String()] = true
			}
		}
		p.database.TablesAndStoredProcs = append(p.database.TablesAndStoredProcs, tableAndProcs)
		visited[h.Name().String()] = true
	}
	if err := p.database.Generate(p); err != nil {
		p.BuildContext.Failf("Error generating schema: %s", err)
	}
	return p.Artifacts()
}

func (p *TsqlfyModule) generateStoredProcs(f pgs.File, tnp *TableAndProcs) {
	p.Push(f.Name().String())
	defer p.Pop()

	p.BuildContext.Logf("Generating stored procs for %s", f.Name().String())
	tnp.Procs = make(map[string]*StoredProcedure)
	v := initTsqlServiceVisitor(p.BuildContext, tnp)
	p.CheckErr(pgs.Walk(v, f), "unable to construct TSQL stored procedures")
	p.BuildContext.Logf("TSQL Stored Procedures: %s", tnp.Procs)
}

func (p *TsqlfyModule) generateTableSchema(f pgs.File, tnp *TableAndProcs) {
	p.Push(f.Name().String())
	defer p.Pop()

	tnp.Table = &Table{Columns: make(map[string]*Column)}

	v := initTsqlfyVisitor(p.BuildContext, tnp.Table)
	p.CheckErr(pgs.Walk(v, f), "unable to construct TSQL schema")
	tnp.Table.Columns["body"] = &Column{Name: "body", SqlType: "nvarchar(max)"}
	p.BuildContext.Logf("TSQL Schema: %s", tnp.Table.TableName)
}
