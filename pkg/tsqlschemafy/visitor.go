package tsqlschemafy

import (
	"strconv"

	"github.com/imiller31/protoc-gen-tsql/proto-ext-tsql/tsql_options"
	pgs "github.com/lyft/protoc-gen-star/v2"
)

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
