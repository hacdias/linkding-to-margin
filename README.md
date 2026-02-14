# Linkding to Margin.at

> Import your [Linkding](https://linkding.link/) bookmarks to [Margin.at](https://margin.at/), an AT Protocol bookmark service.

This tool fetches all bookmarks from your Linkding instance and imports them as bookmarks or annotations (if notes are present) to your AT Protocol account.

## Install

Ensure you have Go 1.25 or later installed, then:

```bash
go install go.hacdias.com/linkding-to-margin@latest
```

Or clone the repository and run:

```bash
go build
```

## Usage

### Configuration

Copy [`.env.example`](.env.example) to `.env` and fill the environment variables as needed.

**Required variables:**

- `ATPROTO_HOST`: The AT Protocol PDS host (e.g., `https://bsky.social`)
- `ATPROTO_IDENTIFIER`: Your AT Protocol handle (e.g., `username.bsky.social`)
- `ATPROTO_PASSWORD`: An (app) password for your AT Protocol account
- `LINKDING_ENDPOINT`: The base URL of your Linkding instance
- `LINKDING_API_KEY`: Your Linkding API key (found in Linkding settings)

**Optional variables:**

- `DRY_RUN`: Set to `true` to preview records without creating them
- `IGNORE_ARCHIVED`: Set to `true` to skip archived bookmarks
- `PROCESSED_IDS_FILE`: Path to the CSV file tracking imported bookmarks (default: `processed-bookmarks.csv`)

### Running the import

```bash
linkding-to-margin
```

The tool will:

1. Load environment variables from `.env`
2. Fetch all bookmarks from your Linkding instance
3. Import them to your AT Protocol account as bookmarks or annotations
4. Display the count of imported bookmarks

**Note:** Bookmarks with notes will be imported as annotations. Bookmarks without notes will be imported as regular bookmarks.

### Error Recovery

If the import fails partway through, simply run the command again. The tool automatically tracks successfully imported bookmarks in `processed-bookmarks.csv` and will resume from where it left off. You can manually edit or delete the CSV file if needed to restart the import.

## License

MIT © Henrique Dias
