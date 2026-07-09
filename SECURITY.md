# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |

## Security Considerations

### Input Validation
- All URLs are validated before benchmarking
- Duration strings are parsed with strict validation
- File paths are sanitized; path traversal is prevented
- YAML files are parsed with safe unmarshaling

### Network Safety
- Configurable request timeouts prevent hanging connections
- Support for HTTP/2 with proper TLS configuration
- Redirect limits (max 10) prevent redirect loops
- Graceful shutdown on SIGINT/SIGTERM

### Secrets and Authentication
- API keys and tokens should use environment variables, not hardcoded values
- Use the `--header` flag for sensitive headers rather than YAML files
- Never commit `.env` files — a `.env.example` is provided

### Reporting
- No sensitive data is collected or transmitted
- All data stays local — no telemetry or external calls

## Reporting a Vulnerability

If you discover a security vulnerability, please open an issue with details.
Do not include sensitive information in the issue body — contact us directly if needed.

## Responsible Disclosure

We follow responsible disclosure practices. Please give us reasonable time to
address issues before public disclosure.