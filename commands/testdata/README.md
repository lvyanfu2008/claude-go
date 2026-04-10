# Golden / drift JSON for `goc/commands`

- **`builtin_commands_default.json`** — TS `getBuiltinCommandsTable()` with plain `bun` (no dev feature flags). After changing `src/commands.ts` `COMMANDS()`, run `bun run export:builtin-commands`, then `cd goc && go run ./cmd/gencode-handwritten` to refresh `handwritten/z_jsondata.go`.
- **`internal_only_commands.json`** — `INTERNAL_ONLY_COMMANDS` data-only dump with `USER_TYPE=ant` (stubs preserved). Refresh with:

  ```bash
  USER_TYPE=ant bun -e "
  import { INTERNAL_ONLY_COMMANDS } from './src/commands.ts'
  function toDataOnly(cmd) { const o = {}; for (const k of Object.keys(cmd)) { const v = cmd[k]; if (typeof v === 'function') continue; o[k] = v }; return o }
  await Bun.write('goc/commands/testdata/internal_only_commands.json', JSON.stringify(INTERNAL_ONLY_COMMANDS.map(toDataOnly), null, 2) + '\n')
  "
  cd goc && go run ./cmd/gencode-handwritten
  ```

- **`bundled_skills_golden.json`** / **`builtin_plugin_skills_golden.json`** — `bun run export:bundled-skills` / `export:builtin-plugin-skills`.

- **`builtin_output_style_explanatory.txt`** / **`builtin_output_style_learning.txt`** — prompts from `src/constants/outputStyles.ts` `OUTPUT_STYLE_CONFIG` (embedded by `gou_demo_output_style.go`). Refresh after changing built-in output styles:

  ```bash
  bun -e "
  import { OUTPUT_STYLE_CONFIG } from './src/constants/outputStyles.ts'
  const e = OUTPUT_STYLE_CONFIG.Explanatory
  const l = OUTPUT_STYLE_CONFIG.Learning
  if (!e?.prompt || !l?.prompt) throw new Error('missing')
  await Bun.write('goc/commands/testdata/builtin_output_style_explanatory.txt', e.prompt.trimEnd() + '\\n')
  await Bun.write('goc/commands/testdata/builtin_output_style_learning.txt', l.prompt.trimEnd() + '\\n')
  "
  ```
