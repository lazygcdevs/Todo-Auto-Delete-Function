# Azure Container Instance Deployment Script for Todo Auto-Delete Function (Go)
# This deploys the Go application as a container instance in Azure

param(
    [Parameter(Mandatory=$true)]
    [string]$ResourceGroupName,
    
    [Parameter(Mandatory=$true)]
    [string]$ContainerInstanceName,
    
    [Parameter(Mandatory=$true)]
    [string]$AWSAccessKeyId,
    
    [Parameter(Mandatory=$true)]
    [string]$AWSSecretAccessKey,
    
    [string]$Location = "East US",
    [string]$AWSRegion = "us-east-1",
    [string]$DynamoDBTableName = "TodoItems",
    [string]$DeleteAllRecords = "false",
    [string]$DeleteOlderThanHours = "6",
    [string]$CronSchedule = "0 */6 * * *",
    [string]$LogLevel = "info"
)

Write-Host "Starting deployment of Todo Auto-Delete Function (Go) as Container Instance..." -ForegroundColor Green

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "Go version detected: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Go is not installed or not in PATH. Please install Go first." -ForegroundColor Red
    exit 1
}

# Build the Go binary for Linux
Write-Host "Building Go binary for Linux..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -o todo-auto-delete .

if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Go build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Go binary built successfully" -ForegroundColor Green

# Create Dockerfile
Write-Host "Creating Dockerfile..." -ForegroundColor Yellow
@"
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY todo-auto-delete .
CMD ["./todo-auto-delete"]
"@ | Out-File -FilePath "Dockerfile" -Encoding UTF8

# Create Docker image name
$imageName = "todo-auto-delete:latest"

# Build Docker image
Write-Host "Building Docker image..." -ForegroundColor Yellow
docker build -t $imageName .

if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Docker build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Docker image built successfully" -ForegroundColor Green

# Login to Azure (if not already logged in)
Write-Host "Checking Azure login status..." -ForegroundColor Yellow
try {
    az account show | Out-Null
    Write-Host "Already logged in to Azure." -ForegroundColor Green
} catch {
    Write-Host "Please login to Azure..." -ForegroundColor Yellow
    az login
}

# Create resource group if it doesn't exist
Write-Host "Creating resource group: $ResourceGroupName" -ForegroundColor Yellow
az group create --name $ResourceGroupName --location $Location

# Create Azure Container Registry (optional, for production use)
$acrName = $ResourceGroupName + "acr"
Write-Host "Creating Azure Container Registry: $acrName" -ForegroundColor Yellow
az acr create --resource-group $ResourceGroupName --name $acrName --sku Basic --admin-enabled true

# Get ACR login server
$loginServer = az acr show --name $acrName --resource-group $ResourceGroupName --query "loginServer" --output tsv

# Tag and push image to ACR
$fullImageName = "$loginServer/todo-auto-delete:latest"
docker tag $imageName $fullImageName

Write-Host "Logging into Azure Container Registry..." -ForegroundColor Yellow
az acr login --name $acrName

Write-Host "Pushing image to Azure Container Registry..." -ForegroundColor Yellow
docker push $fullImageName

# Create container instance
Write-Host "Creating Azure Container Instance: $ContainerInstanceName" -ForegroundColor Yellow
az container create `
    --resource-group $ResourceGroupName `
    --name $ContainerInstanceName `
    --image $fullImageName `
    --cpu 1 --memory 1 `
    --restart-policy Always `
    --environment-variables `
        AWS_ACCESS_KEY_ID=$AWSAccessKeyId `
        AWS_SECRET_ACCESS_KEY=$AWSSecretAccessKey `
        AWS_REGION=$AWSRegion `
        DYNAMODB_TABLE_NAME=$DynamoDBTableName `
        DELETE_ALL_RECORDS=$DeleteAllRecords `
        DELETE_OLDER_THAN_HOURS=$DeleteOlderThanHours `
        CRON_SCHEDULE=$CronSchedule `
        LOG_LEVEL=$LogLevel `
        RUN_ON_STARTUP=true `
    --registry-login-server $loginServer `
    --registry-username $acrName `
    --registry-password $(az acr credential show --name $acrName --query "passwords[0].value" --output tsv)

Write-Host "Deployment completed!" -ForegroundColor Green
Write-Host "Container Instance Details:" -ForegroundColor Cyan
Write-Host "- Resource Group: $ResourceGroupName" -ForegroundColor White
Write-Host "- Container Name: $ContainerInstanceName" -ForegroundColor White
Write-Host "- Image: $fullImageName" -ForegroundColor White
Write-Host "- Schedule: $CronSchedule" -ForegroundColor White
Write-Host "- DynamoDB Table: $DynamoDBTableName" -ForegroundColor White

Write-Host "`nTo view logs:" -ForegroundColor Green
Write-Host "az container logs --resource-group $ResourceGroupName --name $ContainerInstanceName" -ForegroundColor Yellow

Write-Host "`nTo view container status:" -ForegroundColor Green  
Write-Host "az container show --resource-group $ResourceGroupName --name $ContainerInstanceName --query instanceView.state" -ForegroundColor Yellow

# Clean up build artifacts
Remove-Item "todo-auto-delete" -ErrorAction SilentlyContinue
Remove-Item "Dockerfile" -ErrorAction SilentlyContinue
