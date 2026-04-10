# `LoadAllCommands` 与 TS `loadAllCommands` 对照

**产品架构（Go 编排主路径）**见 **[`docs/plans/architecture-go-orchestration.md`](../../docs/plans/architecture-go-orchestration.md)**。

完整路线、阶段表与风险见 **[`docs/plans/goc-load-all-commands.md`](../../docs/plans/goc-load-all-commands.md)**。

Go **`goc/commands.LoadAllCommands`** 对齐 **`src/commands.ts`** **`loadAllCommands`** 的拼接顺序：

`bundledSkills` → `builtinPluginSkills` → **`getSkillDirCommands`** → `workflowCommands` → `pluginCommands` → `pluginSkills` → **`COMMANDS()`**。

**bundled** / **builtin plugin skills** / **内置 `COMMANDS()`** 由 Go **[`handwritten`](handwritten/)** 组装（[`handwritten_load.go`](handwritten_load.go)）；`FEATURE_*` / `USER_TYPE` / `CLAUDE_CODE_GO_ASSUME_3P` 等见 [`featuregates`](featuregates/gates.go)。导出脚本仅作 **drift 对照**，写入 [`testdata/`](testdata/)（见 [`testdata/README.md`](testdata/README.md)）。**plugin marketplace** 在 Go 中仍为空桩（**P5 明确延后，暂不实现**）。**Workflow（P6 延后）**：[`workflow_load.go`](workflow_load.go) 可在显式 **`LoadOptions.WorkflowScripts: true`** 时列出元数据；**[`DefaultLoadOptions`](load_all.go) 默认不启用**。TS **`getWorkflowCommands`** 仍为 stub，执行未实现。不含 TS 侧 `getPromptForCommand` / `load` 等运行时行为，**availability / isEnabled** 由宿主按需过滤（与 TS `getCommands` 一致）。

**文件发现**：legacy `/commands` 下 `*.md` 使用 [`findMarkdownFilesNative`](walk_markdown.go)（与 TS `findMarkdownFilesNative`：无 `.gitignore`、跟随 symlink 目录、防环）。动态技能：[`discover_skill_dirs.go`](discover_skill_dirs.go)（`DiscoverSkillDirsForPaths`）、[`git_checkignore.go`](git_checkignore.go)；[`dynamic_skills_load.go`](dynamic_skills_load.go)（`LoadSkillsFromSkillDirectories`、`LoadDynamicSkillCommandsForPaths`、`LoadAndGetCommandsWithFilePathsDynamic`，对齐 `addSkillDirectories` 合并与 **`getDynamicSkills()`** 的 **Map 插入顺序**（浅→深合并时每个 **name 首次出现**的顺序，**不**再按 name 排序）。

**同一 `/skills` 目录内**：Go [`loadSkillsFromDir`](skill_md.go) 使用 [`os.ReadDir`](https://pkg.go.dev/os#ReadDir)（**按文件名排序**）。TS [`loadSkillsFromSkillsDir`](../../src/skills/loadSkillsDir.ts) 使用 `fs.readdir` **顺序未规范**，与 Go 可能不一致；需要可复现顺序时以 **字典序 skill 子目录名** 为准（Go 侧已稳定）。

**Skill 来源（含 P2）**：**managed policy** `skills/`（`ManagedFilePath()` + `.claude/skills`，可由 **`CLAUDE_CODE_MANAGED_SETTINGS_PATH`** 覆盖）、**user** `CLAUDE_CONFIG_DIR`/`~/.claude` 下 **`skills/`**、**project** 自 **`cwd` 向上**的 **`.claude/skills/`**（停止边界由 **`resolveStopBoundary`** 决定，与 TS [`markdownConfigLoader.ts`](../../src/utils/markdownConfigLoader.ts) 对齐；**worktree 稀疏检出**时若工作区根无 `.claude/skills` 则追加**主仓** `.claude/skills`，见 `appendWorktreeMainRepoProjectDirIfMissing`）；**`LoadOptions.SessionProjectRoot`** 对应 `getProjectRoot()`，用于嵌套 git 仓时扩展到会话主仓根），**`LoadOptions.AddSkillDirs`**；**legacy** **`.claude/commands`** 下 `*.md`（`loadedFrom: commands_DEPRECATED`；TS 对 legacy **不**应用 `paths` 条件，Go 同样忽略 `paths`）。**`LoadOptions.EnabledSettingSources`** / **`isSettingSourceEnabled`** 门控 user/project 扫描（**`policySettings`/`flagSettings`** 在 TS 中恒参与 enabled 集合；managed **`skills/`** 仍仅由 **`CLAUDE_CODE_DISABLE_POLICY_SKILLS`** 关闭）。**条件 skills**：默认只返回 unconditional（**`IncludeConditionalSkills`** 为 true 时保留带 `paths` 的项）。**`CLAUDE_CODE_DISABLE_POLICY_SKILLS`** 仅跳过 **managed `skills/`**（与 TS 一致；**managed `commands/`** 仍参与 legacy）。**`LoadOptions.SkillsPluginOnlyLocked`** 对应 plugin-only：仅保留 managed policy **`skills/`**。加载顺序与 TS 扁平列表一致后，按 **`filepath.EvalSymlinks`** 对 markdown 路径去重（先出现者保留）。

**`meetsAvailabilityRequirement` / `isEnabled`** 不在 `LoadAllCommands` 内执行（与 TS 一致）。与 TS **`getCommands`** 对齐：先 **`FilterGetCommands`**（或 **`LoadAndFilterCommands`**）；若有 **`getDynamicSkills`** 等价列表，再 **`GetCommandsWithDynamicSkills`** / **`LoadAndGetCommandsWithDynamic`**（[`get_commands.go`](get_commands.go)：`BuiltinCommandNameSet`、`InsertDynamicSkillsBeforeBuiltins`）。**条件 skills 运行时**：[`conditional_runtime.go`](conditional_runtime.go)（`syncConditionalSkillsFromLoaded`、`ActivateConditionalSkillsForPaths`、`SnapshotDynamicConditionalSkillsForMerge`）对齐 TS `conditionalSkills` / `activateConditionalSkillsForPaths` / `dynamicSkills`；路径匹配使用 **`go-gitignore`**（与 TS `ignore()` 尽力一致）。

实现入口：**`load_all.go`**、**`skill_dir_load.go`**、**`skill_md.go`**、**`legacy_commands.go`**、**`dedup.go`**、**`paths.go`**、**`git_boundary.go`**。
