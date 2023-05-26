package tsqlschemafy

type TsqlfySchema struct {
	TableName string
	Columns   []TsqlColumns

	PrimaryKeyColumns []string
}

type TsqlColumns struct {
	Name    string
	SqlType string
}
