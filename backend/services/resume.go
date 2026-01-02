package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ResumeService handles AI-powered resume generation
type ResumeService struct {
	ai *AIService
}

// GenerationConfig contains settings for resume generation
type GenerationConfig struct {
	TargetRole string   `json:"target_role"`
	Style      string   `json:"style"`  // chronological, functional, hybrid
	Length     string   `json:"length"` // one-page, two-page, full
	Emphasis   []string `json:"emphasis"`
}

// ViewData represents the complete view data for resume generation
type ViewData struct {
	Profile      map[string]interface{}              `json:"profile"`
	Sections     map[string][]map[string]interface{} `json:"sections"`
	SectionOrder []string                            `json:"section_order"`
	HeroHeadline string                              `json:"hero_headline,omitempty"`
	HeroSummary  string                              `json:"hero_summary,omitempty"`
}

// NewResumeService creates a new resume service
func NewResumeService(ai *AIService) *ResumeService {
	return &ResumeService{ai: ai}
}

// GenerateResume generates a resume from view data
func (r *ResumeService) GenerateResume(
	ctx context.Context,
	provider *AIProvider,
	viewData *ViewData,
	config *GenerationConfig,
	format string,
) ([]byte, error) {
	log.Printf("[AI-PRINT] Starting resume generation, format: %s, config: %+v", format, config)

	// 1. Build the prompt
	prompt := r.buildResumePrompt(viewData, config)
	log.Printf("[AI-PRINT] Prompt built, length: %d chars", len(prompt))

	// 2. Call AI to generate markdown
	markdown, err := r.ai.ImproveContent(ctx, provider, prompt)
	if err != nil {
		log.Printf("[AI-PRINT] AI call failed: %v", err)
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}
	log.Printf("[AI-PRINT] AI response received, length: %d chars", len(markdown))

	// 3. Clean up the markdown (remove code blocks if present)
	markdown = r.cleanMarkdown(markdown)
	log.Printf("[AI-PRINT] Markdown cleaned, length: %d chars", len(markdown))

	// 4. Convert to requested format
	var output []byte
	switch format {
	case "pdf":
		output, err = r.convertToPDF(markdown)
	case "docx":
		output, err = r.convertToDOCX(markdown)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		log.Printf("[AI-PRINT] Conversion to %s failed: %v", format, err)
		return nil, err
	}

	log.Printf("[AI-PRINT] Successfully generated %s, size: %d bytes", format, len(output))
	return output, nil
}

// buildResumePrompt creates the AI prompt for resume generation
func (r *ResumeService) buildResumePrompt(viewData *ViewData, config *GenerationConfig) string {
	var sb strings.Builder

	sb.WriteString(`You are an expert resume writer. Generate a professional resume in clean Markdown format.

IMPORTANT WRITING STYLE RULES:
- Write like a human, not an AI. Be direct and natural.
- NEVER use em dashes (—). Use commas, periods, or "and" instead.
- NEVER use words like "delve", "leverage", "utilize", "spearheaded", "synergy", "cutting-edge", "passionate"
- Use active voice and strong action verbs (led, built, designed, implemented, improved)
- Be concise - every word should add value
- Quantify achievements where data is provided (don't invent numbers)
- Focus on impact and results, not just responsibilities

`)

	// Add target context
	sb.WriteString("TARGET CONTEXT:\n")
	if config.TargetRole != "" {
		sb.WriteString(fmt.Sprintf("- Target Role: %s\n", config.TargetRole))
	}
	if config.Style != "" {
		sb.WriteString(fmt.Sprintf("- Resume Style: %s\n", config.Style))
	}
	if config.Length != "" {
		sb.WriteString(fmt.Sprintf("- Length Constraint: %s\n", config.Length))
	}
	if len(config.Emphasis) > 0 {
		sb.WriteString(fmt.Sprintf("- Emphasis Areas: %s\n", strings.Join(config.Emphasis, ", ")))
	}
	sb.WriteString("\n")

	// Add profile data
	sb.WriteString("PROFILE DATA:\n")
	if viewData.Profile != nil {
		if name, ok := viewData.Profile["name"].(string); ok && name != "" {
			sb.WriteString(fmt.Sprintf("Name: %s\n", name))
		}
		if headline, ok := viewData.Profile["headline"].(string); ok && headline != "" {
			sb.WriteString(fmt.Sprintf("Title/Headline: %s\n", headline))
		}
		if location, ok := viewData.Profile["location"].(string); ok && location != "" {
			sb.WriteString(fmt.Sprintf("Location: %s\n", location))
		}
		if email, ok := viewData.Profile["contact_email"].(string); ok && email != "" {
			sb.WriteString(fmt.Sprintf("Email: %s\n", email))
		}
		if summary, ok := viewData.Profile["summary"].(string); ok && summary != "" {
			sb.WriteString(fmt.Sprintf("Summary: %s\n", summary))
		}
	}

	// Override headline/summary if view has custom ones
	if viewData.HeroHeadline != "" {
		sb.WriteString(fmt.Sprintf("Custom Headline for this view: %s\n", viewData.HeroHeadline))
	}
	if viewData.HeroSummary != "" {
		sb.WriteString(fmt.Sprintf("Custom Summary for this view: %s\n", viewData.HeroSummary))
	}
	sb.WriteString("\n")

	// Add sections in order
	for _, sectionName := range viewData.SectionOrder {
		items, ok := viewData.Sections[sectionName]
		if !ok || len(items) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("=== %s ===\n", strings.ToUpper(sectionName)))
		for _, item := range items {
			r.writeItem(&sb, sectionName, item)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`
OUTPUT REQUIREMENTS:
1. Generate clean Markdown suitable for PDF conversion via Pandoc
2. Use these sections (in order, skip if no data): Contact Info, Professional Summary, Experience, Education, Skills, Projects, Certifications
3. For Experience: Write strong achievement-focused bullet points (start with action verbs)
4. For Skills: Group by category if categories are provided
5. Keep formatting simple - use headers (#, ##), bullet points (-), and bold (**) only
6. Do NOT include any code blocks, explanations, or meta-commentary
7. Start directly with the person's name as an H1 header

Return ONLY the Markdown content for the resume.`)

	return sb.String()
}

// writeItem writes an item to the prompt based on section type
func (r *ResumeService) writeItem(sb *strings.Builder, section string, item map[string]interface{}) {
	switch section {
	case "experience":
		if title, ok := item["title"].(string); ok {
			sb.WriteString(fmt.Sprintf("- Title: %s\n", title))
		}
		if company, ok := item["company"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Company: %s\n", company))
		}
		if start, ok := item["start_date"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Start: %s\n", start))
		}
		if end, ok := item["end_date"].(string); ok {
			sb.WriteString(fmt.Sprintf("  End: %s\n", end))
		}
		if current, ok := item["is_current"].(bool); ok && current {
			sb.WriteString("  Current: Yes\n")
		}
		if desc, ok := item["description"].(string); ok && desc != "" {
			sb.WriteString(fmt.Sprintf("  Description: %s\n", desc))
		}
		if bullets, ok := item["bullets"].([]interface{}); ok && len(bullets) > 0 {
			sb.WriteString("  Bullets:\n")
			for _, b := range bullets {
				if s, ok := b.(string); ok {
					sb.WriteString(fmt.Sprintf("    - %s\n", s))
				}
			}
		}

	case "education":
		if degree, ok := item["degree"].(string); ok {
			sb.WriteString(fmt.Sprintf("- Degree: %s\n", degree))
		}
		if field, ok := item["field"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Field: %s\n", field))
		}
		if school, ok := item["institution"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Institution: %s\n", school))
		}
		if end, ok := item["end_date"].(string); ok {
			sb.WriteString(fmt.Sprintf("  Graduation: %s\n", end))
		}

	case "skills":
		if name, ok := item["name"].(string); ok {
			sb.WriteString(fmt.Sprintf("- %s", name))
			if cat, ok := item["category"].(string); ok && cat != "" {
				sb.WriteString(fmt.Sprintf(" (Category: %s)", cat))
			}
			if level, ok := item["proficiency"].(string); ok && level != "" {
				sb.WriteString(fmt.Sprintf(" [%s]", level))
			}
			sb.WriteString("\n")
		}

	case "projects":
		if title, ok := item["title"].(string); ok {
			sb.WriteString(fmt.Sprintf("- Project: %s\n", title))
		}
		if summary, ok := item["summary"].(string); ok && summary != "" {
			sb.WriteString(fmt.Sprintf("  Summary: %s\n", summary))
		}
		if tech, ok := item["tech_stack"].([]interface{}); ok && len(tech) > 0 {
			var techs []string
			for _, t := range tech {
				if s, ok := t.(string); ok {
					techs = append(techs, s)
				}
			}
			if len(techs) > 0 {
				sb.WriteString(fmt.Sprintf("  Tech: %s\n", strings.Join(techs, ", ")))
			}
		}

	case "certifications":
		if name, ok := item["name"].(string); ok {
			sb.WriteString(fmt.Sprintf("- %s", name))
			if issuer, ok := item["issuer"].(string); ok && issuer != "" {
				sb.WriteString(fmt.Sprintf(" by %s", issuer))
			}
			if date, ok := item["issue_date"].(string); ok && date != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", date))
			}
			sb.WriteString("\n")
		}

	default:
		// Generic handling for other sections
		for k, v := range item {
			if s, ok := v.(string); ok && s != "" && k != "id" && k != "visibility" && k != "is_draft" {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", k, s))
			}
		}
	}
}

// cleanMarkdown removes code blocks and other artifacts from AI response
func (r *ResumeService) cleanMarkdown(markdown string) string {
	// Remove markdown code blocks if the AI wrapped the response
	markdown = strings.TrimSpace(markdown)

	// Remove ```markdown or ``` wrapper
	if strings.HasPrefix(markdown, "```markdown") {
		markdown = strings.TrimPrefix(markdown, "```markdown")
		if idx := strings.LastIndex(markdown, "```"); idx != -1 {
			markdown = markdown[:idx]
		}
	} else if strings.HasPrefix(markdown, "```") {
		markdown = strings.TrimPrefix(markdown, "```")
		if idx := strings.LastIndex(markdown, "```"); idx != -1 {
			markdown = markdown[:idx]
		}
	}

	return strings.TrimSpace(markdown)
}

// CheckPandocAvailable checks if Pandoc is installed
func (r *ResumeService) CheckPandocAvailable() bool {
	_, err := exec.LookPath("pandoc")
	return err == nil
}

// convertToPDF converts markdown to PDF using Pandoc
func (r *ResumeService) convertToPDF(markdown string) ([]byte, error) {
	return r.runPandoc(markdown, "pdf")
}

// convertToDOCX converts markdown to DOCX using Pandoc
func (r *ResumeService) convertToDOCX(markdown string) ([]byte, error) {
	return r.runPandoc(markdown, "docx")
}

// runPandoc executes Pandoc to convert markdown to the target format
func (r *ResumeService) runPandoc(markdown string, format string) ([]byte, error) {
	// Check if Pandoc is available
	if !r.CheckPandocAvailable() {
		log.Printf("[AI-PRINT] Pandoc not found in PATH")
		return nil, fmt.Errorf("Pandoc is not installed. PDF/DOCX generation requires Pandoc.")
	}

	// Create temp input file
	tmpIn, err := os.CreateTemp("", "resume-*.md")
	if err != nil {
		log.Printf("[AI-PRINT] Failed to create temp input file: %v", err)
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err := tmpIn.WriteString(markdown); err != nil {
		log.Printf("[AI-PRINT] Failed to write markdown to temp file: %v", err)
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpIn.Close()

	// Determine output file extension
	ext := format
	if format == "pdf" {
		ext = "pdf"
	} else if format == "docx" {
		ext = "docx"
	}

	// Create temp output file path
	tmpOut := tmpIn.Name() + "." + ext
	defer os.Remove(tmpOut)

	// Build Pandoc command
	args := []string{
		tmpIn.Name(),
		"-o", tmpOut,
		"-V", "geometry:margin=0.75in",
		"-V", "fontsize=11pt",
		"--standalone",
	}

	// Prefer Helvetica for PDF output; try xelatex first (for font support),
	// then fall back to pdflatex if unavailable.
	if format == "pdf" {
		args = append(args,
			"-V", "mainfont=TeX Gyre Heros",
			"-V", "mainfontfallback=Arial",
			"--pdf-engine=xelatex",
		)
	}

	log.Printf("[AI-PRINT] Running Pandoc with args: %v", args)

	cmd := exec.Command("pandoc", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		log.Printf("[AI-PRINT] Pandoc failed: %v, stderr: %s", err, errMsg)

		// If xelatex or font selection failed, fall back to pdflatex with helvet.
		if format == "pdf" && (strings.Contains(errMsg, "xelatex") || strings.Contains(errMsg, "xetex") || strings.Contains(errMsg, "fontspec")) {
			log.Printf("[AI-PRINT] xelatex/font selection failed, trying pdflatex fallback")

			// Remove pdf-engine and mainfont flags
			filtered := make([]string, 0, len(args))
			for i := 0; i < len(args); i++ {
				a := args[i]
				if strings.HasPrefix(a, "--pdf-engine=") {
					continue
				}
				if a == "-V" && i+1 < len(args) {
					val := args[i+1]
					if strings.HasPrefix(val, "mainfont=") || strings.HasPrefix(val, "mainfontfallback=") {
						i++ // skip the value as well
						continue
					}
				}
				filtered = append(filtered, a)
			}

			args = append(filtered,
				"--pdf-engine=pdflatex",
				"-V", "fontfamily=helvet",
				"-V", "fontfamilyoptions=scaled=0.95",
			)

			cmd = exec.Command("pandoc", args...)
			stderr.Reset()
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				log.Printf("[AI-PRINT] Pandoc retry failed: %v, stderr: %s", err, stderr.String())
				return nil, r.formatPandocError(stderr.String())
			}
		} else {
			return nil, r.formatPandocError(errMsg)
		}
	}

	log.Printf("[AI-PRINT] Pandoc succeeded, reading output file: %s", tmpOut)

	// Read the output file
	output, err := os.ReadFile(tmpOut)
	if err != nil {
		log.Printf("[AI-PRINT] Failed to read output file: %v", err)
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	log.Printf("[AI-PRINT] Output file size: %d bytes", len(output))
	return output, nil
}

// formatPandocError converts raw Pandoc/LaTeX errors into user-friendly messages
func (r *ResumeService) formatPandocError(errMsg string) error {
	// Check for missing LaTeX packages
	if strings.Contains(errMsg, ".sty' not found") || strings.Contains(errMsg, "File `") && strings.Contains(errMsg, "not found") {
		// Extract the package name if possible
		var pkgName string
		if idx := strings.Index(errMsg, "File `"); idx != -1 {
			end := strings.Index(errMsg[idx:], "'")
			if end > 6 {
				pkgName = errMsg[idx+6 : idx+end]
			}
		}

		if pkgName != "" {
			return fmt.Errorf("PDF generation requires LaTeX package '%s' which is not installed. Please rebuild your container or try DOCX format instead.", pkgName)
		}
		return fmt.Errorf("PDF generation requires additional LaTeX packages. Please rebuild your container or try DOCX format instead.")
	}

	// Check for missing pdflatex
	if strings.Contains(errMsg, "pdflatex not found") || strings.Contains(errMsg, "pdflatex: not found") {
		return fmt.Errorf("PDF generation requires pdflatex (LaTeX) which is not installed. Please rebuild your container or try DOCX format instead.")
	}

	// Check for font errors
	if strings.Contains(errMsg, "Font") && (strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "error")) {
		return fmt.Errorf("PDF generation encountered a font error. Please try DOCX format or rebuild your container with full LaTeX font support.")
	}

	// Generic error with helpful suggestion
	if strings.Contains(errMsg, "Error producing PDF") {
		return fmt.Errorf("PDF generation failed. Try DOCX format instead, or check server logs for details.")
	}

	// Return the original message for unknown errors
	return fmt.Errorf("Document conversion failed: %s", errMsg)
}
