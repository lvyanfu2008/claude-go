# /loop

Run a prompt or slash command on a recurring interval.

## When To Use

When the user wants to set up a recurring task, poll for status, or run something repeatedly on an interval (for example "check the deploy every 5 minutes" or "keep running /babysit-prs"). Do not use for one-off tasks.

## Arguments

`[interval] <prompt>`

- Default interval: `10m`
- Supported units: `s`, `m`, `h`, `d`

## Parsing Rules (TS Parity, In Priority Order)

1. If the first token matches `^\d+[smhd]$`, treat it as interval; rest is prompt.
2. Otherwise, if input ends with `every <N><unit>` or `every <N> <unit-word>`, extract that interval.
3. Otherwise, use default `10m` and treat whole input as prompt.
4. If prompt is empty after parsing, show usage and stop.

Important caveat for rule 2: only treat trailing `every ...` as interval when it is actually a time expression. For example, `check every PR` should not parse an interval.

## Usage Message

When args are missing or invalid, show:

`Usage: /loop [interval] <prompt>`

## Interval To Cron Mapping

Supported conversion:

| Interval pattern | Cron expression | Notes |
| --- | --- | --- |
| `Nm` where `N <= 59` | `*/N * * * *` | every N minutes |
| `Nm` where `N >= 60` | `0 */H * * *` | round to hours (`H = N/60`, must divide 24) |
| `Nh` where `N <= 23` | `0 */N * * *` | every N hours |
| `Nd` | `0 0 */N * *` | every N days at local midnight |
| `Ns` | treat as `ceil(N/60)m` | cron minimum granularity is 1 minute |

If interval cannot be represented cleanly, round to nearest clean interval and explicitly tell the user.

## Action Flow

1. Parse interval and prompt per rules above.
2. Create recurring cron job with parsed prompt.
3. Confirm what was scheduled, cron expression, human-readable cadence, retention policy, and cancellation method with job id.
4. Immediately execute the parsed prompt once now (do not wait for first cron trigger).
5. If parsed prompt is slash command, invoke as slash command; otherwise execute as normal request.

## Usage Examples

- `/loop 5m /babysit-prs`
- `/loop 30m check the deploy`
- `/loop 1h /standup 1`
- `/loop check the deploy every 20m`
- `/loop check the deploy` (defaults to `10m`)
