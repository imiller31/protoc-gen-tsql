package tsqlschemafy

import (
	"bytes"
	"embed"
	"text/template"

	pgs "github.com/lyft/protoc-gen-star/v2"
)

//go:embed templates/create_table.tmpl
var createTableTmpl embed.FS

var funcs = template.FuncMap{
	"sub": func(a, b int) int {
		return a - b
	},
}

type TsqlfyModule struct {
	*pgs.ModuleBase
}

func TsqlSchemafy() *TsqlfyModule { return &TsqlfyModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *TsqlfyModule) Name() string { return "TsqlSchemafy" }

func (p *TsqlfyModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	for _, f := range targets {
		p.generateSchema(f)
	}

	return p.Artifacts()
}

func (p *TsqlfyModule) generateSchema(f pgs.File) {
	p.Push(f.Name().String())
	defer p.Pop()

	schema := TsqlfySchema{}

	v := initTsqlfyVisitor(p.BuildContext, &schema)
	p.CheckErr(pgs.Walk(v, f), "unable to construct TSQL schema")

	p.BuildContext.Logf("TSQL Schema:\n%s", schema)
	t := template.New("create_table.tmpl").Funcs(funcs)
	t, err := t.ParseFS(createTableTmpl, "templates/create_table.tmpl")
	if err != nil {
		p.Fail("unable to parse template", err)
	}
	var out bytes.Buffer
	if err := t.Execute(&out, schema); err != nil {
		p.Fail("unable to execute template", err)
	}

	p.Logf("TSQL:\n%s", out.String())

	p.AddGeneratorFile(
		f.InputPath().SetExt("_schema.sql").String(),
		out.String(),
	)
}
