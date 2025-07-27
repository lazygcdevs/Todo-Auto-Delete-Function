package main

import (
	"os"
	"testing"
	"time"
)

// TestLoadConfiguration tests configuration loading
func TestLoadConfiguration(t *testing.T) {
	// Set test environment variables
	os.Setenv("AWS_REGION", "us-west-2")
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("DYNAMODB_TABLE_NAME", "TestTodoItems")
	os.Setenv("DELETE_ALL_RECORDS", "true")
	os.Setenv("DELETE_OLDER_THAN_HOURS", "12")
	os.Setenv("CRON_SCHEDULE", "0 */12 * * *")
	os.Setenv("LOG_LEVEL", "debug")

	config, err := LoadConfiguration()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify configuration values
	if config.AWSRegion != "us-west-2" {
		t.Errorf("Expected AWS region 'us-west-2', got '%s'", config.AWSRegion)
	}

	if config.DynamoDBTableName != "TestTodoItems" {
		t.Errorf("Expected table name 'TestTodoItems', got '%s'", config.DynamoDBTableName)
	}

	if !config.DeleteAllRecords {
		t.Error("Expected DeleteAllRecords to be true")
	}

	if config.DeleteOlderThan != 12*time.Hour {
		t.Errorf("Expected DeleteOlderThan to be 12 hours, got %v", config.DeleteOlderThan)
	}

	if config.CronSchedule != "0 */12 * * *" {
		t.Errorf("Expected cron schedule '0 */12 * * *', got '%s'", config.CronSchedule)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.LogLevel)
	}

	t.Log("✅ Configuration loaded successfully")
}

// TestLoadConfigurationDefaults tests configuration with default values
func TestLoadConfigurationDefaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("DYNAMODB_TABLE_NAME")
	os.Unsetenv("DELETE_ALL_RECORDS")
	os.Unsetenv("DELETE_OLDER_THAN_HOURS")
	os.Unsetenv("CRON_SCHEDULE")
	os.Unsetenv("LOG_LEVEL")

	// Set required variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	config, err := LoadConfiguration()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify default values
	if config.AWSRegion != "us-east-1" {
		t.Errorf("Expected default AWS region 'us-east-1', got '%s'", config.AWSRegion)
	}

	if config.DynamoDBTableName != "TodoItems" {
		t.Errorf("Expected default table name 'TodoItems', got '%s'", config.DynamoDBTableName)
	}

	if config.DeleteAllRecords {
		t.Error("Expected DeleteAllRecords to be false by default")
	}

	if config.DeleteOlderThan != 6*time.Hour {
		t.Errorf("Expected default DeleteOlderThan to be 6 hours, got %v", config.DeleteOlderThan)
	}

	if config.CronSchedule != "0 */6 * * *" {
		t.Errorf("Expected default cron schedule '0 */6 * * *', got '%s'", config.CronSchedule)
	}

	t.Log("✅ Default configuration values are correct")
}

// TestLoadConfigurationMissingRequired tests error handling for missing required config
func TestLoadConfigurationMissingRequired(t *testing.T) {
	// Clear required environment variables
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	_, err := LoadConfiguration()
	if err == nil {
		t.Error("Expected error for missing AWS_ACCESS_KEY_ID")
	}

	// Set one but not the other
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	_, err = LoadConfiguration()
	if err == nil {
		t.Error("Expected error for missing AWS_SECRET_ACCESS_KEY")
	}

	t.Log("✅ Required configuration validation works correctly")
}

// TestNewDynamoDBService tests DynamoDB service creation
func TestNewDynamoDBService(t *testing.T) {
	config := &Configuration{
		AWSRegion:         "us-east-1",
		AWSAccessKey:      "test-key",
		AWSSecretKey:      "test-secret",
		DynamoDBTableName: "TestTodoItems",
	}

	service, err := NewDynamoDBService(config)
	if err != nil {
		t.Fatalf("Failed to create DynamoDB service: %v", err)
	}

	if service.config.DynamoDBTableName != "TestTodoItems" {
		t.Errorf("Expected table name 'TestTodoItems', got '%s'", service.config.DynamoDBTableName)
	}

	t.Log("✅ DynamoDB service created successfully")
}

// TestLogger tests the logger functionality
func TestLogger(t *testing.T) {
	// Test info level logger
	logger := NewLogger("info")
	if logger.level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", logger.level)
	}

	// Test debug level logger
	debugLogger := NewLogger("debug")
	if debugLogger.level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", debugLogger.level)
	}

	t.Log("✅ Logger functionality works correctly")
}

// TestAutoDeleteJob tests the auto delete job structure
func TestAutoDeleteJob(t *testing.T) {
	config := &Configuration{
		AWSRegion:         "us-east-1",
		AWSAccessKey:      "test-key",
		AWSSecretKey:      "test-secret",
		DynamoDBTableName: "TestTodoItems",
		DeleteAllRecords:  false,
		DeleteOlderThan:   6 * time.Hour,
	}

	dbService, err := NewDynamoDBService(config)
	if err != nil {
		t.Fatalf("Failed to create DynamoDB service: %v", err)
	}

	logger := NewLogger("info")
	job := NewAutoDeleteJob(dbService, logger)

	if job.dbService == nil {
		t.Error("Expected job to have dbService")
	}

	if job.logger == nil {
		t.Error("Expected job to have logger")
	}

	t.Log("✅ AutoDeleteJob structure is correct")
}

// TestTimerSchedule tests the timer schedule logic
func TestTimerSchedule(t *testing.T) {
	// Test 6-hour duration calculation
	sixHours := 6 * time.Hour
	expectedMinutes := 360 // 6 * 60

	if int(sixHours.Minutes()) != expectedMinutes {
		t.Errorf("Expected %d minutes, got %d", expectedMinutes, int(sixHours.Minutes()))
	}

	// Test timestamp calculation
	now := time.Now()
	sixHoursAgo := now.Add(-sixHours)

	if sixHoursAgo.After(now) {
		t.Error("Six hours ago should be before now")
	}

	t.Log("✅ Timer schedule logic is correct")
}

// BenchmarkDeleteBatch benchmarks the batch deletion performance
func BenchmarkDeleteBatch(b *testing.B) {
	// Create mock items
	items := make([]TodoItem, 25) // Maximum batch size
	for i := 0; i < 25; i++ {
		items[i] = TodoItem{
			ID:        "test-id-" + string(rune(i)),
			Text:      "Test item " + string(rune(i)),
			Completed: false,
			CreatedAt: time.Now().Add(-7 * time.Hour).Format(time.RFC3339),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate batch processing logic (without actual DynamoDB calls)
		for _, item := range items {
			_ = item.ID // Simulate processing
		}
	}
}

// TestApplication tests the main application structure
func TestApplication(t *testing.T) {
	// Set required environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("DYNAMODB_TABLE_NAME", "TestTodoItems")

	app, err := NewApplication()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}

	if app.config == nil {
		t.Error("Expected application to have config")
	}

	if app.dbService == nil {
		t.Error("Expected application to have dbService")
	}

	if app.logger == nil {
		t.Error("Expected application to have logger")
	}

	if app.cron == nil {
		t.Error("Expected application to have cron scheduler")
	}

	t.Log("✅ Application structure is correct")
}
