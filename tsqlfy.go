package main

import (
	"bytes"
	"embed"
	"strconv"
	"text/template"

	"github.com/imiller31/protoc-gen-tsql/proto-ext-tsql/tsql_options"
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

type TsqlfySchema struct {
	TableName string
	Columns   []TsqlColumns

	PrimaryKeyColumns []string
}

type TsqlColumns struct {
	Name    string
	SqlType string
}

func Tsqlfy() *TsqlfyModule { return &TsqlfyModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *TsqlfyModule) Name() string { return "tsqlfy" }

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

type TsqlfyVisitor struct {
	pgs.Visitor
	schema *TsqlfySchema

	buildCtx pgs.BuildContext
}

func initTsqlfyVisitor(ctx pgs.BuildContext, schema *TsqlfySchema) pgs.Visitor {
	v := TsqlfyVisitor{
		buildCtx: ctx,

		schema: schema,
	}
	v.Visitor = pgs.PassThroughVisitor(&v)
	return v
}

func (v TsqlfyVisitor) writeSubNode() pgs.Visitor {
	return initTsqlfyVisitor(v.buildCtx, v.schema)
}

func (v TsqlfyVisitor) VisitFile(f pgs.File) (pgs.Visitor, error) {
	return v.writeSubNode(), nil
}

func (v TsqlfyVisitor) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	v.schema.TableName = m.Name().String()
	return v.writeSubNode(), nil
}

func (v TsqlfyVisitor) VisitField(f pgs.Field) (pgs.Visitor, error) {
	var isTsqlColumn bool
	if _, err := f.Extension(tsql_options.E_TsqlColumn, &isTsqlColumn); err != nil {
		v.buildCtx.Logf("Error getting extension: %s", err)
	}

	var isTsqlPrimaryKey bool
	if _, err := f.Extension(tsql_options.E_TsqlPrimaryKey, &isTsqlPrimaryKey); err != nil {
		v.buildCtx.Logf("Error getting extension: %s", err)
	}

	var sqlType string
	if _, err := f.Extension(tsql_options.E_TsqlType, &sqlType); err != nil {
		v.buildCtx.Logf("Error getting extension: %s", err)
	}

	if isTsqlColumn {
		v.buildCtx.Logf("TsqlColumn: %s", f.Name().String())
		column := TsqlColumns{
			Name:    strconv.Itoa(int(f.Descriptor().GetNumber())),
			SqlType: sqlType,
		}
		v.schema.Columns = append(v.schema.Columns, column)
	}

	if isTsqlPrimaryKey {
		v.buildCtx.Logf("TsqlPrimaryKey: %s", f.Name().String())
		v.schema.PrimaryKeyColumns = append(v.schema.PrimaryKeyColumns, strconv.Itoa(int(f.Descriptor().GetNumber())))
	}

	v.buildCtx.Logf("TsqlSchema: %s", v.schema)
	return nil, nil
}
