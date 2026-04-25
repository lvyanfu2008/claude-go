package localtools

// Parity with claude-code FileReadTool / FileWriteTool / FileEditTool (see src/tools/File*Tool/*.ts).
// All features are listed here; unimplemented paths return explicit errors or stubs — never silent success.

// ParityStatus describes how closely Go matches TypeScript for a feature.
type ParityStatus uint8

const (
	// ParityImplemented: same wire shape and core behavior as TS for typical inputs.
	ParityImplemented ParityStatus = iota
	// ParityPartial: same JSON shape or subset; semantics differ (e.g. rough token estimate vs API count).
	ParityPartial
	// ParityStub: reserved; returns a clear error or empty implementation until filled in.
	ParityStub
)

// FileReadFeature names every Read-tool capability from TS (FileReadTool.ts + readFileInRange + pdf + notebook + image).
type FileReadFeature string

const (
	ReadFeatTextOffsetLimit     FileReadFeature = "read_text_offset_limit"
	ReadFeatReadFileStateDedup  FileReadFeature = "read_file_state_dedup"
	ReadFeatNotebookRawCells    FileReadFeature = "read_notebook_raw_cells"
	ReadFeatNotebookProcessed   FileReadFeature = "read_notebook_processed_cells" // TS processCell / mapNotebookCellsToToolResult
	ReadFeatImageBase64         FileReadFeature = "read_image_base64"
	ReadFeatImageTokenBudget    FileReadFeature = "read_image_token_budget_resize"
	ReadFeatPDFPagesExtract     FileReadFeature = "read_pdf_pages_extract"
	ReadFeatPDFFullDocument     FileReadFeature = "read_pdf_full_document"
	ReadFeatLargeFileStreaming  FileReadFeature = "read_large_file_streaming"
	ReadFeatPermissionsDenylist FileReadFeature = "read_permissions_denylist"
	ReadFeatUNCPathHandling     FileReadFeature = "read_unc_path_handling"
	ReadFeatBinaryExtensionDeny FileReadFeature = "read_binary_extension_deny"
	ReadFeatDevicePathBlock     FileReadFeature = "read_device_path_block"
	ReadFeatSimilarFileENOENT   FileReadFeature = "read_enoent_similar_file_suggestion"
	ReadFeatCyberRiskReminder   FileReadFeature = "read_cyber_risk_reminder_in_tool_result"
)

// FileWriteFeature names Write-tool capabilities (FileWriteTool.ts).
type FileWriteFeature string

const (
	WriteFeatSessionPermissions FileWriteFeature = "write_session_permissions"
	WriteFeatDenylist           FileWriteFeature = "write_denylist_rules"
	WriteFeatTeamMemSecrets     FileWriteFeature = "write_team_memory_secret_guard"
	WriteFeatGitDiffRemote      FileWriteFeature = "write_git_diff_telemetry"
	WriteFeatLSPNotify          FileWriteFeature = "write_lsp_did_change_save"
	WriteFeatVSCodeNotify       FileWriteFeature = "write_vscode_diff_notify"
	WriteFeatAtomicStaleness    FileWriteFeature = "write_atomic_staleness_section"
)

// FileEditFeature names Edit-tool capabilities (FileEditTool.ts).
type FileEditFeature string

const (
	EditFeatSessionPermissions FileEditFeature = "edit_session_permissions"
	EditFeatDenylist           FileEditFeature = "edit_denylist_rules"
	EditFeatTeamMemSecrets     FileEditFeature = "edit_team_memory_secret_guard"
	EditFeatSettingsFileRefine FileEditFeature = "edit_settings_file_validate"
	EditFeatNotebookRedirect   FileEditFeature = "edit_ipynb_redirect_notebook_edit"
	EditFeatGitDiffRemote      FileEditFeature = "edit_git_diff_telemetry"
	EditFeatLSPNotify          FileEditFeature = "edit_lsp_notify"
)

// FileReadFeatureStatus reports TS parity for a Read feature (for tests / diagnostics).
func FileReadFeatureStatus(f FileReadFeature) ParityStatus {
	switch f {
	case ReadFeatTextOffsetLimit, ReadFeatReadFileStateDedup, ReadFeatBinaryExtensionDeny,
		ReadFeatDevicePathBlock, ReadFeatSimilarFileENOENT:
		return ParityImplemented
	case ReadFeatNotebookRawCells, ReadFeatImageBase64:
		return ParityImplemented
	case ReadFeatCyberRiskReminder:
		return ParityImplemented
	case ReadFeatNotebookProcessed, ReadFeatImageTokenBudget, ReadFeatPDFPagesExtract,
		ReadFeatPDFFullDocument, ReadFeatLargeFileStreaming, ReadFeatPermissionsDenylist,
		ReadFeatUNCPathHandling:
		return ParityImplemented
	default:
		return ParityStub
	}
}

// FileWriteFeatureStatus reports TS parity for Write in Go localtools.
// All features use dependency injection (WriteDeps) — parity_runner.go wires the callbacks;
// standalone callers pass nil to skip. See [WriteDeps] in write.go for callback signatures.
func FileWriteFeatureStatus(f FileWriteFeature) ParityStatus {
	switch f {
	case WriteFeatAtomicStaleness:
		return ParityImplemented
	case WriteFeatSessionPermissions:
		return ParityImplemented
	case WriteFeatDenylist:
		return ParityImplemented
	case WriteFeatTeamMemSecrets:
		return ParityImplemented
	case WriteFeatGitDiffRemote:
		return ParityImplemented
	case WriteFeatLSPNotify:
		return ParityImplemented
	case WriteFeatVSCodeNotify:
		return ParityImplemented
	default:
		return ParityStub
	}
}
// FileEditFeatureStatus reports TS parity for Edit in Go localtools.
// All features use dependency injection (EditDeps) — parity_runner.go wires the callbacks;
// standalone callers pass nil to skip. See [EditDeps] in edit.go for callback signatures.
func FileEditFeatureStatus(f FileEditFeature) ParityStatus {
	switch f {
	case EditFeatSessionPermissions:
		return ParityImplemented
	case EditFeatDenylist:
		return ParityImplemented
	case EditFeatTeamMemSecrets:
		return ParityImplemented
	case EditFeatSettingsFileRefine:
		return ParityImplemented
	case EditFeatNotebookRedirect:
		return ParityImplemented
	case EditFeatGitDiffRemote:
		return ParityImplemented
	case EditFeatLSPNotify:
		return ParityImplemented
	default:
		return ParityStub
	}
}
