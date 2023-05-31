package shellcmd

import (
	"fmt"
	"strings"

	"github.com/libsql/libsql-shell-go/internal/db"
	"github.com/spf13/cobra"
)

var dumpCmd = &cobra.Command{
	Use:   ".dump",
	Short: "Render database content as SQL",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, ok := cmd.Context().Value(dbCtx{}).(*DbCmdConfig)
		if !ok {
			return fmt.Errorf("missing db connection")
		}

		fmt.Fprintln(config.OutF, "PRAGMA foreign_keys=OFF;")

		getTableNamesStatementResult, err := getDbTableNames(config)
		if err != nil {
			return err
		}

		err = dumpTables(getTableNamesStatementResult, config)
		if err != nil {
			return err
		}

		return nil
	},
}

func dumpTables(getTableStatementResult db.StatementResult, config *DbCmdConfig) error {
	for tableNameRowResult := range getTableStatementResult.RowCh {
		if tableNameRowResult.Err != nil {
			return tableNameRowResult.Err
		}
		formattedRow, err := db.FormatData(tableNameRowResult.Row, db.TABLE)
		if err != nil {
			return err
		}

		formattedTableName := formattedRow[0]

		tableSchemaStatementResult, err := getTableSchema(config, formattedTableName)
		if err != nil {
			return err
		}

		err = dumpTableSchema(tableSchemaStatementResult, config, formattedTableName)
		if err != nil {
			return err
		}

		tableRecordsStatementResult, err := getTableRecords(config, formattedTableName)
		if err != nil {
			return err
		}

		err = dumpTableRecords(tableRecordsStatementResult, config, formattedTableName)
		if err != nil {
			return err
		}
	}

	return nil
}

func dumpTableSchema(tableSchemaStatementResult db.StatementResult, config *DbCmdConfig, tableName string) error {
	for tableSchemaRowResult := range tableSchemaStatementResult.RowCh {
		if tableSchemaRowResult.Err != nil {
			return tableSchemaRowResult.Err
		}
		sql := tableSchemaRowResult.Row[0]
		if sql != nil {
			formattedSql, _ := db.FormatData([]interface{}{sql}, db.TABLE)
			fmt.Fprintln(config.OutF, formattedSql[0])
		}
	}
	return nil
}

func dumpTableRecords(tableRecordsStatementResult db.StatementResult, config *DbCmdConfig, tableName string) error {
	for tableRecordsRowResult := range tableRecordsStatementResult.RowCh {
		if tableRecordsRowResult.Err != nil {
			return tableRecordsRowResult.Err
		}
		insertStatement := "INSERT INTO " + tableName + " VALUES ("

		tableRecordsFormattedRow, err := db.FormatData(tableRecordsRowResult.Row, db.SQLITE)
		if err != nil {
			return err
		}

		insertStatement += strings.Join(tableRecordsFormattedRow, ", ")
		insertStatement += ");"
		fmt.Fprintln(config.OutF, insertStatement)
	}

	return nil
}

func getDbTableNames(config *DbCmdConfig) (db.StatementResult, error) {
	listTablesResult, err := config.Db.ExecuteStatements("SELECT name FROM sqlite_master WHERE type='table' and name not like 'sqlite_%' and name != '_litestream_seq' and name != '_litestream_lock' and name != 'libsql_wasm_func_table'")
	if err != nil {
		return db.StatementResult{}, err
	}

	statementResult := <-listTablesResult.StatementResultCh
	if statementResult.Err != nil {
		return db.StatementResult{}, statementResult.Err
	}

	return statementResult, nil
}

func getTableSchema(config *DbCmdConfig, tableName string) (db.StatementResult, error) {
	tableInfoResult, err := config.Db.ExecuteStatements(
		fmt.Sprintf("SELECT SQL FROM sqlite_master WHERE TBL_NAME='%s'", tableName),
	)
	if err != nil {
		return db.StatementResult{}, err
	}

	statementResult := <-tableInfoResult.StatementResultCh
	if statementResult.Err != nil {
		return db.StatementResult{}, statementResult.Err
	}

	return statementResult, nil
}

func getTableRecords(config *DbCmdConfig, tableName string) (db.StatementResult, error) {
	tableRecordsResult, err := config.Db.ExecuteStatements(
		fmt.Sprintf("SELECT * FROM %s", tableName),
	)
	if err != nil {
		return db.StatementResult{}, err
	}

	statementResult := <-tableRecordsResult.StatementResultCh
	if statementResult.Err != nil {
		return db.StatementResult{}, statementResult.Err
	}

	return statementResult, nil
}
