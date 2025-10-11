# Data Directory

This directory contains runtime data files for the Pi Controller application.

## Contents

- **Database Files**: Application databases (SQLite) are stored here
- **Runtime Data**: Any persistent data generated during application execution

## Important Notes

- ⚠️ **Do not commit database files** - These files are environment-specific and contain runtime data
- 🗂️ **Directory Structure**: This directory is preserved in version control, but its contents are ignored
- 🔄 **Auto-Generated**: Database and data files are automatically created when the application runs

## Database Files

The application will automatically create:

- `pi-controller.db` - Main application database (SQLite)

These files are automatically ignored by git via `.gitignore` patterns.
