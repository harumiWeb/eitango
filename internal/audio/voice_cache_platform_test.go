//go:build darwin || windows

package audio

// resetVoiceCatalogCache clears the process-global test cache between platform-specific cases.
func resetVoiceCatalogCache() {
	voiceCatalogMu.Lock()
	defer voiceCatalogMu.Unlock()
	voiceCatalogCached = false
	voiceCatalog = voiceCatalogState{}
}
