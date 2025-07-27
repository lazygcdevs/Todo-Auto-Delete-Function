package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/robfig/cron/v3"
)

// Configuration holds all application configuration
type Configuration struct {
	AWSRegion         string
	AWSAccessKey      string
	AWSSecretKey      string
	DynamoDBTableName string
	DeleteAllRecords  bool
	DeleteOlderThan   time.Duration
	CronSchedule      string
	LogLevel          string
}

// TodoItem represents a todo item structure in DynamoDB
type TodoItem struct {
	ID        string `dynamodbav:"id"`
	Text      string `dynamodbav:"text"`
	Completed bool   `dynamodbav:"completed"`
	CreatedAt string `dynamodbav:"createdAt"`
}

// DynamoDBService encapsulates DynamoDB operations
type DynamoDBService struct {
	client *dynamodb.DynamoDB
	config *Configuration
}

// Logger provides structured logging
type Logger struct {
	level string
}

// NewLogger creates a new logger instance
func NewLogger(level string) *Logger {
	return &Logger{level: level}
}

// Info logs info level messages
func (l *Logger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

// Error logs error level messages
func (l *Logger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

// Debug logs debug level messages
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level == "debug" {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

// LoadConfiguration loads configuration from environment variables
func LoadConfiguration() (*Configuration, error) {
	config := &Configuration{
		AWSRegion:         getEnvOrDefault("AWS_REGION", "us-east-1"),
		AWSAccessKey:      getEnvOrDefault("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:      getEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""),
		DynamoDBTableName: getEnvOrDefault("DYNAMODB_TABLE_NAME", "TodoItems"),
		CronSchedule:      getEnvOrDefault("CRON_SCHEDULE", "0 */6 * * *"), // Every 6 hours
		LogLevel:          getEnvOrDefault("LOG_LEVEL", "info"),
	}

	// Parse DELETE_ALL_RECORDS
	deleteAllStr := getEnvOrDefault("DELETE_ALL_RECORDS", "false")
	deleteAll, err := strconv.ParseBool(deleteAllStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DELETE_ALL_RECORDS value: %s", deleteAllStr)
	}
	config.DeleteAllRecords = deleteAll

	// Parse DELETE_OLDER_THAN_HOURS
	olderThanStr := getEnvOrDefault("DELETE_OLDER_THAN_HOURS", "6")
	olderThanHours, err := strconv.Atoi(olderThanStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DELETE_OLDER_THAN_HOURS value: %s", olderThanStr)
	}
	config.DeleteOlderThan = time.Duration(olderThanHours) * time.Hour

	// Validate required fields
	if config.AWSAccessKey == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID environment variable is required")
	}
	if config.AWSSecretKey == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY environment variable is required")
	}

	return config, nil
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// NewDynamoDBService creates a new DynamoDB service instance
func NewDynamoDBService(config *Configuration) (*DynamoDBService, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion),
		Credentials: credentials.NewStaticCredentials(
			config.AWSAccessKey,
			config.AWSSecretKey,
			"", // token
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Create DynamoDB client
	client := dynamodb.New(sess)

	return &DynamoDBService{
		client: client,
		config: config,
	}, nil
}

// DeleteOldRecords deletes records older than the specified duration
func (d *DynamoDBService) DeleteOldRecords(ctx context.Context, logger *Logger) (int64, error) {
	logger.Info("Starting deletion of records older than %v", d.config.DeleteOlderThan)

	// Calculate cutoff timestamp
	cutoffTime := time.Now().Add(-d.config.DeleteOlderThan)
	cutoffTimestamp := cutoffTime.Format(time.RFC3339)

	logger.Info("Deleting records older than: %s", cutoffTimestamp)

	// Scan for items to delete
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(d.config.DynamoDBTableName),
		FilterExpression: aws.String("createdAt < :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cutoff": {
				S: aws.String(cutoffTimestamp),
			},
		},
	}

	return d.processAndDeleteItems(ctx, scanInput, logger)
}

// DeleteAllRecords deletes ALL records from the table (use with caution)
func (d *DynamoDBService) DeleteAllRecords(ctx context.Context, logger *Logger) (int64, error) {
	logger.Error("WARNING: Deleting ALL records from table %s", d.config.DynamoDBTableName)

	// Scan all items
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(d.config.DynamoDBTableName),
	}

	return d.processAndDeleteItems(ctx, scanInput, logger)
}

// processAndDeleteItems processes scan results and deletes items in batches
func (d *DynamoDBService) processAndDeleteItems(ctx context.Context, scanInput *dynamodb.ScanInput, logger *Logger) (int64, error) {
	var deletedCount int64
	var items []TodoItem

	// Scan the table for items
	err := d.client.ScanPagesWithContext(ctx, scanInput, func(page *dynamodb.ScanOutput, lastPage bool) bool {
		for _, item := range page.Items {
			var todoItem TodoItem
			err := dynamodbattribute.UnmarshalMap(item, &todoItem)
			if err != nil {
				logger.Error("Error unmarshaling item: %v", err)
				continue
			}
			items = append(items, todoItem)
		}
		return !lastPage
	})

	if err != nil {
		return 0, fmt.Errorf("error scanning table: %v", err)
	}

	logger.Info("Found %d items to delete", len(items))

	if len(items) == 0 {
		logger.Info("No items to delete")
		return 0, nil
	}

	// Delete items in batches of 25 (DynamoDB limit)
	const batchSize = 25
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		deleted, err := d.deleteBatch(ctx, batch, logger)
		if err != nil {
			logger.Error("Error deleting batch: %v", err)
			continue
		}
		deletedCount += deleted

		// Add small delay to avoid throttling
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Successfully deleted %d items", deletedCount)
	return deletedCount, nil
}

// deleteBatch deletes a batch of items
func (d *DynamoDBService) deleteBatch(ctx context.Context, items []TodoItem, logger *Logger) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	// Build batch write request
	writeRequests := make([]*dynamodb.WriteRequest, len(items))
	for i, item := range items {
		writeRequests[i] = &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"id": {
						S: aws.String(item.ID),
					},
				},
			},
		}
	}

	batchInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			d.config.DynamoDBTableName: writeRequests,
		},
	}

	// Execute batch write with retry logic
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		result, err := d.client.BatchWriteItemWithContext(ctx, batchInput)
		if err != nil {
			if retry == maxRetries-1 {
				return 0, fmt.Errorf("batch write failed after %d retries: %v", maxRetries, err)
			}
			logger.Debug("Batch write failed, retrying in %d seconds: %v", retry+1, err)
			time.Sleep(time.Duration(retry+1) * time.Second)
			continue
		}

		// Handle unprocessed items
		if len(result.UnprocessedItems) > 0 {
			logger.Debug("Found %d unprocessed items, retrying", len(result.UnprocessedItems[d.config.DynamoDBTableName]))
			batchInput.RequestItems = result.UnprocessedItems
			time.Sleep(time.Duration(retry+1) * time.Second)
			continue
		}

		break
	}

	// Log deleted items
	for _, item := range items {
		logger.Debug("Deleted item: %s - %s", item.ID, item.Text)
	}

	return int64(len(items)), nil
}

// AutoDeleteJob represents the scheduled job
type AutoDeleteJob struct {
	dbService *DynamoDBService
	logger    *Logger
}

// NewAutoDeleteJob creates a new auto delete job
func NewAutoDeleteJob(dbService *DynamoDBService, logger *Logger) *AutoDeleteJob {
	return &AutoDeleteJob{
		dbService: dbService,
		logger:    logger,
	}
}

// Run executes the auto delete job
func (job *AutoDeleteJob) Run() {
	startTime := time.Now()
	job.logger.Info("TodoAutoDelete job started at: %s", startTime.Format(time.RFC3339))

	ctx := context.Background()
	var deletedCount int64
	var err error

	if job.dbService.config.DeleteAllRecords {
		// Delete ALL records (use with extreme caution)
		deletedCount, err = job.dbService.DeleteAllRecords(ctx, job.logger)
	} else {
		// Delete records older than specified duration (default behavior)
		deletedCount, err = job.dbService.DeleteOldRecords(ctx, job.logger)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	if err != nil {
		job.logger.Error("Auto delete job failed: %v", err)
		return
	}

	job.logger.Info("TodoAutoDelete job completed successfully - Deleted: %d records, Duration: %s", 
		deletedCount, duration.String())
}

// Application represents the main application
type Application struct {
	config    *Configuration
	dbService *DynamoDBService
	logger    *Logger
	cron      *cron.Cron
}

// NewApplication creates a new application instance
func NewApplication() (*Application, error) {
	// Load configuration
	config, err := LoadConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	// Create logger
	logger := NewLogger(config.LogLevel)

	// Create DynamoDB service
	dbService, err := NewDynamoDBService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB service: %v", err)
	}

	// Create cron scheduler
	cronScheduler := cron.New()

	return &Application{
		config:    config,
		dbService: dbService,
		logger:    logger,
		cron:      cronScheduler,
	}, nil
}

// Start starts the application
func (app *Application) Start() error {
	app.logger.Info("Starting Todo Auto-Delete Application")
	app.logger.Info("Configuration:")
	app.logger.Info("  AWS Region: %s", app.config.AWSRegion)
	app.logger.Info("  DynamoDB Table: %s", app.config.DynamoDBTableName)
	app.logger.Info("  Delete All Records: %v", app.config.DeleteAllRecords)
	app.logger.Info("  Delete Older Than: %v", app.config.DeleteOlderThan)
	app.logger.Info("  Cron Schedule: %s", app.config.CronSchedule)

	// Create auto delete job
	job := NewAutoDeleteJob(app.dbService, app.logger)

	// Schedule the job
	_, err := app.cron.AddFunc(app.config.CronSchedule, job.Run)
	if err != nil {
		return fmt.Errorf("failed to schedule job: %v", err)
	}

	// Start the cron scheduler
	app.cron.Start()
	app.logger.Info("Cron scheduler started successfully")

	// Run once immediately for testing (optional)
	runOnStartup := getEnvOrDefault("RUN_ON_STARTUP", "false")
	if runOnStartup == "true" {
		app.logger.Info("Running job immediately on startup")
		go job.Run()
	}

	return nil
}

// Stop stops the application
func (app *Application) Stop() {
	app.logger.Info("Stopping Todo Auto-Delete Application")
	
	// Stop the cron scheduler
	ctx := app.cron.Stop()
	
	// Wait for running jobs to complete
	select {
	case <-ctx.Done():
		app.logger.Info("All scheduled jobs completed")
	case <-time.After(30 * time.Second):
		app.logger.Info("Timeout waiting for jobs to complete")
	}
	
	app.logger.Info("Application stopped successfully")
}

// main function
func main() {
	// Create application
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	app.logger.Info("Application is running. Press Ctrl+C to stop.")
	<-sigChan

	// Graceful shutdown
	app.Stop()
}
