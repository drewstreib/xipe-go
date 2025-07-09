#!/bin/bash
# Install Git hooks for the xipe-go project

echo "Installing Git hooks..."

# Copy pre-commit hook
cp -f .git/hooks/pre-commit .git/hooks/pre-commit.bak 2>/dev/null || true

cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Auto-format Go code before commit

echo "Running pre-commit formatting..."

# Format all Go files
gofmt -w .
goimports -w .

# Add formatted files back to staging
git add .

# Check if there are any formatting issues left
if ! gofmt -l . | grep -q .; then
    echo "✅ Code formatting is correct"
else
    echo "❌ Code formatting issues found:"
    gofmt -l .
    echo "Files have been auto-formatted and staged."
fi

# Run tests
echo "Running tests..."
if ! go test ./... -v; then
    echo "❌ Tests failed"
    exit 1
fi

echo "✅ Pre-commit checks passed"
EOF

chmod +x .git/hooks/pre-commit

echo "✅ Git hooks installed successfully"
echo "Now gofmt and goimports will run automatically before every commit"