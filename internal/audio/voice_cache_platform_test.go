//go:build darwin || windows

package audio

func resetVoiceCatalogCache() {
	voiceCatalogMu.Lock()
	defer voiceCatalogMu.Unlock()
	voiceCatalogCached = false
	voiceCatalog = voiceCatalogState{}
}
