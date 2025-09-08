package identifier

import (
	"strings"

	"github.com/go-enry/go-enry/v2"
)

// Heuristically determine the language of a file given its metadata and content
func GetLanguage(filename string, content []byte) Language {
	if strings.HasSuffix(filename, ".c") || strings.HasSuffix(filename, ".h") ||
		strings.HasSuffix(filename, ".c.in") || strings.HasSuffix(filename, ".h.in") {
		return LanguageC
	} else if strings.HasSuffix(filename, ".java") {
		return LanguageJava
	}

	candidates := enry.GetLanguages(filename, content)
	for _, candidate := range candidates {
		mapping := languageMapping[candidate]
		if mapping != LanguageInvalid {
			return mapping
		}
	}

	return LanguageInvalid
}
