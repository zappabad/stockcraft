# Stockcraft Documentation

This directory contains documentation for the Stockcraft trading simulation system.

## Overview

Stockcraft is a modular trading simulation with the following subsystems:

| Document | Description |
|----------|-------------|
| [Architecture Overview](architecture.md) | High-level system design and data flow |
| [Orderbook System](orderbook.md) | Core matching engine, views, and service |
| [Market System](market.md) | Multi-ticker market aggregation |
| [News System](news.md) | News publishing and delivery |
| [Trader System](trader.md) | Strategy interface and runner |
| [Broker System](broker.md) | Player orchestration (minimal) |
| [Game Wiring](game.md) | System composition and lifecycle |
| [TUI](tui.md) | Terminal user interface |

## Quick Start

```bash
# Build and run the TUI
go build ./cmd/tui/...
./tui

# Or run directly
go run ./cmd/tui/...
```

## Package Structure

```
/internal
  /orderbook          # Order matching subsystem
    /core             # Deterministic matching engine (no I/O)
    /view             # Read-only book state projection
    /service          # Goroutine owner, thread-safe API

  /market             # Multi-ticker aggregation
    /view             # Market-wide state (best prices, last trades)
    /service          # Manages multiple orderbooks

  /news               # News event system
    /view             # Ring buffer of news items
    /service          # Publishing and subscription

  /trader             # Automated trading agents
    /strategy         # Strategy interface + examples
    /runner           # Tick-based execution loop

  /broker             # Player interaction (minimal stub)
    /view             # Request tracking
    /service          # Event attachment

  /game               # Top-level composition
    config.go         # Game configuration
    game.go           # Lifecycle management

/cmd
  /tui                # Terminal UI entry point
```

## Design Principles

1. **Separation of Concerns**: Core logic has no I/O; services own goroutines
2. **Event-Driven**: All state changes emit events for views and subscribers
3. **Thread Safety**: Views use `sync.RWMutex`; services use channels
4. **Value Semantics**: Public APIs return copies, never internal references
5. **Graceful Shutdown**: All goroutines terminate on `Close()` via `WaitGroup`
