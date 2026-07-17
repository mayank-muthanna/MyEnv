package infer

import "github.com/myenv-cli/myenv/internal/schema"

type Change struct {
	Added     int
	Removed   int
	Preserved int
}

// Merge keeps user-configured rules, adds newly inferred rules, and removes rules no longer in dotenv.
func Merge(existing, inferred schema.Schema) (schema.Schema, Change) {
	merged := make(schema.Schema, len(inferred))
	change := Change{}

	for key, rule := range inferred {
		if current, found := existing[key]; found {
			merged[key] = current
			change.Preserved++
			continue
		}
		merged[key] = rule
		change.Added++
	}
	for key := range existing {
		if _, found := inferred[key]; !found {
			change.Removed++
		}
	}
	return merged, change
}
