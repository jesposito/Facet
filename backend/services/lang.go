package services

import (
	"fmt"
	"strings"

	"github.com/abadojack/whatlanggo"
)

// minDetectableChars guards against trigram detection misfiring on very short
// strings (a three-word headline). Below this we fall back to the passive
// prompt rules instead of risking a wrong explicit directive.
const minDetectableChars = 12

// DetectContentLanguage returns the English name of the language of the first
// text with a reliable detection (e.g. "German"), or "" when that text is
// English, empty, or too short/ambiguous to call.
//
// Texts are checked in priority order: the first reliable detection wins, so
// pass the user's main content first and context fields after it. "" means
// "no explicit language directive" — English prompts already produce English.
func DetectContentLanguage(texts ...string) string {
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if len(t) < minDetectableChars {
			continue
		}
		info := whatlanggo.Detect(t)
		if !info.IsReliable() {
			continue
		}
		if info.Lang == whatlanggo.Eng {
			return ""
		}
		if name, ok := whatlanggo.Langs[info.Lang]; ok {
			return name
		}
	}
	return ""
}

// LanguagePromptDirective returns an explicit response-language instruction
// for the top of an AI prompt, or "" when lang is empty.
//
// Small local models (llama3.2 3B via Ollama) ignore passive "detect and keep
// the language" rules buried mid-prompt and mirror the prompt's dominant
// language, English. They do follow an explicit named language pinned to the
// first and last lines, which is why this exists alongside the passive rules.
func LanguagePromptDirective(lang string) string {
	if lang == "" {
		return ""
	}
	return fmt.Sprintf("CRITICAL LANGUAGE REQUIREMENT: The source content is written in %s. Write your ENTIRE response in %s. Do NOT translate to English.\n\n", lang, lang)
}

// LanguagePromptReminder returns a closing-line reinforcement of the response
// language, or "" when lang is empty. Append it to the prompt's final
// instruction line: weak models weight the last line the most.
func LanguagePromptReminder(lang string) string {
	if lang == "" {
		return ""
	}
	return fmt.Sprintf(" Respond ONLY in %s.", lang)
}
