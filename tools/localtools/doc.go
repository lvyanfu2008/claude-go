// Package localtools implements core tools for [skilltools.ParityToolRunner] (Read, Write, Edit, Glob, Grep, Bash, …).
//
// TypeScript source of truth: claude-code src/tools/FileReadTool, FileWriteTool, FileEditTool.
//
// Parity matrix: see [FileReadFeatureStatus], [FileWriteFeatureStatus], [FileEditFeatureStatus] in filetool_parity.go.
// Features not yet ported return explicit errors (e.g. [ErrReadPDFPagesNotImplementedInGo]) or [ParityStub] status — never silent success.
//
// Session read/write state: [ReadFileState] mirrors toolUseContext.readFileState for dedup and Write/Edit staleness.
package localtools
