# pkg/errors/

Stable string codes for agents and HTTP clients.

| Code | When |
|---|---|
| `invalid_query` | Bad input (missing chain/asset, fractional atomic, bad override) |
| `insufficient_liquidity` | Empty pay_to, max amount, plan action insufficient / corridor_unsupported on execute |
| `liquidity_rail_unavailable` | Execute requested but Circle rails not configured |

Use `New` / `Wrap` / `CodeOf`. Prefer `errors.As` on `*Error`.
