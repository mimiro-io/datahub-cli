# Mimiro Datahub CLI

Cli for the Mimiro datahub.

## Usage

```bash
Usage:
  mim [flags]
  mim [command]

Available Commands:
  login       Log in to the datahub
  dataset     Manage datahub datasets from cli
  jobs        Manage datahub jobs from cli
  help        Help about any command

Flags:
      --disable-banner   Set to true to disable the banner (why would you?)
  -h, --help             help for mim

Use "mim [command] --help" for more information about a command.
```

## About

The Mimiro Datahub CLI is used to communicate with the Datahub.

You can use mim dataset to work with datasets.

You can use mim jobs to work with jobs.

You can use mim login to log in to the datahub.  

## Fancy stuff

The following has been added to show off how fancy the help system is:

```go
func (q CDCQuery) BuildQuery() string {
	date := "GETDATE()-1"
	data, err := base64.StdEncoding.DecodeString(q.Request.Since)
	if err == nil {
		dt, _ := time.Parse(time.RFC3339, string(data))
		date = fmt.Sprintf("DATETIMEFROMPARTS( %d, %d, %d, %d, %d, %d, 0)",
			dt.Year(), dt.Month(), dt.Day(), dt.Hour(), dt.Minute(), dt.Second())
	}
	query := fmt.Sprintf(`
		DECLARE @begin_time DATETIME, @end_time DATETIME, @begin_lsn BINARY(10), @end_lsn BINARY(10);
		SELECT @begin_time = %s, @end_time = GETDATE();
		SELECT @begin_lsn = sys.fn_cdc_map_time_to_lsn('smallest greater than', @begin_time);
		SELECT @end_lsn = sys.fn_cdc_map_time_to_lsn('largest less than or equal', @end_time);
		IF @begin_lsn IS NOT NULL BEGIN
			SELECT * FROM [cdc].[fn_cdc_get_all_changes_dbo_%s](@begin_lsn,@end_lsn,N'ALL');
		END
	`, date, q.TableDef.TableName)

	return query
}
```

