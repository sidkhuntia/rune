# Rune 🚀

**Rune** is an AI-powered CLI tool that generates descriptive Git commit messages by analyzing your code changes. It supports multiple AI providers including Novita.ai (Qwen models) and Google Gemini Flash.

## Features

- **AI-Powered**: Generates meaningful commit messages using advanced language models
- **Multi-Provider Support**: Choose between Novita.ai and Google Gemini Flash
- **Configurable**: Interactive setup with personalized preferences
- **Editor Integration**: Edit generated messages before committing
- **GitHub Conventions**: Follows GitHub commit message best practices
- **Smart Staging**: Configurable staging behavior for your workflow
- **Flexible**: Supports dry-run, verbose output, and custom models

## Installation

### Using Homebrew

```bash
brew install sidkhuntia/rune/rune
```

### From Source

```bash
git clone https://github.com/sidkhuntia/rune.git
cd rune
go build -o rune ./cmd/rune
```

### Using Go Install

```bash
go install github.com/sidkhuntia/rune/cmd/rune@latest
```

## Setup

### Initial Configuration

Run the interactive setup:

```bash
rune --setup
```

This will guide you through:

1. **AI Provider Selection**: Choose between Novita.ai or Google Gemini Flash
2. **API Key Configuration**: Set your API key for the chosen provider
3. **Model Selection**: Choose the AI model to use
4. **Staging Preferences**: Configure how Rune handles file staging

### API Keys

#### Novita.ai
1. Sign up at [Novita.ai](https://novita.ai)
2. Navigate to API Keys section
3. Create a new API key
4. Use it during setup or set `NOVITA_API_KEY` environment variable

#### Google Gemini Flash
1. Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Use it during setup or set `GEMINI_API_KEY` environment variable

## Usage

### Basic Usage

Generate and commit a message for staged changes:

```bash
rune
```

### Options

```bash
# Include all changes (not just staged)
rune --all

# Generate without committing (dry run)
rune --dry-run

# Use a specific model
rune --model qwen/qwen2.5-7b-instruct

# Skip editor (auto-commit)
rune --edit=false

# Verbose output
rune --verbose

# Reconfigure settings
rune --setup
```

### Staging Behavior

Rune offers flexible staging options configured during setup:

1. **Staged Only**: Only commit already staged changes
2. **Auto-Stage All**: Automatically stage all changes before committing

When using `--all` flag, Rune will warn you if it needs to stage additional changes.

### Examples

```bash
# Basic workflow
git add .
rune

# Include unstaged changes
rune --all

# Preview generated message
rune --dry-run

# Use different model temporarily
rune --model gemini-2.0-flash

# Verbose output with editing
rune --verbose --edit
```

## Configuration

Configuration is stored in `~/.config/rune/config.json`:

```json
{
  "provider": "novita",
  "api_key": "your-api-key",
  "model": "qwen/qwen2.5-7b-instruct",
  "staged_only": false,
  "auto_stage_all": true
}
```

### Supported Models

#### Novita.ai
- `qwen/qwen2.5-7b-instruct` (default)

#### Google Gemini
- `gemini-2.0-flash` (default)

## Commit Message Format

Rune follows GitHub commit message conventions:

- **Subject line**: ≤50 characters, capitalized, no trailing period
- **Body**: Wrapped at 72 characters, separated by blank line
- **Format**: Clean, professional, and descriptive

Example output:
```
Add user authentication middleware

Implement JWT-based authentication middleware for securing API endpoints.
The middleware validates tokens, handles refresh logic, and provides
proper error responses for unauthorized access attempts.
```

## Troubleshooting

### Common Issues

**"No staged changes found"**
- Stage your changes with `git add` or use `--all` flag

**"API key not found"**
- Run `rune --setup` to configure your API key
- Or set environment variable: `export NOVITA_API_KEY=your-key`

**"Model not found"**
- Check available models for your provider
- Verify API key has access to the specified model

**"Editor issues"**
- Set your preferred editor: `export EDITOR=nano`
- Default editor is `vi` if `EDITOR` is not set

### Debug Mode

Use verbose flag for detailed output:

```bash
rune --verbose --dry-run
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Add tests if applicable
5. Commit using Rune: `rune`
6. Push and create a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0
- Initial release with Novita.ai and Google Gemini support
- Interactive setup and configuration
- Configurable staging behavior
- GitHub commit conventions
- Editor integration and dry-run mode 