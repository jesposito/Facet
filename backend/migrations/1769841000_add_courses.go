package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds courses feature with 5 collections: courses, modules, lessons, course_progress, lesson_progress.
// Also includes cover_image_library_url, config fields (enable_progress, etc.), and lesson attachments.
func init() {
	m.Register(func(app core.App) error {
		// 1. Create courses collection
		if _, err := app.FindCollectionByNameOrId("courses"); err == nil {
			return nil // Already exists
		}

		publicRule := ""

		courses := core.NewBaseCollection("courses")
		courses.ListRule = &publicRule
		courses.ViewRule = &publicRule

		courses.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      200,
		})
		courses.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
			Max:      200,
		})
		courses.Fields.Add(&core.EditorField{
			Name: "description",
		})
		courses.Fields.Add(&core.TextField{
			Name: "instructor_name",
			Max:  200,
		})
		courses.Fields.Add(&core.EditorField{
			Name: "instructor_bio",
		})
		courses.Fields.Add(&core.FileField{
			Name:      "cover_image",
			MaxSize:   10485760, // 10MB
			MimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
		})
		courses.Fields.Add(&core.TextField{
			Name: "cover_image_library_url",
			Max:  500,
		})
		courses.Fields.Add(&core.SelectField{
			Name:      "access_tier",
			MaxSelect: 1,
			Values:    []string{"free", "paid"},
			Required:  true,
		})
		courses.Fields.Add(&core.NumberField{
			Name: "price",
			Min:  floatPtr(0),
		})
		courses.Fields.Add(&core.BoolField{
			Name: "sequential_access",
		})
		courses.Fields.Add(&core.SelectField{
			Name:      "difficulty",
			MaxSelect: 1,
			Values:    []string{"beginner", "intermediate", "advanced"},
		})
		courses.Fields.Add(&core.NumberField{
			Name: "estimated_hours",
			Min:  floatPtr(0),
		})
		courses.Fields.Add(&core.JSONField{
			Name: "tags",
		})
		courses.Fields.Add(&core.JSONField{
			Name: "learning_outcomes",
		})
		courses.Fields.Add(&core.SelectField{
			Name:      "visibility",
			MaxSelect: 1,
			Values:    []string{"public", "unlisted", "private"},
		})
		courses.Fields.Add(&core.BoolField{
			Name: "is_draft",
		})
		courses.Fields.Add(&core.NumberField{
			Name: "sort_order",
			Min:  floatPtr(0),
		})
		// Config fields
		courses.Fields.Add(&core.BoolField{
			Name: "enable_progress",
		})
		courses.Fields.Add(&core.BoolField{
			Name: "show_progress_bar",
		})
		courses.Fields.Add(&core.BoolField{
			Name: "require_enrollment",
		})
		courses.Fields.Add(&core.BoolField{
			Name: "enable_certificates",
		})

		courses.Indexes = []string{
			"CREATE UNIQUE INDEX idx_courses_slug ON courses(slug) WHERE slug != ''",
		}

		if err := app.Save(courses); err != nil {
			return err
		}

		// 2. Create modules collection
		modules := core.NewBaseCollection("modules")
		modules.ListRule = &publicRule
		modules.ViewRule = &publicRule

		modules.Fields.Add(&core.RelationField{
			Name:         "course",
			MaxSelect:    1,
			Required:     true,
			CollectionId: courses.Id,
		})
		modules.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      200,
		})
		modules.Fields.Add(&core.EditorField{
			Name: "description",
		})
		modules.Fields.Add(&core.NumberField{
			Name:     "sort_order",
			Required: true,
			Min:      floatPtr(0),
		})
		// Drip content fields for modules
		modules.Fields.Add(&core.NumberField{
			Name: "drip_days_after_enrollment",
			Min:  floatPtr(0),
		})
		modules.Fields.Add(&core.DateField{
			Name: "drip_unlock_date",
		})

		modules.Indexes = []string{
			"CREATE INDEX idx_modules_course_sort ON modules(course, sort_order)",
		}

		if err := app.Save(modules); err != nil {
			return err
		}

		// 3. Create lessons collection
		lessons := core.NewBaseCollection("lessons")
		lessons.ListRule = &publicRule
		lessons.ViewRule = &publicRule

		lessons.Fields.Add(&core.RelationField{
			Name:         "module",
			MaxSelect:    1,
			Required:     true,
			CollectionId: modules.Id,
		})
		lessons.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      200,
		})
		lessons.Fields.Add(&core.SelectField{
			Name:      "content_type",
			MaxSelect: 1,
			Values:    []string{"video", "text", "mixed"},
			Required:  true,
		})
		lessons.Fields.Add(&core.EditorField{
			Name: "content",
		})
		lessons.Fields.Add(&core.TextField{
			Name: "video_url",
			Max:  500,
		})
		lessons.Fields.Add(&core.NumberField{
			Name: "estimated_minutes",
			Min:  floatPtr(0),
		})
		lessons.Fields.Add(&core.NumberField{
			Name:     "sort_order",
			Required: true,
			Min:      floatPtr(0),
		})
		lessons.Fields.Add(&core.BoolField{
			Name: "is_preview",
		})
		lessons.Fields.Add(&core.FileField{
			Name:      "attachments",
			MaxSize:   52428800, // 50MB per file
			MaxSelect: 5,
			MimeTypes: []string{
				"application/pdf",
				"application/zip",
				"application/x-zip-compressed",
				"image/jpeg",
				"image/png",
				"image/webp",
				"application/vnd.openxmlformats-officedocument.presentationml.presentation",
				"application/vnd.ms-powerpoint",
				"text/plain",
				"application/octet-stream",
			},
		})

		lessons.Indexes = []string{
			"CREATE INDEX idx_lessons_module_sort ON lessons(module, sort_order)",
		}

		if err := app.Save(lessons); err != nil {
			return err
		}

		// 4. Create course_progress collection
		courseProgress := core.NewBaseCollection("course_progress")
		// No public access - backend only
		courseProgress.ListRule = nil
		courseProgress.ViewRule = nil

		courseProgress.Fields.Add(&core.RelationField{
			Name:         "course",
			MaxSelect:    1,
			Required:     true,
			CollectionId: courses.Id,
		})
		courseProgress.Fields.Add(&core.EmailField{
			Name:     "buyer_email",
			Required: true,
		})
		courseProgress.Fields.Add(&core.NumberField{
			Name: "progress_percent",
			Min:  floatPtr(0),
			Max:  floatPtr(100),
		})
		courseProgress.Fields.Add(&core.NumberField{
			Name: "lessons_completed",
			Min:  floatPtr(0),
		})
		courseProgress.Fields.Add(&core.NumberField{
			Name: "total_lessons",
			Min:  floatPtr(0),
		})
		courseProgress.Fields.Add(&core.DateField{
			Name: "started_at",
		})
		courseProgress.Fields.Add(&core.DateField{
			Name: "completed_at",
		})
		courseProgress.Fields.Add(&core.DateField{
			Name: "last_accessed",
		})
		courseProgress.Fields.Add(&core.TextField{
			Name: "student_name",
			Max:  200,
		})

		courseProgress.Indexes = []string{
			"CREATE UNIQUE INDEX idx_course_progress_unique ON course_progress(course, buyer_email)",
		}

		if err := app.Save(courseProgress); err != nil {
			return err
		}

		// 5. Create lesson_progress collection
		lessonProgress := core.NewBaseCollection("lesson_progress")
		// No public access - backend only
		lessonProgress.ListRule = nil
		lessonProgress.ViewRule = nil

		lessonProgress.Fields.Add(&core.RelationField{
			Name:         "lesson",
			MaxSelect:    1,
			Required:     true,
			CollectionId: lessons.Id,
		})
		lessonProgress.Fields.Add(&core.EmailField{
			Name:     "buyer_email",
			Required: true,
		})
		lessonProgress.Fields.Add(&core.BoolField{
			Name: "completed",
		})
		lessonProgress.Fields.Add(&core.DateField{
			Name: "completed_at",
		})

		lessonProgress.Indexes = []string{
			"CREATE UNIQUE INDEX idx_lesson_progress_unique ON lesson_progress(lesson, buyer_email)",
		}

		return app.Save(lessonProgress)
	}, func(app core.App) error {
		// Rollback: delete collections in reverse order
		collections := []string{"lesson_progress", "course_progress", "lessons", "modules", "courses"}
		for _, name := range collections {
			if collection, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
