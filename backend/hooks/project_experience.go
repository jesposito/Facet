package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const maxProjectExperienceBytes = 500000

var (
	projectExperienceIDPattern       = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
	projectExperienceFieldKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,31}$`)
	projectExperienceFragmentPattern = regexp.MustCompile(`^#[a-zA-Z][a-zA-Z0-9_-]*$`)
	projectExperienceTones           = map[string]bool{"paper": true, "aqua": true, "sand": true, "green": true, "pink": true, "ink": true, "yellow": true, "orange": true}
)

func RegisterProjectExperienceHooks(app *pocketbase.PocketBase) {
	for _, collection := range []string{"projects", "demo_projects"} {
		app.OnRecordCreate(collection).BindFunc(validateProjectExperienceRecord)
		app.OnRecordUpdate(collection).BindFunc(validateProjectExperienceRecord)
	}
}

func validateProjectExperienceRecord(e *core.RecordEvent) error {
	if err := validateProjectExperience(e.Record.Get("experience")); err != nil {
		return apis.NewBadRequestError("Check the project experience: "+err.Error(), nil)
	}
	return e.Next()
}

func validateProjectExperience(value any) error {
	raw, err := projectExperienceJSON(value)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if len(raw) > maxProjectExperienceBytes {
		return fmt.Errorf("experience exceeds %d bytes", maxProjectExperienceBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return fmt.Errorf("malformed JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("experience must contain one JSON value")
	}
	if decoded == nil {
		return nil
	}

	doc, err := projectExperienceObject(decoded, "experience", []string{"version", "enabled", "layout", "hero", "sections"}, []string{"design"})
	if err != nil {
		return err
	}
	version, ok := doc["version"].(json.Number)
	if !ok || version.String() == "" {
		return fmt.Errorf("experience.version must be 1")
	}
	versionNumber, err := version.Float64()
	if err != nil || versionNumber != 1 {
		return fmt.Errorf("experience.version must be 1")
	}
	if _, ok := doc["enabled"].(bool); !ok {
		return fmt.Errorf("experience.enabled must be a boolean")
	}
	if doc["layout"] != "immersive" {
		return fmt.Errorf("experience.layout must be immersive")
	}
	if value, exists := doc["design"]; exists {
		design, err := projectExperienceObject(value, "experience.design", []string{"preset"}, []string{"width"})
		if err != nil {
			return err
		}
		if preset, ok := design["preset"].(string); !ok || preset != "playful" && preset != "technical" && preset != "editorial" {
			return fmt.Errorf("experience.design.preset is invalid")
		}
		if value, exists := design["width"]; exists {
			if width, ok := value.(string); !ok || width != "standard" && width != "wide" {
				return fmt.Errorf("experience.design.width is invalid")
			}
		}
	}

	hero, err := projectExperienceObject(doc["hero"], "experience.hero", []string{"intro", "tone", "imagePosition"}, []string{"image", "action", "imageNote"})
	if err != nil {
		return err
	}
	if _, err := projectExperienceProse(hero, "intro", "experience.hero.intro"); err != nil {
		return err
	}
	if _, exists := hero["imageNote"]; exists {
		if _, err := projectExperienceString(hero, "imageNote", "experience.hero.imageNote", 200, true); err != nil {
			return err
		}
	}
	if tone, ok := hero["tone"].(string); !ok || !projectExperienceTones[tone] {
		return fmt.Errorf("experience.hero.tone is invalid")
	}
	if position := hero["imagePosition"]; position != "left" && position != "right" {
		return fmt.Errorf("experience.hero.imagePosition is invalid")
	}
	if value, exists := hero["image"]; exists {
		if _, err := projectExperienceImage(value, "experience.hero.image"); err != nil {
			return err
		}
	}
	if value, exists := hero["action"]; exists {
		action, err := projectExperienceObject(value, "experience.hero.action", []string{"label", "href"}, nil)
		if err != nil {
			return err
		}
		if _, err := projectExperienceTitle(action, "label", "experience.hero.action.label"); err != nil {
			return err
		}
		if _, err := projectExperienceURL(action, "href", "experience.hero.action.href", true); err != nil {
			return err
		}
	}

	sections, ok := doc["sections"].([]any)
	if !ok || len(sections) > 30 {
		return fmt.Errorf("experience.sections must be an array with at most 30 items")
	}
	ids := make(map[string]struct{})
	for i, value := range sections {
		path := fmt.Sprintf("experience.sections[%d]", i)
		section, err := projectExperienceObject(value, path, []string{"id", "title", "intro", "tone", "layout", "blocks"}, []string{"headingLayout"})
		if err != nil {
			return err
		}
		if err := projectExperienceID(section, path, ids); err != nil {
			return err
		}
		if _, err := projectExperienceTitle(section, "title", path+".title"); err != nil {
			return err
		}
		if _, err := projectExperienceProse(section, "intro", path+".intro"); err != nil {
			return err
		}
		if tone, ok := section["tone"].(string); !ok || !projectExperienceTones[tone] {
			return fmt.Errorf("%s.tone is invalid", path)
		}
		if layout := section["layout"]; layout != "stack" && layout != "split" {
			return fmt.Errorf("%s.layout is invalid", path)
		}
		if value, exists := section["headingLayout"]; exists {
			if headingLayout, ok := value.(string); !ok || headingLayout != "stack" && headingLayout != "split" {
				return fmt.Errorf("%s.headingLayout is invalid", path)
			}
		}
		blocks, ok := section["blocks"].([]any)
		if !ok || len(blocks) > 20 {
			return fmt.Errorf("%s.blocks must be an array with at most 20 items", path)
		}
		for j, block := range blocks {
			if err := validateProjectExperienceBlock(block, fmt.Sprintf("%s.blocks[%d]", path, j), ids); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateProjectExperienceBlock(value any, path string, ids map[string]struct{}) error {
	base, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", path)
	}
	if err := projectExperienceID(base, path, ids); err != nil {
		return err
	}

	switch base["type"] {
	case "text":
		block, err := projectExperienceObject(value, path, []string{"id", "type", "body"}, nil)
		if err != nil {
			return err
		}
		_, err = projectExperienceProse(block, "body", path+".body")
		return err
	case "image":
		block, err := projectExperienceObject(value, path, []string{"id", "type", "image", "caption"}, nil)
		if err != nil {
			return err
		}
		if _, err := projectExperienceImage(block["image"], path+".image"); err != nil {
			return err
		}
		_, err = projectExperienceMetadata(block, "caption", path+".caption")
		return err
	case "disclosure":
		block, err := projectExperienceObject(value, path, []string{"id", "type", "title", "body"}, nil)
		if err != nil {
			return err
		}
		if _, err := projectExperienceTitle(block, "title", path+".title"); err != nil {
			return err
		}
		_, err = projectExperienceProse(block, "body", path+".body")
		return err
	case "video":
		block, err := projectExperienceObject(value, path, []string{"id", "type", "title", "lessons"}, []string{"presentation"})
		if err != nil {
			return err
		}
		if _, err := projectExperienceTitle(block, "title", path+".title"); err != nil {
			return err
		}
		if value, exists := block["presentation"]; exists {
			if presentation, ok := value.(string); !ok || presentation != "stack" && presentation != "sidebar" && presentation != "spotlight" {
				return fmt.Errorf("%s.presentation is invalid", path)
			}
		}
		lessons, ok := block["lessons"].([]any)
		if !ok || len(lessons) > 30 {
			return fmt.Errorf("%s.lessons must be an array with at most 30 items", path)
		}
		for i, value := range lessons {
			if err := validateProjectExperienceLesson(value, fmt.Sprintf("%s.lessons[%d]", path, i), ids); err != nil {
				return err
			}
		}
		return nil
	case "code":
		block, err := projectExperienceObject(value, path, []string{"id", "type", "title", "steps"}, nil)
		if err != nil {
			return err
		}
		if _, err := projectExperienceTitle(block, "title", path+".title"); err != nil {
			return err
		}
		steps, ok := block["steps"].([]any)
		if !ok || len(steps) > 30 {
			return fmt.Errorf("%s.steps must be an array with at most 30 items", path)
		}
		for i, value := range steps {
			if err := validateProjectExperienceStep(value, fmt.Sprintf("%s.steps[%d]", path, i), ids); err != nil {
				return err
			}
		}
		return nil
	case "story":
		return validateProjectExperienceStory(value, path)
	default:
		return fmt.Errorf("%s.type is unknown", path)
	}
}

func validateProjectExperienceLesson(value any, path string, ids map[string]struct{}) error {
	lesson, err := projectExperienceObject(value, path, []string{"id", "title", "src", "description", "poster", "transcript", "credit"}, []string{"captions"})
	if err != nil {
		return err
	}
	if err := projectExperienceID(lesson, path, ids); err != nil {
		return err
	}
	if _, err := projectExperienceTitle(lesson, "title", path+".title"); err != nil {
		return err
	}
	if _, err := projectExperienceURL(lesson, "src", path+".src", false); err != nil {
		return err
	}
	if _, err := projectExperienceMetadata(lesson, "description", path+".description"); err != nil {
		return err
	}
	if _, err := projectExperienceURL(lesson, "poster", path+".poster", false); err != nil {
		return err
	}
	if _, err := projectExperienceProse(lesson, "transcript", path+".transcript"); err != nil {
		return err
	}
	if _, err := projectExperienceMetadata(lesson, "credit", path+".credit"); err != nil {
		return err
	}
	if value, exists := lesson["captions"]; exists {
		captions, err := projectExperienceObject(value, path+".captions", []string{"src", "language", "label"}, nil)
		if err != nil {
			return err
		}
		if _, err := projectExperienceURL(captions, "src", path+".captions.src", false); err != nil {
			return err
		}
		if _, err := projectExperienceMetadata(captions, "language", path+".captions.language"); err != nil {
			return err
		}
		if _, err := projectExperienceTitle(captions, "label", path+".captions.label"); err != nil {
			return err
		}
	}
	return nil
}

func validateProjectExperienceStep(value any, path string, ids map[string]struct{}) error {
	step, err := projectExperienceObject(value, path, []string{"id", "title", "body", "code", "language"}, []string{"image"})
	if err != nil {
		return err
	}
	if err := projectExperienceID(step, path, ids); err != nil {
		return err
	}
	if _, err := projectExperienceTitle(step, "title", path+".title"); err != nil {
		return err
	}
	if _, err := projectExperienceProse(step, "body", path+".body"); err != nil {
		return err
	}
	if _, err := projectExperienceProse(step, "code", path+".code"); err != nil {
		return err
	}
	if _, err := projectExperienceMetadata(step, "language", path+".language"); err != nil {
		return err
	}
	if value, exists := step["image"]; exists {
		if _, err := projectExperienceImage(value, path+".image"); err != nil {
			return err
		}
	}
	return nil
}

func validateProjectExperienceStory(value any, path string) error {
	story, err := projectExperienceObject(value, path, []string{"id", "type", "title", "template", "fields", "pauseMs", "credit"}, []string{"presentation"})
	if err != nil {
		return err
	}
	if _, err := projectExperienceTitle(story, "title", path+".title"); err != nil {
		return err
	}
	if value, exists := story["presentation"]; exists {
		if presentation, ok := value.(string); !ok || presentation != "plain" && presentation != "paper" {
			return fmt.Errorf("%s.presentation is invalid", path)
		}
	}
	if _, err := projectExperienceProse(story, "template", path+".template"); err != nil {
		return err
	}
	if _, err := projectExperienceMetadata(story, "credit", path+".credit"); err != nil {
		return err
	}
	pause, ok := story["pauseMs"].(json.Number)
	if !ok {
		return fmt.Errorf("%s.pauseMs must be a number", path)
	}
	pauseMs, err := pause.Float64()
	if err != nil || math.IsNaN(pauseMs) || math.IsInf(pauseMs, 0) || pauseMs < 0 || pauseMs > 5000 {
		return fmt.Errorf("%s.pauseMs must be between 0 and 5000", path)
	}
	fields, ok := story["fields"].([]any)
	if !ok || len(fields) > 12 {
		return fmt.Errorf("%s.fields must be an array with at most 12 items", path)
	}
	keys := make(map[string]struct{})
	for i, value := range fields {
		fieldPath := fmt.Sprintf("%s.fields[%d]", path, i)
		field, err := projectExperienceObject(value, fieldPath, []string{"key", "label", "initial"}, nil)
		if err != nil {
			return err
		}
		key, ok := field["key"].(string)
		if !ok || !projectExperienceFieldKeyPattern.MatchString(key) {
			return fmt.Errorf("%s.key is invalid", fieldPath)
		}
		if _, exists := keys[key]; exists {
			return fmt.Errorf("%s.key is duplicated", fieldPath)
		}
		keys[key] = struct{}{}
		if _, err := projectExperienceTitle(field, "label", fieldPath+".label"); err != nil {
			return err
		}
		if _, err := projectExperienceMetadata(field, "initial", fieldPath+".initial"); err != nil {
			return err
		}
	}
	return nil
}

func projectExperienceObject(value any, path string, required, optional []string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", path)
	}
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, key := range required {
		allowed[key] = true
		if _, exists := object[key]; !exists {
			return nil, fmt.Errorf("%s.%s is required", path, key)
		}
	}
	for _, key := range optional {
		allowed[key] = true
	}
	for key := range object {
		if !allowed[key] {
			return nil, fmt.Errorf("%s.%s is unknown", path, key)
		}
	}
	return object, nil
}

func projectExperienceID(object map[string]any, path string, ids map[string]struct{}) error {
	id, ok := object["id"].(string)
	if !ok || !projectExperienceIDPattern.MatchString(id) {
		return fmt.Errorf("%s.id is invalid", path)
	}
	if _, exists := ids[id]; exists {
		return fmt.Errorf("%s.id %q is duplicated", path, id)
	}
	ids[id] = struct{}{}
	return nil
}

func projectExperienceImage(value any, path string) (map[string]any, error) {
	image, err := projectExperienceObject(value, path, []string{"src", "alt"}, []string{"decorative"})
	if err != nil {
		return nil, err
	}
	if _, err := projectExperienceURL(image, "src", path+".src", false); err != nil {
		return nil, err
	}
	if _, err := projectExperienceMetadata(image, "alt", path+".alt"); err != nil {
		return nil, err
	}
	if value, exists := image["decorative"]; exists {
		if _, ok := value.(bool); !ok {
			return nil, fmt.Errorf("%s.decorative must be a boolean", path)
		}
	}
	return image, nil
}

func projectExperienceTitle(object map[string]any, key, path string) (string, error) {
	return projectExperienceString(object, key, path, 200, true)
}

func projectExperienceMetadata(object map[string]any, key, path string) (string, error) {
	return projectExperienceString(object, key, path, 500, true)
}

func projectExperienceProse(object map[string]any, key, path string) (string, error) {
	return projectExperienceString(object, key, path, 50000, false)
}

func projectExperienceString(object map[string]any, key, path string, max int, singleLine bool) (string, error) {
	value, ok := object[key].(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", path)
	}
	if utf8.RuneCountInString(value) > max {
		return "", fmt.Errorf("%s exceeds %d characters", path, max)
	}
	if singleLine && strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("%s must be single-line", path)
	}
	return value, nil
}

func projectExperienceURL(object map[string]any, key, path string, link bool) (string, error) {
	value, err := projectExperienceString(object, key, path, 2048, true)
	if err != nil {
		return "", err
	}
	if value == "" {
		return value, nil
	}
	for _, r := range value {
		if r <= 0x20 || r == 0x7f || r == '\\' {
			return "", fmt.Errorf("%s contains unsafe characters", path)
		}
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return value, nil
	}
	if link && projectExperienceFragmentPattern.MatchString(value) {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("%s must be a root-relative or credential-free HTTPS URL", path)
	}
	return value, nil
}

func projectExperienceJSON(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		return []byte(v), nil
	case []byte:
		return v, nil
	}
	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return rv.Bytes(), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("experience is not valid JSON: %w", err)
	}
	return raw, nil
}
