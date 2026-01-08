package moderation

// MinNSFWThreshold is the minimum score to consider content as non-safe.
const MinNSFWThreshold = 0.3

// NSFWDetails contains provider-specific NSFW detection details.
// This is a value object that aggregates all detection scores and metadata.
type NSFWDetails struct {
	// NudityScore is the probability of nudity content (0.0 - 1.0).
	NudityScore float64

	// WeaponScore is the probability of weapons in the image (0.0 - 1.0).
	WeaponScore float64

	// ViolenceScore is the probability of violent content (0.0 - 1.0).
	ViolenceScore float64

	// OffensiveScore is the probability of offensive content (0.0 - 1.0).
	OffensiveScore float64

	// DrugScore is the probability of drug-related content (0.0 - 1.0).
	DrugScore float64

	// SubCategories contains additional detail subcategories detected.
	SubCategories []string

	// RawResponse stores the original API response (for debugging).
	// This should not contain any PII or be logged in production.
	RawResponse map[string]interface{}
}

// NewNSFWDetails creates a new NSFWDetails with the provided scores.
func NewNSFWDetails(nudity, weapon, violence, offensive, drug float64) NSFWDetails {
	return NSFWDetails{
		NudityScore:    clampScore(nudity),
		WeaponScore:    clampScore(weapon),
		ViolenceScore:  clampScore(violence),
		OffensiveScore: clampScore(offensive),
		DrugScore:      clampScore(drug),
		SubCategories:  []string{},
		RawResponse:    nil,
	}
}

// WithSubCategories returns a copy with the specified subcategories.
func (d NSFWDetails) WithSubCategories(categories []string) NSFWDetails {
	d.SubCategories = make([]string, len(categories))
	copy(d.SubCategories, categories)
	return d
}

// WithRawResponse returns a copy with the raw API response.
func (d NSFWDetails) WithRawResponse(raw map[string]interface{}) NSFWDetails {
	d.RawResponse = raw
	return d
}

// MaxScore returns the highest score among all NSFW categories.
func (d NSFWDetails) MaxScore() float64 {
	highest := d.NudityScore
	if d.WeaponScore > highest {
		highest = d.WeaponScore
	}
	if d.ViolenceScore > highest {
		highest = d.ViolenceScore
	}
	if d.OffensiveScore > highest {
		highest = d.OffensiveScore
	}
	if d.DrugScore > highest {
		highest = d.DrugScore
	}
	return highest
}

// DominantCategory returns the category with the highest score.
func (d NSFWDetails) DominantCategory() NSFWCategory {
	topScore := 0.0
	dominant := CategorySafe

	if d.NudityScore > topScore {
		topScore = d.NudityScore
		dominant = CategoryNudity
	}
	if d.OffensiveScore > topScore {
		topScore = d.OffensiveScore
		dominant = CategoryExplicit
	}
	if d.ViolenceScore > topScore || d.WeaponScore > topScore {
		topScore = maxFloat64(d.ViolenceScore, d.WeaponScore)
		dominant = CategoryViolence
	}

	// Only return non-safe if score is significant
	if topScore < MinNSFWThreshold {
		return CategorySafe
	}

	return dominant
}

// HasSubCategory checks if a specific subcategory was detected.
func (d NSFWDetails) HasSubCategory(sub string) bool {
	for _, s := range d.SubCategories {
		if s == sub {
			return true
		}
	}
	return false
}

// IsEmpty returns true if all scores are zero.
func (d NSFWDetails) IsEmpty() bool {
	return d.NudityScore == 0 &&
		d.WeaponScore == 0 &&
		d.ViolenceScore == 0 &&
		d.OffensiveScore == 0 &&
		d.DrugScore == 0
}

// clampScore ensures a score is within the valid range [0.0, 1.0].
func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// maxFloat64 returns the larger of two float64 values.
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
