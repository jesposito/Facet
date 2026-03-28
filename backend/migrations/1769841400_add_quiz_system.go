package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds quiz system with 3 collections: quizzes, quiz_questions, quiz_attempts.
// Also adds "quiz" to lessons content_type enum and a quiz relation field to lessons.
func init() {
	m.Register(func(app core.App) error {
		// 1. Create quizzes collection
		if _, err := app.FindCollectionByNameOrId("quizzes"); err == nil {
			return nil // Already exists
		}

		// Find lessons collection for relation
		lessonsCollection, err := app.FindCollectionByNameOrId("lessons")
		if err != nil {
			return err
		}

		quizzes := core.NewBaseCollection("quizzes")
		// No public access - admin only via API hooks
		quizzes.ListRule = nil
		quizzes.ViewRule = nil

		quizzes.Fields.Add(&core.RelationField{
			Name:          "lesson",
			MaxSelect:     1,
			Required:      true,
			CollectionId:  lessonsCollection.Id,
			CascadeDelete: true,
		})
		quizzes.Fields.Add(&core.TextField{
			Name:     "title",
			Required: true,
			Max:      200,
		})
		quizzes.Fields.Add(&core.NumberField{
			Name: "passing_score",
			Min:  floatPtr(0),
			Max:  floatPtr(100),
		})
		quizzes.Fields.Add(&core.BoolField{
			Name: "randomize_questions",
		})
		quizzes.Fields.Add(&core.NumberField{
			Name: "max_attempts",
			Min:  floatPtr(0),
		})

		quizzes.Indexes = []string{
			"CREATE UNIQUE INDEX idx_quizzes_lesson ON quizzes(lesson)",
		}

		if err := app.Save(quizzes); err != nil {
			return err
		}

		// 2. Create quiz_questions collection
		quizQuestions := core.NewBaseCollection("quiz_questions")
		quizQuestions.ListRule = nil
		quizQuestions.ViewRule = nil

		quizQuestions.Fields.Add(&core.RelationField{
			Name:          "quiz",
			MaxSelect:     1,
			Required:      true,
			CollectionId:  quizzes.Id,
			CascadeDelete: true,
		})
		quizQuestions.Fields.Add(&core.TextField{
			Name:     "question_text",
			Required: true,
			Max:      2000,
		})
		quizQuestions.Fields.Add(&core.SelectField{
			Name:      "question_type",
			MaxSelect: 1,
			Values:    []string{"multiple_choice", "true_false"},
			Required:  true,
		})
		quizQuestions.Fields.Add(&core.JSONField{
			Name: "options",
		})
		quizQuestions.Fields.Add(&core.TextField{
			Name: "explanation",
			Max:  2000,
		})
		quizQuestions.Fields.Add(&core.NumberField{
			Name: "points",
			Min:  floatPtr(1),
		})
		quizQuestions.Fields.Add(&core.NumberField{
			Name:     "sort_order",
			Required: true,
			Min:      floatPtr(0),
		})

		quizQuestions.Indexes = []string{
			"CREATE INDEX idx_quiz_questions_quiz_sort ON quiz_questions(quiz, sort_order)",
		}

		if err := app.Save(quizQuestions); err != nil {
			return err
		}

		// 3. Create quiz_attempts collection
		quizAttempts := core.NewBaseCollection("quiz_attempts")
		quizAttempts.ListRule = nil
		quizAttempts.ViewRule = nil

		quizAttempts.Fields.Add(&core.RelationField{
			Name:         "quiz",
			MaxSelect:    1,
			Required:     true,
			CollectionId: quizzes.Id,
		})
		quizAttempts.Fields.Add(&core.EmailField{
			Name:     "buyer_email",
			Required: true,
		})
		quizAttempts.Fields.Add(&core.NumberField{
			Name: "score",
			Min:  floatPtr(0),
			Max:  floatPtr(100),
		})
		quizAttempts.Fields.Add(&core.BoolField{
			Name: "passed",
		})
		quizAttempts.Fields.Add(&core.JSONField{
			Name: "answers",
		})
		quizAttempts.Fields.Add(&core.AutodateField{
			Name:     "attempted_at",
			OnCreate: true,
		})

		quizAttempts.Indexes = []string{
			"CREATE INDEX idx_quiz_attempts_quiz_email ON quiz_attempts(quiz, buyer_email)",
		}

		if err := app.Save(quizAttempts); err != nil {
			return err
		}

		// 4. Add "quiz" to lessons content_type enum and add quiz relation field
		contentTypeField := lessonsCollection.Fields.GetByName("content_type")
		if contentTypeField != nil {
			if selectField, ok := contentTypeField.(*core.SelectField); ok {
				hasQuiz := false
				for _, v := range selectField.Values {
					if v == "quiz" {
						hasQuiz = true
						break
					}
				}
				if !hasQuiz {
					selectField.Values = append(selectField.Values, "quiz")
				}
			}
		}

		// Add quiz relation field to lessons
		if lessonsCollection.Fields.GetByName("quiz") == nil {
			lessonsCollection.Fields.Add(&core.RelationField{
				Name:         "quiz",
				MaxSelect:    1,
				Required:     false,
				CollectionId: quizzes.Id,
			})
		}

		if err := app.Save(lessonsCollection); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback: remove quiz field from lessons, restore content_type, delete quiz collections
		if lessonsCollection, err := app.FindCollectionByNameOrId("lessons"); err == nil {
			if quizField := lessonsCollection.Fields.GetByName("quiz"); quizField != nil {
				lessonsCollection.Fields.RemoveById(quizField.GetId())
			}

			if contentTypeField := lessonsCollection.Fields.GetByName("content_type"); contentTypeField != nil {
				if selectField, ok := contentTypeField.(*core.SelectField); ok {
					newValues := make([]string, 0)
					for _, v := range selectField.Values {
						if v != "quiz" {
							newValues = append(newValues, v)
						}
					}
					selectField.Values = newValues
				}
			}

			_ = app.Save(lessonsCollection)
		}

		// Delete collections in reverse order
		collections := []string{"quiz_attempts", "quiz_questions", "quizzes"}
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
