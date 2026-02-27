using System;
using System.Data;
using System.Text.Json;
using Microsoft.Data.SqlClient;

class Program
{
    static async Task Main(string[] args)
    {
        if (args.Length < 1)
        {
            Console.WriteLine("Usage: MetadataFetcher <connection_string>");
            return;
        }

        string connectionString = args[0];
        try
        {
            using var conn = new SqlConnection(connectionString);
            await conn.OpenAsync();

            var schemaQuery = @"
                SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE
                FROM INFORMATION_SCHEMA.COLUMNS
                ORDER BY TABLE_NAME, ORDINAL_POSITION";

            using var cmd = new SqlCommand(schemaQuery, conn);
            using var reader = await cmd.ExecuteReaderAsync();

            var schema = new List<TableInfo>();
            TableInfo currentTable = null;

            while (await reader.ReadAsync())
            {
                var tableName = reader["TABLE_NAME"].ToString();
                var colName = reader["COLUMN_NAME"].ToString();
                var dataType = reader["DATA_TYPE"].ToString();

                if (currentTable == null || currentTable.Name != tableName)
                {
                    currentTable = new TableInfo { Name = tableName };
                    schema.Add(currentTable);
                }
                currentTable.Columns.Add(new ColumnInfo { Name = colName, Type = dataType });
            }

            Console.WriteLine(JsonSerializer.Serialize(schema));
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine($"Error: {ex.Message}");
            Environment.Exit(1);
        }
    }
}

class TableInfo
{
    public string Name { get; set; }
    public List<ColumnInfo> Columns { get; set; } = new();
}

class ColumnInfo
{
    public string Name { get; set; }
    public string Type { get; set; }
}
