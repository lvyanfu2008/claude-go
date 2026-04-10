package commands

// isSettingSourceEnabled mirrors src/utils/settings/constants.ts isSettingSourceEnabled (getEnabledSettingSources always includes policySettings + flagSettings).
// When EnabledSettingSources is nil, behaves like default TS allowedSettingSources (user, project, local + policy/flag). Empty slice = isolation: only policySettings/flagSettings match TS.
func (o LoadOptions) isSettingSourceEnabled(source string) bool {
	if o.SkillsPluginOnlyLocked {
		switch source {
		case "userSettings", "projectSettings":
			return false
		}
	}
	// TS getEnabledSettingSources: allowed ∪ { policySettings, flagSettings }
	if source == "policySettings" || source == "flagSettings" {
		return true
	}
	if o.EnabledSettingSources == nil {
		switch source {
		case "userSettings", "projectSettings", "localSettings":
			return true
		default:
			return false
		}
	}
	for _, s := range o.EnabledSettingSources {
		if s == source {
			return true
		}
	}
	return false
}
