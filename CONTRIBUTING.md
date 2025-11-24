# Contributing to provider-garage

Thank you for your interest in contributing to provider-garage! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please be respectful and constructive in all interactions.

## Getting Started

### Prerequisites

- Go 1.21 or later
- kubectl
- Access to a Kubernetes cluster (for testing)
- Access to a Garage instance (for integration testing)

### Setting Up Development Environment

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/provider-garage.git
   cd provider-garage
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/kikokikok/provider-garage.git
   ```

4. Install dependencies:
   ```bash
   go mod download
   ```

5. Build the provider:
   ```bash
   make build
   ```

## Development Workflow

### Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following our coding standards

3. Test your changes:
   ```bash
   # Run unit tests
   make test
   
   # Build to verify compilation
   make build
   ```

4. Commit your changes:
   ```bash
   git add .
   git commit -m "Description of your changes"
   ```

### Coding Standards

#### Go Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

Example:
```go
// CreateBucket creates a new Garage bucket with the specified configuration.
// It returns an error if the bucket already exists or if the operation fails.
func CreateBucket(ctx context.Context, name string) error {
    // Implementation
}
```

#### Project Structure

```
provider-garage/
├── apis/                  # API type definitions
│   ├── v1beta1/          # Core provider APIs
│   ├── bucket/v1alpha1/  # Bucket resource APIs
│   └── key/v1alpha1/     # Key resource APIs
├── cmd/                   # Main applications
│   └── provider/         # Provider entry point
├── config/               # Resource configurations
│   ├── bucket/          # Bucket config
│   ├── key/             # Key config
│   └── provider.go      # Provider setup
├── internal/             # Private packages
│   ├── clients/         # API clients
│   ├── controller/      # Controllers
│   └── features/        # Feature flags
└── examples/            # Usage examples
```

### Testing

#### Unit Tests

Write unit tests for all new functionality:

```go
func TestCreateBucket(t *testing.T) {
    // Test implementation
}
```

Run tests:
```bash
make test
```

#### Integration Tests

For integration testing:

1. Set up a test Garage instance
2. Configure test credentials
3. Run integration tests (when implemented)

### Documentation

Update documentation when you:
- Add new features
- Change existing behavior
- Fix bugs that affect usage

Documentation locations:
- `README.md`: Main documentation
- `GETTING_STARTED.md`: User guide
- `ARCHITECTURE.md`: Technical details
- Code comments: Inline documentation

## Pull Request Process

### Before Submitting

1. Ensure tests pass:
   ```bash
   make test
   ```

2. Verify build succeeds:
   ```bash
   make build
   ```

3. Update documentation if needed

4. Sync with upstream:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

### Creating a Pull Request

1. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Go to GitHub and create a Pull Request

3. Fill out the PR template:
   - Description of changes
   - Related issues
   - Testing performed
   - Screenshots (if UI changes)

### PR Title Format

Use conventional commits format:

- `feat: Add new feature`
- `fix: Fix bug in X`
- `docs: Update documentation`
- `test: Add tests for Y`
- `refactor: Refactor Z`
- `chore: Update dependencies`

### Review Process

1. Automated checks must pass:
   - Build verification
   - Unit tests
   - Linting

2. Code review by maintainers:
   - Code quality
   - Design decisions
   - Documentation

3. Address review feedback:
   - Make requested changes
   - Respond to comments
   - Update PR

4. Approval and merge:
   - Requires approval from maintainer
   - Squash and merge preferred

## Types of Contributions

### Bug Reports

Found a bug? Please create an issue with:

- Clear description of the bug
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment details (versions, platform)
- Logs (if applicable)

Example issue template:
```markdown
**Description**
Brief description of the bug

**To Reproduce**
1. Create resource X
2. Configure Y
3. Observe error

**Expected Behavior**
What should happen

**Actual Behavior**
What actually happens

**Environment**
- Provider version: v0.1.0
- Crossplane version: v1.14.0
- Kubernetes version: v1.28.0
- Garage version: v0.9.0

**Logs**
```
paste logs here
```
```

### Feature Requests

Have an idea? Create an issue with:

- Clear description of the feature
- Use case and benefits
- Proposed implementation (optional)
- Examples (if applicable)

### Documentation

Improvements to documentation are always welcome:

- Fix typos
- Clarify instructions
- Add examples
- Update outdated content

### Code Contributions

Areas that need contributions:

1. **New Resources**: Support for additional Garage resources
2. **Testing**: Unit and integration tests
3. **Examples**: More usage examples and compositions
4. **Performance**: Optimization and improvements
5. **Bug Fixes**: Address open issues

## Resource Configuration Guide

### Adding a New Resource

To add support for a new Garage resource:

1. **Add external name configuration** (`config/external_name.go`):
   ```go
   var ExternalNameConfigs = map[string]config.ExternalName{
       "garage_new_resource": config.IdentifierFromProvider,
   }
   ```

2. **Create resource configurator** (`config/newresource/config.go`):
   ```go
   package newresource

   import "github.com/crossplane/upjet/pkg/config"

   func Configure(p *config.Provider) {
       p.AddResourceConfigurator("garage_new_resource", func(r *config.Resource) {
           r.ShortGroup = "newresource"
           r.Kind = "NewResource"
           r.Description = "NewResource is..."
       })
   }
   ```

3. **Register configurator** (`config/provider.go`):
   ```go
   import "github.com/kikokikok/provider-garage/config/newresource"
   
   // In GetProvider():
   for _, configure := range []func(provider *config.Provider){
       bucket.Configure,
       key.Configure,
       newresource.Configure, // Add this
   } {
       configure(pc)
   }
   ```

4. **Generate code**:
   ```bash
   make generate
   ```

5. **Test the resource**:
   - Create example manifest in `examples/newresource/`
   - Test resource creation and deletion
   - Verify status updates

## Release Process

Releases are managed by maintainers:

1. Version bump in relevant files
2. Update CHANGELOG.md
3. Create git tag
4. Build and publish container image
5. Create GitHub release

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue
- **Security**: Email maintainers privately
- **Chat**: Join our community chat (if available)

## Recognition

Contributors will be:
- Listed in CONTRIBUTORS.md
- Credited in release notes
- Acknowledged in the repository

Thank you for contributing to provider-garage!
