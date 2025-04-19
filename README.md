# Archiver

A CLI tool that ingests an external drive, transcodes videos, summarizes documents, uploads to Backblaze B2, and provides a searchable index.

## Features

- Scans and builds a manifest of external drives
- Transcodes videos using Apple VideoToolbox acceleration
- Converts images from HEIC/AVIF to optimized formats
- Extracts and summarizes document content via LLM with cost caps
- Uploads files to Backblaze B2 storage
- Creates local stubs and a Bleve search index

## Requirements

- Go 1.22 or higher
- macOS with Apple Silicon (for VideoToolbox)
- ffmpeg with VideoToolbox support
- Backblaze B2 account and credentials
- Optional: API keys for OpenAI, Anthropic, or Groq

## Installation

```bash
# Clone the repository
git clone https://github.com/jth/archiver.git
cd archiver

# Build the project
make build
```

## Usage

```bash
# Basic usage
./archiver --source /Volumes/ExtDrive --b2-key-id $B2_KEY_ID --b2-app-key $B2_APP_KEY --bucket my-archive

# With all options
./archiver \
  --source /Volumes/ExtDrive \
  --b2-key-id $B2_KEY_ID \
  --b2-app-key $B2_APP_KEY \
  --bucket my-archive \
  --summarise default \
  --stub-mode webloc \
  --cost-cap $COST_CAP_USD
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `B2_KEY_ID` | Backblaze B2 Key ID |
| `B2_APP_KEY` | Backblaze B2 Application Key |
| `GROQ_API_KEY` | API key for Groq (Llama 3 8B) |
| `ANTHROPIC_KEY` | API key for Anthropic Claude |
| `OPENAI_API_KEY` | API key for OpenAI (optional) |
| `COST_CAP_USD` | Maximum LLM spend (default: 5 USD) |

## License

MIT 