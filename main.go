package main

import (
	"github.com/imiller31/protoc-gen-tsql/pkg/tsqlschemafy"
	pgs "github.com/lyft/protoc-gen-star/v2"
)

func main() {
	pgs.Init(
		pgs.DebugEnv("DEBUG"),
	).RegisterModule(
		tsqlschemafy.TsqlSchemafy(),
	).Render()
}
