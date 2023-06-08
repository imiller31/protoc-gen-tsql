package tsqlschemafy

import (
	"bytes"
	"embed"
	"sort"
	"strconv"
	"text/template"
)

type Column struct {
	Name         string
	SqlType      string
	IsPrimaryKey bool
}

type Table struct {
	TableName string
	Columns   map[string]*Column
}

type StoredProcedure struct {
	TableName           string
	StoredProcedureType string
	Parameters          map[string]*Column
}

type Database struct {
	DatabaseName         string
	TablesAndStoredProcs []*TableAndProcs
}

type TableAndProcs struct {
	Table *Table
	Procs map[string]*StoredProcedure
}

//go:embed templates
var sqlTemplates embed.FS

var funcs = template.FuncMap{
	"sub": func(a, b int) int {
		return a - b
	},
	"add": func(a, b int) int {
		return a + b
	},
}

func (db *Database) Generate(p *TsqlfyModule) error {
	migrationPrefix := 0
	for _, tbl := range db.TablesAndStoredProcs {
		migrationPrefix, err := tbl.Table.GenerateMigrations(p, migrationPrefix)
		if err != nil {
			return err
		}
		//sort the stored procs for consistency
		keys := make([]string, 0, len(tbl.Procs))
		for k := range tbl.Procs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if err := tbl.Procs[k].GenerateMigrations(p, migrationPrefix); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sp *StoredProcedure) GenerateMigrations(p *TsqlfyModule, migrationPrefix int) error {
	t := template.New("create_get_sp.up.tmpl").Funcs(funcs)
	t, err := t.ParseFS(sqlTemplates, "templates/create_get_sp.up.tmpl")
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, sp); err != nil {
		return err
	}

	p.AddGeneratorFile(
		strconv.Itoa(migrationPrefix)+
			"_create_sp_"+
			sp.StoredProcedureType+"_"+
			sp.TableName+".up.sql",
		out.String(),
	)
	migrationPrefix++
	return err
}

func (tbl *Table) GenerateMigrations(p *TsqlfyModule, migrationPrefix int) (int, error) {
	t := template.New("create_table.up.tmpl").Funcs(funcs)
	t, err := t.ParseFS(sqlTemplates, "templates/create_table.up.tmpl")
	if err != nil {
		return migrationPrefix, err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, tbl); err != nil {
		return migrationPrefix, err
	}

	p.AddGeneratorFile(
		strconv.Itoa(migrationPrefix)+"_create_table_"+tbl.TableName+".up.sql",
		out.String(),
	)
	migrationPrefix++

	// recreate the map with the column numbers as the keys
	cols := make(map[string]*Column)
	for _, val := range tbl.Columns {
		cols[val.Name] = val
	}

	// Create a slice of keys
	keys := make([]string, 0, len(cols))
	for k := range cols {
		keys = append(keys, k)
	}
	// Sort the slice of keys
	sort.Strings(keys)

	pKeys := make([]string, 0, len(cols))
	for _, k := range keys {

		col := cols[k]
		p.BuildContext.Logf("Column: %v, Key: %s", col, k)
		colWriter := struct {
			TableName string
			Name      string
			SqlType   string
		}{
			TableName: tbl.TableName,
			Name:      col.Name,
			SqlType:   col.SqlType,
		}
		t = template.New("create_column.up.tmpl").Funcs(funcs)
		t, err = t.ParseFS(sqlTemplates, "templates/create_column.up.tmpl")
		if err != nil {
			return migrationPrefix, err
		}
		out.Reset()
		if err := t.Execute(&out, colWriter); err != nil {
			return migrationPrefix, err
		}
		p.AddGeneratorFile(
			strconv.Itoa(migrationPrefix)+"_create_column_"+col.Name+".up.sql",
			out.String(),
		)
		if col.IsPrimaryKey {
			pKeys = append(pKeys, col.Name)
		}
		migrationPrefix++
	}
	pkeyWriter := struct {
		TableName         string
		PrimaryKeyColumns []string
	}{
		TableName:         tbl.TableName,
		PrimaryKeyColumns: pKeys,
	}
	t = template.New("create_pkey.up.tmpl").Funcs(funcs)
	t, err = t.ParseFS(sqlTemplates, "templates/create_pkey.up.tmpl")
	if err != nil {
		return migrationPrefix, err
	}
	out.Reset()
	if err := t.Execute(&out, pkeyWriter); err != nil {
		return migrationPrefix, err
	}
	p.AddGeneratorFile(
		strconv.Itoa(migrationPrefix)+"_create_pkey_"+tbl.TableName+".up.sql",
		out.String(),
	)
	migrationPrefix++

	return migrationPrefix, err
}
