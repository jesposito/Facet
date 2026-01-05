#!/bin/bash
# Test script for resume upload feature

echo "🧪 Testing Resume Upload Feature"
echo "================================="
echo ""

# Check if backend is running
if ! curl -s http://localhost:8090/api/health > /dev/null 2>&1; then
    echo "❌ Backend is not running on :8090"
    echo ""
    echo "Start it with:"
    echo "  cd backend && air"
    echo ""
    exit 1
fi

echo "✅ Backend is running"
echo ""

# Check for test resume file
RESUME_FILE="${1:-test-resume.pdf}"

if [ ! -f "$RESUME_FILE" ]; then
    echo "❌ Resume file not found: $RESUME_FILE"
    echo ""
    echo "Usage: $0 <path-to-resume.pdf|docx>"
    echo ""
    echo "Example:"
    echo "  $0 ~/Downloads/my-resume.pdf"
    echo ""
    exit 1
fi

echo "📄 Using resume file: $RESUME_FILE"
echo ""

# Get auth token (you'll need to login first)
echo "🔐 Authentication"
echo "You need to be logged in to upload a resume."
echo ""
echo "Login at: http://localhost:5173/admin/login"
echo ""
read -p "Enter your PocketBase auth token (from browser DevTools): " AUTH_TOKEN

if [ -z "$AUTH_TOKEN" ]; then
    echo "❌ Auth token is required"
    exit 1
fi

echo ""
echo "🚀 Uploading resume..."
echo ""

# Upload resume
RESPONSE=$(curl -X POST http://localhost:8090/api/resume/upload \
  -H "Authorization: Bearer $AUTH_TOKEN" \
  -F "file=@$RESUME_FILE" \
  -F "provider_id=" \
  -w "\n%{http_code}" \
  -s)

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP Status: $HTTP_CODE"
echo ""

if [ "$HTTP_CODE" == "200" ]; then
    echo "✅ SUCCESS!"
    echo ""
    echo "Response:"
    echo "$BODY" | jq '.'
    echo ""
    echo "🎉 Resume imported! Check the data:"
    echo "  - Experience: $(echo "$BODY" | jq -r '.counts.experience // 0') items"
    echo "  - Education: $(echo "$BODY" | jq -r '.counts.education // 0') items"
    echo "  - Skills: $(echo "$BODY" | jq -r '.counts.skills // 0') items"
    echo "  - Certifications: $(echo "$BODY" | jq -r '.counts.certifications // 0') items"
    echo "  - Projects: $(echo "$BODY" | jq -r '.counts.projects // 0') items"
    echo ""
    echo "All items are private by default. Review them in the admin panel!"
else
    echo "❌ FAILED"
    echo ""
    echo "Response:"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
fi
