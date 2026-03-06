package i18n_test

import (
	"testing"

	"github.com/DC-TechHQ/tais-core/i18n"
)

func TestGet_RegisteredCode(t *testing.T) {
	got := i18n.Get(i18n.ErrNotFound, i18n.LangRU)
	if got == "" || got == i18n.ErrNotFound {
		t.Errorf("expected RU translation, got %q", got)
	}
}

func TestGet_AllLanguages(t *testing.T) {
	langs := []string{i18n.LangTJ, i18n.LangRU, i18n.LangEN}
	for _, lang := range langs {
		got := i18n.Get(i18n.MsgSuccess, lang)
		if got == "" || got == i18n.MsgSuccess {
			t.Errorf("lang=%s: expected translation, got %q", lang, got)
		}
	}
}

func TestGet_UnknownCode_ReturnCode(t *testing.T) {
	got := i18n.Get("ErrSomethingCustom", i18n.LangRU)
	if got != "ErrSomethingCustom" {
		t.Errorf("expected code itself as fallback, got %q", got)
	}
}

func TestGet_FallbackToRU(t *testing.T) {
	i18n.Register(map[string]map[string]string{
		"TestOnlyRU": {
			i18n.LangRU: "Только русский",
		},
	})

	// Requested TJ but only RU exists — must return RU
	got := i18n.Get("TestOnlyRU", i18n.LangTJ)
	if got != "Только русский" {
		t.Errorf("expected RU fallback, got %q", got)
	}
}

func TestRegister_Concurrent(t *testing.T) {
	done := make(chan struct{})
	for range 10 {
		go func() {
			i18n.Register(map[string]map[string]string{
				"ConcurrentCode": {i18n.LangRU: "test"},
			})
			done <- struct{}{}
		}()
	}
	for range 10 {
		<-done
	}
}
