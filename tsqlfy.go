package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/imiller31/protoc-gen-tsql/proto-ext-tsql/tsql_options"
	pgs "github.com/lyft/protoc-gen-star/v2"
)

type TsqlfyModule struct {
	*pgs.ModuleBase
}

type TsqlfySchema struct {
	tableName   string
	columnNames []tsqlColumns

	primaryKeys []string
}

type tsqlColumns struct {
	name    string
	sqlType string
}

func Tsqlfy() *TsqlfyModule { return &TsqlfyModule{ModuleBase: &pgs.ModuleBase{}} }

func (p *TsqlfyModule) Name() string { return "tsqlfy" }

func (p *TsqlfyModule) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	buf := &bytes.Buffer{}

	for _, f := range targets {
		p.printFile(f, buf)
	}

	return p.Artifacts()
}

func (p *TsqlfyModule) printFile(f pgs.File, buf *bytes.Buffer) {
	p.Push(f.Name().String())
	defer p.Pop()

	buf.Reset()

	schema := TsqlfySchema{}

	v := initTsqlfyVisitor(p.BuildContext, &schema)
	p.CheckErr(pgs.Walk(v, f), "unable to construct TSQL schema")

	p.BuildContext.Logf("TSQL Schema:\n%s", schema)
	out := generateTsql(schema)

	p.Logf("TSQL:\n%s", out)

	p.AddGeneratorFile(
		f.InputPath().SetExt("_schema.sql").String(),
		out,
	)
}

type TsqlfyVisitor struct {
	pgs.Visitor
	w io.Writer

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
	v.schema.tableName = m.Name().String()
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
		column := tsqlColumns{
			name:    strconv.Itoa(int(f.Descriptor().GetNumber())),
			sqlType: sqlType,
		}
		v.schema.columnNames = append(v.schema.columnNames, column)
	}

	if isTsqlPrimaryKey {
		v.buildCtx.Logf("TsqlPrimaryKey: %s", f.Name().String())
		v.schema.primaryKeys = append(v.schema.primaryKeys, strconv.Itoa(int(f.Descriptor().GetNumber())))
	}

	v.buildCtx.Logf("TsqlSchema: %s", v.schema)
	return nil, nil
}

func generateTsql(schema TsqlfySchema) string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("IF  NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[%s]') AND type in (N'U'))", schema.tableName))
	out.WriteString("\nBEGIN\n")
	out.WriteString(fmt.Sprintf("CREATE TABLE [dbo].[%s] (\n", schema.tableName))
	out.WriteString("\t[seqID] BIGINT IDENTITY(1,1) NOT NULL,\n")
	for _, column := range schema.columnNames {
		out.WriteString(fmt.Sprintf("\t[%s] %s NOT NULL,\n", column.name, column.sqlType))
	}
	out.WriteString("\t[body] VARBINARY(MAX)\n")
	out.WriteString(fmt.Sprintf("CONSTRAINT PK_%s_UNIQUE PRIMARY KEY (", schema.tableName))
	for i, pkey := range schema.primaryKeys {
		out.WriteString(fmt.Sprintf("[%s]", pkey))
		if i < len(schema.primaryKeys)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString(")\n)\n")
	out.WriteString("END\n")

	return out.String()
}
