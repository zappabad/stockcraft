# Project Structure

```
/internal
  /orderbook
    /core        # deterministic matching core (your current internal/engine)
    /service     # goroutine owner + command RPC + event fanout (your current internal/api)
    /view        # read model built from core events (your current internal/view)

  /market
    /service     # owns many orderbook services; routes submits; consolidates events
    /view        # market-level read model (best bid/ask, last trade, tape per ticker, etc.)
    types.go     # Ticker, TickerID, helpers

  /news
    /core        # deterministic news generation rules (optional; only if you have “logic”)
    /service     # scheduler/generator + public API + event stream
    /view        # last N news items, filters, etc.
    types.go

  /trader
    /strategy    # strategies (pure decision logic over snapshots/news)
    /runner      # goroutine(s) that feed strategies with snapshots/news; emits requests/orders
    types.go     # trader events, requests, IDs, etc.

  /broker
    /service     # “player” orchestration: subscribes to trader requests/news; can approve/act
    /view        # optional (player state, inventory, pnl)
    types.go

  /game         # wiring/root: constructs market, news, traders, broker; connects channels

/cmd
  /tui          # the UI app; depends on internal/game (or internal/market+news+broker views)
```