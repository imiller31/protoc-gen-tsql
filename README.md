# protoc-gen-tsql
a protoc plugin for creating fully compatible tsql schema

CURRENTLY A POC

## Use Case
Storing protocol buffers is hard. Many people create fullly releational mappings of their protocol buffers with extensive schemas and retrieval procedure.
This method requires consistant schema/stored procedure updates and has corner cases which break full forward and backwards compatibility.

The alternative is to store the protobuf in its binary form and look up each protobuf with identifying keys.

This project aims to use protoc to generate a simple protobuf storage schema, allowing the user to use custom descriptors to choose primary keys, indices, and store their protobuf.

We use the proto descriptor number as the column names as they are immutable in the proto world (at least for best practices).
## Example Proto with Descriptors
```proto
syntax = "proto3";

package example;

option go_package = "github.com/imiller31/protoc-gen-tsql/testdata/generated/example";

import "google/protobuf/timestamp.proto";
import "tsql_options/tsql_options.proto";

message Person {
  string name = 1 [
    (tsql_options.tsql_column) = true,
    (tsql_options.tsql_primaryKey) = true,
    (tsql_options.tsql_type) = "NVARCHAR(64)"
  ];
  string uuid = 2 [
    (tsql_options.tsql_column) = true,
    (tsql_options.tsql_primaryKey) = true,
    (tsql_options.tsql_type) = "UNIQUEIDENTIFIER"
  ];

  google.protobuf.Timestamp last_updated_time = 3 [
    (tsql_options.tsql_column) = true,
    (tsql_options.tsql_type) = "DATETIME2"
  ];
}
```
## Example Generated Schema
```tsql
IF  NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Person]') AND type in (N'U'))
BEGIN
CREATE TABLE [dbo].[Person] (
	[seqID] BIGINT IDENTITY(1,1) NOT NULL,
	[1] NVARCHAR(64) NOT NULL,
	[2] UNIQUEIDENTIFIER NOT NULL,
	[3] DATETIME2 NOT NULL,
	[body] VARBINARY(MAX)
CONSTRAINT PK_Person_UNIQUE PRIMARY KEY ([1], [2])
)
END
```
