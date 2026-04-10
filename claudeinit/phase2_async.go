package claudeinit

// phase2AsyncSideEffects: TS fire-and-forget tasks (OAuth, JetBrains, repo detect, remote promises).
func phase2AsyncSideEffects() {
	// P2d detectCurrentRepository (async warm-up; [DumpState] also resolves synchronously).
	startDetectRepositoryBackground()
}
