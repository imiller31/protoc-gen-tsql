package tsqlschemafy

import (
	"strconv"

	"github.com/imiller31/protoc-gen-tsql/proto-ext-tsql/tsql_options"
	pgs "github.com/lyft/protoc-gen-star/v2"
)

type TsqlTableVisitor struct {
	pgs.Visitor
	table *Table

	buildCtx pgs.BuildContext
}

func initTsqlfyVisitor(ctx pgs.BuildContext, table *Table) pgs.Visitor {
	v := TsqlTableVisitor{
		buildCtx: ctx,
		table:    table,
	}
	v.Visitor = pgs.PassThroughVisitor(&v)
	return v
}

func (v TsqlTableVisitor) writeSchemaSubNode() pgs.Visitor {
	return initTsqlfyVisitor(v.buildCtx, v.table)
}

func (v TsqlTableVisitor) VisitFile(f pgs.File) (pgs.Visitor, error) {
	return v.writeSchemaSubNode(), nil
}

func (v TsqlTableVisitor) VisitMessage(m pgs.Message) (pgs.Visitor, error) {
	v.table.TableName = m.Name().String()
	return v.writeSchemaSubNode(), nil
}

func (v TsqlTableVisitor) VisitField(f pgs.Field) (pgs.Visitor, error) {
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
		v.buildCtx.Logf("Column: %s", f.Name().String())

		v.table.Columns[f.Name().String()] = &Column{
			Name:         strconv.Itoa(int(f.Descriptor().GetNumber())),
			SqlType:      sqlType,
			IsPrimaryKey: isTsqlPrimaryKey,
		}
	}

	v.buildCtx.Logf("TsqlSchema: %s", v.table.TableName)
	return nil, nil
}
