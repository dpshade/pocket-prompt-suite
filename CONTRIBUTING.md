# Contributing to Pocket Prompt Suite

Thank you for your interest in contributing! This guide covers contributing to both the core application and Raycast extension.

## 🏗️ Mono-repo Structure

This project uses a mono-repo structure with workspaces:

```
pocket-prompt-suite/
├── core/                    # Go application
├── raycast-extension/       # TypeScript Raycast extension
├── package.json            # Workspace configuration
└── .github/workflows/      # CI/CD pipelines
```

## 🚀 Getting Started

### Prerequisites
- **Go 1.21+** for core development
- **Bun 1.0+** for extension development
- **Git** for version control

### Initial Setup
```bash
# Clone the repository
git clone <repository-url>
cd pocket-prompt-suite

# Install workspace dependencies
bun install

# Build everything
bun run build
```

## 🔧 Development Workflow

### Core Application (Go)
```bash
# Run tests
bun run test:core

# Start development server
bun run server

# Start TUI interface
bun run tui

# Build binary
bun run build:core
```

### Raycast Extension (TypeScript)
```bash
# Development mode
bun run dev

# Run linting
bun run lint:extension

# Build for production
bun run build:extension

# Submit to store
bun run publish:extension
```

### Full Suite Commands
```bash
# Test everything
bun run test

# Build everything
bun run build

# Lint everything
bun run lint
```

## 🎯 Making Changes

### 1. **Create a Branch**
```bash
git checkout -b feature/your-feature-name
```

### 2. **Make Your Changes**
- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. **Test Locally**
```bash
# Test the specific component you changed
bun run test:core      # For Go changes
bun run test:extension # For TypeScript changes

# Or test everything
bun run test
```

### 4. **Commit & Push**
```bash
git add .
git commit -m "feat: add your feature description"
git push origin feature/your-feature-name
```

### 5. **Create Pull Request**
- Open a PR against the `main` branch
- Include a clear description of changes
- Link any related issues

## 📋 Code Guidelines

### Go Code (Core)
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for public APIs
- Include unit tests for new functionality

### TypeScript Code (Extension)
- Follow the existing ESLint configuration
- Use TypeScript strict mode
- Prefer functional components and hooks
- Follow Raycast extension best practices

### Commit Messages
Use conventional commits format:
- `feat:` for new features
- `fix:` for bug fixes  
- `docs:` for documentation changes
- `test:` for adding tests
- `refactor:` for code improvements
- `chore:` for maintenance tasks

## 🔄 API Changes

When making changes that affect both core and extension:

### 1. **Update Core API**
- Modify Go structs and handlers
- Update API documentation
- Add/update tests

### 2. **Update Extension Types**
- Update TypeScript interfaces in `raycast-extension/src/types/`
- Modify API client calls in `utils/api.ts`
- Test integration with development server

### 3. **Coordinate Testing**
- Start development server: `bun run server`
- Test extension with: `bun run dev`
- Verify end-to-end functionality

## 🧪 Testing

### Unit Tests
```bash
# Core unit tests
cd core && go test ./...

# Extension tests (when available)
cd raycast-extension && bun test
```

### Integration Tests
```bash
# Start server
bun run server

# In another terminal, test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/pocket-prompt/list?format=json
```

### Manual Testing
1. Start the server: `bun run server`
2. Load extension: `bun run dev`
3. Test search functionality in Raycast
4. Verify API responses match expected formats

## 🔍 CI/CD

The project includes automated CI/CD:

### **Continuous Integration**
- Tests run on every PR
- Both Go and TypeScript are tested
- Integration tests verify API compatibility
- All builds must pass before merging

### **Release Process**
- Tags trigger automated releases
- Binaries built for multiple platforms
- Raycast extension can be published separately

## 📝 Documentation

When contributing:

### **Code Documentation**
- Add GoDoc comments for Go functions
- Use JSDoc for TypeScript functions
- Update README files for significant changes

### **API Documentation**
- Update endpoint documentation in main README
- Include example requests/responses
- Document any breaking changes

## 🐛 Reporting Issues

### **Bug Reports**
Include:
- Steps to reproduce
- Expected vs actual behavior
- System information (OS, Go version, etc.)
- Relevant logs or error messages

### **Feature Requests**
Include:
- Clear description of the feature
- Use cases and benefits
- Proposed implementation approach

## 💬 Questions?

- **General Questions**: Use GitHub Discussions
- **Bug Reports**: Create GitHub Issues
- **Feature Requests**: Create GitHub Issues with `enhancement` label

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.