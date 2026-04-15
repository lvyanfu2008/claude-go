package localtools

import "testing"

func TestFileReadFeatureStatus_exhaustive(t *testing.T) {
	features := []FileReadFeature{
		ReadFeatTextOffsetLimit,
		ReadFeatReadFileStateDedup,
		ReadFeatNotebookRawCells,
		ReadFeatNotebookProcessed,
		ReadFeatImageBase64,
		ReadFeatImageTokenBudget,
		ReadFeatPDFPagesExtract,
		ReadFeatPDFFullDocument,
		ReadFeatLargeFileStreaming,
		ReadFeatPermissionsDenylist,
		ReadFeatUNCPathHandling,
		ReadFeatBinaryExtensionDeny,
		ReadFeatDevicePathBlock,
		ReadFeatSimilarFileENOENT,
		ReadFeatCyberRiskReminder,
	}
	for _, f := range features {
		s := FileReadFeatureStatus(f)
		if s != ParityImplemented && s != ParityPartial && s != ParityStub {
			t.Fatalf("bad status for %q: %v", f, s)
		}
	}
}
