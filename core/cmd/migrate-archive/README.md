# Archive Migration Utility

This utility helps migrate archived prompts from the old archive system (prompts with "archive" tag in the prompts folder) to the new archive system (prompts physically moved to the `archive/` folder).

## When to Use

- **After upgrading** to a version with the new archive folder system
- **Data recovery** scenarios where archived prompts are in the wrong location
- **Debugging** archive-related issues
- **Clean up** when prompts with archive tags are still in the prompts folder

## How to Run

```bash
# From the project root
go run cmd/migrate-archive/main.go
```

## What It Does

1. **Scans** the prompts folder for any prompts with the "archive" tag
2. **Lists** found prompts and asks for confirmation
3. **Moves** each archived prompt to the `archive/` folder with versioned filename
4. **Removes** the old file from the prompts folder
5. **Reports** success/failure for each migration

## Safe Operation

- **Non-destructive**: Creates new file before removing old one
- **Confirmation required**: Always asks before making changes
- **Error handling**: Continues processing even if individual files fail
- **Git-friendly**: Changes can be committed to git after migration

## Example Output

```
Found 0 archived prompts already in archive folder
Found 1 archived prompts that need migration:
  - Transcriber (v1.0.0) at prompts/transcriber-v1.0.0.md

Proceed with migration? (y/N): y
Moving prompts/transcriber-v1.0.0.md to archive/transcriber-v1.0.0.md
Migration completed! Successfully moved 1 prompts to archive folder
```