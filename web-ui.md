# [Draft] Simple WebUI for qb-auto

A simple React SPA + Pocketbase SDK web ui for qb-auto to view and update jobs.

## Feature

- View history/pending/failed jobs
- Restart failed job (allow manual input required info and skip directly to rsync)
- Login using Pocketbase superuser account

## Implementation Plan

Golang can embed static files into built binary. I want to make use of this feature so that we can keep single binary file release.

web ui technical stacks:
- MUI
- react router
- Pocketbase SDK