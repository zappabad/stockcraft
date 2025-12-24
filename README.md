# Project Structure

```
/internal
/engine   # deterministic core (no goroutines, no channels, no time.Now, no mutex)
/view     # read model built from engine events (snapshots, trade tape)
/api      # external-facing service: goroutine owner, command RPC, fanout to view + public events
```