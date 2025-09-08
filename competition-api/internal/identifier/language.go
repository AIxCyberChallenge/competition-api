package identifier

import (
	"errors"
	"fmt"
)

type Language string

const (
	LanguageC    = "c"
	LanguageJava = "java"
	// Returned when we do not determine the language to be `java` or `c`
	LanguageInvalid = ""
)

func toLanguage(v string) (Language, error) {
	vLanguage := Language(v)
	switch vLanguage {
	case LanguageC, LanguageJava:
		return vLanguage, nil
	default:
		return LanguageInvalid, errors.New(`must be one of "java" or "c"`)
	}
}

func (l Language) String() string {
	return string(l)
}

func (l *Language) Set(v string) error {
	vLanguage, err := toLanguage(v)
	if err != nil {
		return err
	}

	*l = vLanguage
	return nil
}

func (*Language) Type() string {
	return "Language"
}

// Allow use as a cobra flag

type LanguageSlice []Language

func (l *LanguageSlice) String() string {
	return fmt.Sprintf("%v", *l)
}

func (l *LanguageSlice) Set(v string) error {
	vLanguage, err := toLanguage(v)
	if err != nil {
		return err
	}

	*l = append(*l, vLanguage)
	return nil
}

func (*LanguageSlice) Type() string {
	return "LanguageSlice"
}

// go-enry to useful language mappings
var languageMapping = map[string]Language{
	"C":    LanguageC,
	"C++":  LanguageC,
	"Java": LanguageJava,
}
