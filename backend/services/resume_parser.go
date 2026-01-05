package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
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

// extractDOCX extracts text from DOCX using go-docx with XML fallback
func (rp *ResumeParser) extractDOCX(fileBytes []byte) (text string, err error) {
	// Recover from panics in go-docx library (can happen with malformed/complex DOCX files)
	defer func() {
		if r := recover(); r != nil {
			// If go-docx panics, try the XML fallback
			text, err = rp.extractDOCXFromXML(fileBytes)
			if err != nil {
				err = fmt.Errorf("failed to parse DOCX: the file may be corrupted or use unsupported formatting. Error: %v", r)
			}
		}
	}()

	reader := bytes.NewReader(fileBytes)
	doc, parseErr := docx.Parse(reader, int64(len(fileBytes)))
	if parseErr != nil {
		// Try XML fallback before giving up
		text, fallbackErr := rp.extractDOCXFromXML(fileBytes)
		if fallbackErr == nil {
			return text, nil
		}
		return "", fmt.Errorf("failed to parse DOCX: %w. Try converting to PDF or re-saving the file", parseErr)
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
		return "", fmt.Errorf("no text found in DOCX - the file may be empty or contain only images/tables")
	}

	return extractedText, nil
}

// extractDOCXFromXML extracts text from DOCX by parsing the XML directly
// DOCX files are ZIP archives containing XML. This is a fallback when go-docx fails.
func (rp *ResumeParser) extractDOCXFromXML(fileBytes []byte) (string, error) {
	// Open DOCX as ZIP archive
	reader := bytes.NewReader(fileBytes)
	zipReader, err := zip.NewReader(reader, int64(len(fileBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to read DOCX as ZIP: %w", err)
	}

	// Find and read word/document.xml
	var documentXML []byte
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open document.xml: %w", err)
			}
			defer rc.Close()

			documentXML, err = io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("failed to read document.xml: %w", err)
			}
			break
		}
	}

	if documentXML == nil {
		return "", fmt.Errorf("document.xml not found in DOCX file")
	}

	// Parse XML and extract text from <w:t> elements
	type Text struct {
		Value string `xml:",chardata"`
	}
	type Document struct {
		Texts []Text `xml:"body>p>r>t"`
	}

	// Use a simpler approach: extract all <w:t> tags with regex
	// This is more robust than full XML parsing for our use case
	var textBuilder strings.Builder
	decoder := xml.NewDecoder(bytes.NewReader(documentXML))

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "t" {
				// Read the text content
				var text string
				if err := decoder.DecodeElement(&text, &elem); err == nil {
					textBuilder.WriteString(text)
				}
			} else if elem.Name.Local == "p" {
				// Add newline for paragraphs
				textBuilder.WriteString("\n")
			}
		}
	}

	extractedText := textBuilder.String()
	if strings.TrimSpace(extractedText) == "" {
		return "", fmt.Errorf("no text found in DOCX XML")
	}

	return extractedText, nil
}

// ParseResume uses AI to extract structured data from resume text
func (rp *ResumeParser) ParseResume(ctx context.Context, provider *AIProvider, resumeText string) (*ParsedResume, error) {
	// Limit resume text to prevent token overflow (roughly 4000 words = 5000 tokens)
	const maxChars = 16000
	truncated := false
	if len(resumeText) > maxChars {
		resumeText = resumeText[:maxChars]
		truncated = true
	}

	prompt := rp.buildParsingPrompt(resumeText)

	// Call AI provider with parsing prompt
	response, err := rp.ai.ImproveContent(ctx, provider, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI parsing failed: %w", err)
	}

	// Try to extract and parse JSON from response
	cleaned := extractJSONFromResponse(response)

	var parsed ParsedResume
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		// If JSON parsing fails, try to fix common issues
		fixed, fixErr := fixTruncatedJSON(cleaned)
		if fixErr == nil {
			if err := json.Unmarshal([]byte(fixed), &parsed); err == nil {
				// Success after fix - add warning
				if parsed.Metadata.Warnings == nil {
					parsed.Metadata.Warnings = []string{}
				}
				parsed.Metadata.Warnings = append(parsed.Metadata.Warnings,
					"AI response was incomplete - some data may be missing")
				rp.validateAndEnrich(&parsed)
				return &parsed, nil
			}
		}

		// Still failed - return detailed error
		preview := cleaned
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w. Response preview: %s", err, preview)
	}

	// Add warning if resume was truncated
	if truncated {
		if parsed.Metadata.Warnings == nil {
			parsed.Metadata.Warnings = []string{}
		}
		parsed.Metadata.Warnings = append(parsed.Metadata.Warnings,
			"Resume text was very long and had to be truncated - some information may be missing")
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

// fixTruncatedJSON attempts to fix truncated JSON by closing unclosed structures
func fixTruncatedJSON(jsonStr string) (string, error) {
	jsonStr = strings.TrimSpace(jsonStr)

	// Count opening and closing braces/brackets
	openBraces := strings.Count(jsonStr, "{")
	closeBraces := strings.Count(jsonStr, "}")
	openBrackets := strings.Count(jsonStr, "[")
	closeBrackets := strings.Count(jsonStr, "]")

	// If JSON looks complete, return as-is
	if openBraces == closeBraces && openBrackets == closeBrackets {
		return jsonStr, nil
	}

	// Remove trailing incomplete data (incomplete string, number, etc.)
	jsonStr = removeTrailingIncomplete(jsonStr)

	// Close unclosed arrays
	for i := 0; i < openBrackets-closeBrackets; i++ {
		jsonStr += "]"
	}

	// Close unclosed objects
	for i := 0; i < openBraces-closeBraces; i++ {
		jsonStr += "}"
	}

	return jsonStr, nil
}

// removeTrailingIncomplete removes trailing incomplete JSON elements
func removeTrailingIncomplete(jsonStr string) string {
	jsonStr = strings.TrimSpace(jsonStr)

	// If ends with comma, remove it
	if strings.HasSuffix(jsonStr, ",") {
		jsonStr = strings.TrimSuffix(jsonStr, ",")
	}

	// If has unclosed string at the end, remove it
	// Look for the last complete field
	lastValidPos := len(jsonStr)
	inString := false
	escaped := false

	// Scan backwards to find last complete position
	for i := len(jsonStr) - 1; i >= 0; i-- {
		c := jsonStr[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			if !inString {
				// Found end of a complete string
				// Check if this is likely a complete field
				if i < len(jsonStr)-1 {
					next := jsonStr[i+1:]
					if strings.HasPrefix(strings.TrimSpace(next), ",") ||
						strings.HasPrefix(strings.TrimSpace(next), "}") ||
						strings.HasPrefix(strings.TrimSpace(next), "]") {
						// This looks complete
						break
					}
				}
			}
		}

		// If we find a comma or closing brace/bracket outside a string, that's a safe point
		if !inString && (c == ',' || c == '}' || c == ']') {
			lastValidPos = i + 1
			break
		}
	}

	if lastValidPos < len(jsonStr) {
		jsonStr = jsonStr[:lastValidPos]
	}

	return strings.TrimSpace(jsonStr)
}

// Helper to read file bytes from io.Reader
func ReadFileBytes(file io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
