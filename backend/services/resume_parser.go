package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fumiama/go-docx"
	"github.com/gen2brain/go-fitz"
)

// ResumeParser handles extraction of text from resume files
type ResumeParser struct {
	ai *AIService
}

// NewResumeParser creates a new resume parser instance
func NewResumeParser(ai *AIService) *ResumeParser {
	return &ResumeParser{ai: ai}
}

// ParsedResume represents the structured data extracted from a resume
type ParsedResume struct {
	Profile        ProfileData         `json:"profile"`
	Experience     []ExperienceData    `json:"experience"`
	Education      []EducationData     `json:"education"`
	Skills         []SkillData         `json:"skills"`
	Certifications []CertificationData `json:"certifications"`
	Projects       []ProjectData       `json:"projects"`
	Awards         []AwardData         `json:"awards"`
	Talks          []TalkData          `json:"talks"`
	Metadata       MetadataData        `json:"metadata"`
}

type ProfileData struct {
	Name         string `json:"name"`
	Headline     string `json:"headline"`
	Location     string `json:"location"`
	Summary      string `json:"summary"`
	ContactEmail string `json:"contact_email"`
}

type ExperienceData struct {
	Company     string   `json:"company"`
	Title       string   `json:"title"`
	Location    string   `json:"location"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"` // null if current
	Description string   `json:"description"`
	Bullets     []string `json:"bullets"`
	Skills      []string `json:"skills"`
}

type EducationData struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Field       string `json:"field"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Description string `json:"description"`
}

type SkillData struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Proficiency string `json:"proficiency"` // expert, proficient, familiar
}

type CertificationData struct {
	Name          string `json:"name"`
	Issuer        string `json:"issuer"`
	IssueDate     string `json:"issue_date"`
	ExpiryDate    string `json:"expiry_date"` // null if no expiry
	CredentialID  string `json:"credential_id"`
	CredentialURL string `json:"credential_url"`
}

type ProjectData struct {
	Title       string              `json:"title"`
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	TechStack   []string            `json:"tech_stack"`
	Links       []map[string]string `json:"links"` // [{"type": "github", "url": "..."}]
}

type AwardData struct {
	Title       string `json:"title"`
	Issuer      string `json:"issuer"`
	AwardedAt   string `json:"awarded_at"`
	Description string `json:"description"`
}

type TalkData struct {
	Title       string `json:"title"`
	Event       string `json:"event"`
	Date        string `json:"date"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

type MetadataData struct {
	Confidence string   `json:"confidence"` // high, medium, low
	Warnings   []string `json:"warnings"`
	Notes      string   `json:"notes"`
}

// ExtractText extracts text from PDF or DOCX file
func (rp *ResumeParser) ExtractText(fileBytes []byte, mimeType string) (string, error) {
	switch mimeType {
	case "application/pdf":
		return rp.extractPDF(fileBytes)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return rp.extractDOCX(fileBytes)
	default:
		return "", fmt.Errorf("unsupported file type: %s", mimeType)
	}
}

// extractPDF extracts text from PDF using go-fitz
func (rp *ResumeParser) extractPDF(fileBytes []byte) (string, error) {
	doc, err := fitz.NewFromMemory(fileBytes)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var textBuilder strings.Builder
	n := doc.NumPage()

	for i := 0; i < n; i++ {
		text, err := doc.Text(i)
		if err != nil {
			continue // Skip pages with errors
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n") // Page separator
	}

	extractedText := textBuilder.String()
	if strings.TrimSpace(extractedText) == "" {
		return "", fmt.Errorf("no text found in PDF - it may be a scanned image")
	}

	return extractedText, nil
}

// extractDOCX extracts text from DOCX using go-docx
func (rp *ResumeParser) extractDOCX(fileBytes []byte) (string, error) {
	reader := bytes.NewReader(fileBytes)
	doc, err := docx.Parse(reader, int64(len(fileBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to parse DOCX: %w", err)
	}

	var textBuilder strings.Builder

	// Iterate through document body to extract text from paragraphs
	// (Tables are usually for formatting in resumes, text is in paragraphs)
	for _, item := range doc.Document.Body.Items {
		if para, ok := item.(*docx.Paragraph); ok {
			text := para.String()
			if strings.TrimSpace(text) != "" {
				textBuilder.WriteString(text)
				textBuilder.WriteString("\n")
			}
		}
	}

	extractedText := textBuilder.String()
	if strings.TrimSpace(extractedText) == "" {
		return "", fmt.Errorf("no text found in DOCX")
	}

	return extractedText, nil
}

// ParseResume uses AI to extract structured data from resume text
func (rp *ResumeParser) ParseResume(ctx context.Context, provider *AIProvider, resumeText string) (*ParsedResume, error) {
	prompt := rp.buildParsingPrompt(resumeText)

	// Call AI provider with parsing prompt
	response, err := rp.ai.ImproveContent(ctx, provider, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI parsing failed: %w", err)
	}

	// Parse JSON response
	var parsed ParsedResume
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		// Try to extract JSON from markdown code block if present
		cleaned := extractJSONFromResponse(response)
		if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
		}
	}

	// Validate and enrich
	rp.validateAndEnrich(&parsed)

	return &parsed, nil
}

// buildParsingPrompt creates the AI prompt for resume parsing
func (rp *ResumeParser) buildParsingPrompt(resumeText string) string {
	return fmt.Sprintf(`You are an expert resume parser. Extract structured data from the resume text below.

Resume text:
"""
%s
"""

Extract the following sections and return ONLY valid JSON (no explanations, no markdown):

{
  "profile": {
    "name": "Full Name",
    "headline": "Professional title or headline",
    "location": "City, State/Country",
    "summary": "Professional summary or objective (2-3 sentences)",
    "contact_email": "email@example.com"
  },
  "experience": [
    {
      "company": "Company Name",
      "title": "Job Title",
      "location": "City, State",
      "start_date": "YYYY-MM",
      "end_date": "YYYY-MM" or null if current,
      "description": "Brief role description",
      "bullets": ["Achievement 1", "Achievement 2", "Achievement 3"],
      "skills": ["Skill1", "Skill2"]
    }
  ],
  "education": [
    {
      "institution": "University/School Name",
      "degree": "Degree Type (BS, MS, PhD, etc.)",
      "field": "Field of Study",
      "start_date": "YYYY-MM",
      "end_date": "YYYY-MM",
      "description": "Honors, GPA, relevant activities"
    }
  ],
  "skills": [
    {
      "name": "Skill Name",
      "category": "Programming|Tools|Languages|Soft Skills|etc",
      "proficiency": "expert|proficient|familiar"
    }
  ],
  "certifications": [
    {
      "name": "Certification Name",
      "issuer": "Issuing Organization",
      "issue_date": "YYYY-MM",
      "expiry_date": "YYYY-MM" or null if no expiry,
      "credential_id": "ID if present",
      "credential_url": "URL if present"
    }
  ],
  "projects": [
    {
      "title": "Project Name",
      "summary": "One-sentence summary",
      "description": "Detailed description (2-3 sentences)",
      "tech_stack": ["Technology1", "Technology2"],
      "links": [{"type": "github|demo|website", "url": "https://..."}]
    }
  ],
  "awards": [
    {
      "title": "Award Name",
      "issuer": "Issuing Organization",
      "awarded_at": "YYYY-MM",
      "description": "Why awarded or significance"
    }
  ],
  "talks": [
    {
      "title": "Talk/Presentation Title",
      "event": "Event or Conference Name",
      "date": "YYYY-MM",
      "location": "City, State",
      "description": "Talk description"
    }
  ],
  "metadata": {
    "confidence": "high|medium|low",
    "warnings": ["Warning 1", "Warning 2"],
    "notes": "Any parsing notes for the user"
  }
}

**IMPORTANT PARSING RULES**:
1. Extract dates in YYYY-MM format (e.g., "2024-01"). If only year is given, use "YYYY-01".
2. If month is ambiguous or not specified, default to "-01" (January).
3. Use null for end_date if the position is current (indicated by "Present", "Current", "Now", etc.).
4. Preserve bullet points as separate array items, not as a single string.
5. Categorize skills logically (Programming, Tools, Soft Skills, Languages, etc.).
6. Split freelance/contract work with multiple clients into separate experience items.
7. If confidence is low for any section, note it in metadata.warnings.
8. For ambiguous proficiency levels, default to "proficient".
9. **DO NOT hallucinate data** - if information isn't clearly present, omit that field entirely.
10. If multiple similar skills exist (e.g., "JavaScript" and "JS"), merge them and add a warning.
11. Extract email from contact info, signatures, or headers.
12. Professional summary should be concise (2-3 sentences max).

Return ONLY the JSON object, no markdown formatting, no explanations.`, resumeText)
}

// validateAndEnrich validates parsed data and adds helpful warnings
func (rp *ResumeParser) validateAndEnrich(parsed *ParsedResume) {
	if parsed.Metadata.Warnings == nil {
		parsed.Metadata.Warnings = []string{}
	}

	// Validate profile
	if parsed.Profile.Name == "" {
		parsed.Metadata.Warnings = append(parsed.Metadata.Warnings, "No name found in resume")
		parsed.Metadata.Confidence = "low"
	}

	// Check for potential duplicates in skills
	skillNames := make(map[string]bool)
	for _, skill := range parsed.Skills {
		lower := strings.ToLower(skill.Name)
		if skillNames[lower] {
			parsed.Metadata.Warnings = append(parsed.Metadata.Warnings,
				fmt.Sprintf("Duplicate skill detected: %s", skill.Name))
		}
		skillNames[lower] = true
	}

	// Warn about freelance/consulting roles
	for i, exp := range parsed.Experience {
		if strings.Contains(strings.ToLower(exp.Company), "freelance") ||
			strings.Contains(strings.ToLower(exp.Company), "consulting") ||
			strings.Contains(strings.ToLower(exp.Title), "consultant") {
			parsed.Metadata.Warnings = append(parsed.Metadata.Warnings,
				fmt.Sprintf("Experience #%d (%s): Consider splitting if multiple clients were involved", i+1, exp.Company))
		}
	}

	// Set default confidence if not set
	if parsed.Metadata.Confidence == "" {
		if len(parsed.Metadata.Warnings) == 0 {
			parsed.Metadata.Confidence = "high"
		} else if len(parsed.Metadata.Warnings) <= 2 {
			parsed.Metadata.Confidence = "medium"
		} else {
			parsed.Metadata.Confidence = "low"
		}
	}
}

// extractJSONFromResponse extracts JSON from markdown code blocks
func extractJSONFromResponse(response string) string {
	// Remove markdown code fences if present
	response = strings.TrimSpace(response)

	// Check for ```json ... ``` blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	}

	return strings.TrimSpace(response)
}

// Helper to read file bytes from io.Reader
func ReadFileBytes(file io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
