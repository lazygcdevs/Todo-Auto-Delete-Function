# Todo Auto-Delete Function (Pure Go)

This is a **pure Go application** that automatically deletes records from a DynamoDB database every 6 hours. It runs as a standalone service with no JSON configuration files.

## Features

- **Pure Go**: No JSON configuration files, everything in Go code
- **Cron Scheduling**: Built-in cron scheduler runs every 6 hours  
- **DynamoDB Integration**: Connects to AWS DynamoDB to delete old records
- **Batch Processing**: Deletes records in batches of 25 (DynamoDB limit)
- **Environment Configuration**: All configuration via environment variables
- **Structured Logging**: Comprehensive logging with configurable levels
- **Graceful Shutdown**: Handles SIGINT/SIGTERM for clean shutdown
- **Retry Logic**: Built-in retry mechanism for failed operations
- **High Performance**: Optimized for efficiency and low resource usage

## Prerequisites

- Go 1.21 or later
- AWS credentials with DynamoDB access
- Docker (for containerized deployment)

## Project Structure

```
.
├── main.go           # Complete application in pure Go
├── main_test.go      # Comprehensive unit tests
├── go.mod           # Go module dependencies (no JSON!)
├── deploy.ps1       # Azure Container Instance deployment
├── README.md        # This file
└── .gitignore       # Git ignore file
```

## Configuration

### Environment Variables

Set these in your Azure Function App settings or `local.settings.json`:

```json
{
  "AWS_ACCESS_KEY_ID": "your-aws-access-key",
  "AWS_SECRET_ACCESS_KEY": "your-aws-secret-key",
  "AWS_REGION": "us-east-1",
  "DYNAMODB_TABLE_NAME": "TodoItems",
  "DELETE_ALL_RECORDS": "false"
}
```

### DynamoDB Table Requirements

Your DynamoDB table should have:
- A primary key field named `id` (string)
- A timestamp field named `createdAt` (string, ISO 8601 format)

Example item structure:
```json
{
  "id": "uuid-here",
  "text": "Todo item text",
  "completed": false,
  "createdAt": "2025-07-27T10:00:00Z"
}
```

## Local Development

### Install Dependencies

```bash
go mod tidy
```

### Run Tests

```bash
go test -v
```

### Run Locally

```bash
# Test the function logic
go run main.go

# Or start with Azure Functions Core Tools
func start
```

## Schedule

The function runs every 6 hours using the CRON expression: `0 0 */6 * * *`

This translates to:
- 12:00 AM (midnight)
- 6:00 AM  
- 12:00 PM (noon)
- 6:00 PM

## Deployment

### Build for Azure

```bash
# Build for Linux (Azure Functions runtime)
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o main .
```

### Deploy to Azure

```powershell
# Use the deployment script
.\deploy.ps1 -ResourceGroupName "myResourceGroup" -FunctionAppName "myTodoAutoDelete" -StorageAccountName "mystorageaccount"
```

### Manual Deployment Steps

1. **Login to Azure:**
   ```bash
   az login
   ```

2. **Create Function App:**
   ```bash
   az functionapp create --resource-group myResourceGroup --consumption-plan-location eastus --runtime custom --functions-version 4 --name myTodoAutoDeleteApp --storage-account mystorageaccount
   ```

3. **Deploy:**
   ```bash
   func azure functionapp publish myTodoAutoDeleteApp
   ```

4. **Set Environment Variables:**
   ```bash
   az functionapp config appsettings set --name myTodoAutoDeleteApp --resource-group myResourceGroup --settings AWS_ACCESS_KEY_ID=your-key AWS_SECRET_ACCESS_KEY=your-secret AWS_REGION=us-east-1 DYNAMODB_TABLE_NAME=TodoItems
   ```

## Customization

### Delete All Records

To delete ALL records instead of just old ones, set the environment variable:

```
DELETE_ALL_RECORDS=true
```

**⚠️ Warning: This will delete ALL records in the table. Use with extreme caution!**

### Change Schedule

Modify the `schedule` in `TodoAutoDelete/function.json`:

```json
{
  "schedule": "0 0 */6 * * *"  // Every 6 hours
}
```

Common CRON patterns:
- Every hour: `0 0 * * * *`
- Every 12 hours: `0 0 */12 * * *`
- Daily at midnight: `0 0 0 * * *`

### Modify Time Filter

Change the time filter in `main.go`:

```go
// Delete records older than 6 hours
deletedCount, err = dbService.DeleteOldRecords(ctx, 6*time.Hour)

// Examples:
// 24 hours: dbService.DeleteOldRecords(ctx, 24*time.Hour)
// 30 days: dbService.DeleteOldRecords(ctx, 30*24*time.Hour)
```

### Custom Table Schema

If your DynamoDB table has a different schema, update the `TodoItem` struct and key mapping in `deleteBatch()`:

```go
type TodoItem struct {
    ID        string `json:"id" dynamodbav:"id"`
    // Add your custom fields here
}

// Update the key mapping in deleteBatch()
Key: map[string]*dynamodb.AttributeValue{
    "your-primary-key": {
        S: aws.String(item.YourPrimaryKey),
    },
    // Add sort key if needed
},
```

## Monitoring

Monitor the function through:
- Azure Portal → Function Apps → Your App → Functions → TodoAutoDelete → Monitor
- Application Insights (if configured)
- Azure Monitor Logs

Example log output:
```
TodoAutoDelete function started at: 2025-07-27T12:00:00Z
Deleting records older than: 2025-07-27T06:00:00Z
Found 15 items to delete
Deleted item: abc-123 - Old todo item
Successfully deleted 15 items
Function completed: {"deletedCount":15,"duration":"2.5s","message":"TodoAutoDelete function completed successfully"}
```

## Performance

- **Batch Processing**: Processes deletions in batches of 25 items (DynamoDB limit)
- **Pagination**: Handles large datasets with automatic pagination
- **Error Handling**: Continues processing even if individual batches fail
- **Memory Efficient**: Streams data instead of loading everything into memory

## Security

- Store AWS credentials securely in Azure Key Vault
- Use IAM roles with minimal required permissions
- Regularly rotate access keys
- Monitor CloudWatch/CloudTrail for DynamoDB access

### Required DynamoDB Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:Scan",
        "dynamodb:BatchWriteItem"
      ],
      "Resource": "arn:aws:dynamodb:region:account:table/TodoItems"
    }
  ]
}
```

## Troubleshooting

1. **Function not triggering**: Check the timer schedule format
2. **AWS connection issues**: Verify credentials and region settings
3. **DynamoDB errors**: Ensure proper permissions and table schema
4. **Build failures**: Ensure Go 1.21+ is installed and GOOS/GOARCH are set correctly
5. **Deployment issues**: Check Azure Functions Core Tools version

## Testing

Run the test suite:

```bash
# Run all tests
go test -v

# Run with coverage
go test -cover

# Run benchmarks
go test -bench=.
```

## License

This project is licensed under the MIT License.
