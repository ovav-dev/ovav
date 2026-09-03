# OVAV Test Harnesses — Canonical Source

## Overview

This directory contains test harnesses used to validate OVAV functionality.

## Structure

```
harnesses/
  go/
    validate_harness.go   # Main validation harness
```

## Go Harnesses

### validate_harness.go

Validates OVAV components:
- F0 validators (runtime integrity)
- F1 validators (git governance)
- F2 validators (security gates)

## Usage

```bash
cd go-runtime
go test -run TestHarness ./internal/...
```
