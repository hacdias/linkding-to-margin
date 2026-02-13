# Linkding to Margin.at

> Import your [Linkding](https://docs.linkding.app/) bookmarks to [Margin.at](https://margin.at/), an AT Protocol bookmark service.

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

### Step 1: Set up environment variables

Create a `.env` file in your working directory with the following variables:

```bash
# Linkding configuration
LINKDING_ENDPOINT=https://your-linkding-instance.com
LINKDING_API_KEY=your-linkding-api-key

# AT Protocol (e.g. Bluesky) configuration
ATPROTO_HOST=https://bsky.social
ATPROTO_IDENTIFIER=your.bsky.social
ATPROTO_PASSWORD=your-app-password

# Optional configuration
DRY_RUN=true
IGNORE_ARCHIVED=true
```

- **LINKDING_ENDPOINT**: The base URL of your Linkding instance
- **LINKDING_API_KEY**: Your Linkding API key (found in Linkding settings)
- **ATPROTO_HOST**: The AT Protocol PDS host (default is `https://bsky.social`)
- **ATPROTO_IDENTIFIER**: Your AT Protocol handle (e.g., `username.bsky.social`)
- **ATPROTO_PASSWORD**: An [app password](https://bsky.app/settings/app-passwords) for your AT Protocol account
- **DRY_RUN** (optional): Set to `true` to preview what would be imported without making actual changes (useful for testing)
- **IGNORE_ARCHIVED** (optional): Set to `true` to skip importing archived bookmarks from Linkding

### Step 2: Run the tool

```bash
linkding-to-margin
```

The tool will:

1. Fetch all bookmarks from your Linkding instance
2. Create corresponding bookmark or annotation records in AT Protocol
3. Print the number of bookmarks imported

**Note**: For bookmarks with notes, they will be imported as annotations. Bookmarks without notes will be imported as regular bookmarks.

## License

MIT © Henrique Dias
