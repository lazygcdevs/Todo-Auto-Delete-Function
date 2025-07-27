# Git Setup Instructions for Todo Auto-Delete Function

## Option 1: Clone Existing Repository and Create Develop Branch

If you have an existing repository, follow these steps:

```powershell
# Navigate to parent directory
cd "c:\Code"

# Clone the repository (replace with your actual repo URL)
git clone https://github.com/yourusername/todo-auto-delete-function.git

# Navigate into the cloned repository
cd "todo-auto-delete-function"

# Create and switch to develop branch
git checkout -b develop

# Copy your code files from the current directory
Copy-Item "c:\Code\Todo application - Auto-Delete-function\main.go" -Destination "."
Copy-Item "c:\Code\Todo application - Auto-Delete-function\main_test.go" -Destination "."
Copy-Item "c:\Code\Todo application - Auto-Delete-function\go.mod" -Destination "."
Copy-Item "c:\Code\Todo application - Auto-Delete-function\go.sum" -Destination "." -ErrorAction SilentlyContinue
Copy-Item "c:\Code\Todo application - Auto-Delete-function\deploy.ps1" -Destination "."
Copy-Item "c:\Code\Todo application - Auto-Delete-function\README.md" -Destination "."
Copy-Item "c:\Code\Todo application - Auto-Delete-function\demo.go" -Destination "."

# Add all files to staging
git add .

# Commit the changes
git commit -m "Add Todo Auto-Delete Function - Pure Go implementation

- Complete standalone Go application for DynamoDB auto-deletion
- Runs every 6 hours using built-in cron scheduler
- Environment variable configuration (no JSON files)
- Comprehensive unit tests
- Azure Container Instance deployment script
- Ready for review"

# Push the develop branch to remote
git push -u origin develop
```

## Option 2: Initialize New Repository in Current Directory

If you want to create a new repository:

```powershell
# Initialize Git repository
git init

# Create develop branch
git checkout -b develop

# Add all files
git add .

# Initial commit
git commit -m "Initial commit: Todo Auto-Delete Function - Pure Go implementation"

# Add remote repository (replace with your actual repo URL)
git remote add origin https://github.com/yourusername/todo-auto-delete-function.git

# Push develop branch
git push -u origin develop
```

## Current Project Files Ready for Review:

✅ **main.go** - Complete Todo Auto-Delete Function application
✅ **main_test.go** - Comprehensive unit tests (8 test functions)
✅ **go.mod** - Go module dependencies (Pure Go - No JSON!)
✅ **deploy.ps1** - Azure Container Instance deployment script
✅ **README.md** - Complete documentation
✅ **demo.go** - Demonstration and explanation script

## Key Features for Review:

- 🚫 **No JSON Configuration Files** - Pure Go implementation as requested
- ⏰ **Built-in Cron Scheduler** - Runs every 6 hours automatically
- 🗑️ **DynamoDB Integration** - Deletes records older than 6 hours
- 🧪 **Unit Tests** - Comprehensive test coverage
- 🚀 **Azure Deployment Ready** - Container Instance deployment script
- 📝 **Environment Configuration** - All settings via environment variables
- 🛡️ **Error Handling** - Graceful shutdown and retry logic

## Next Steps:

1. Choose Option 1 or Option 2 above based on your setup
2. Execute the PowerShell commands
3. Your code will be in the `develop` branch ready for review
4. Share the repository URL with your team for code review
