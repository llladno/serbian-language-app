package store

// HasLearningActivity reports whether the learner has ever opened a lesson or
// answered an exercise. An account that has done neither is one the app sends
// straight into the first lesson when it opens.
func (s *Store) HasLearningActivity(userID string) (bool, error) {
	var has bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM lesson_progress WHERE user_id = ?)
		     OR EXISTS(SELECT 1 FROM attempts WHERE user_id = ?)`,
		userID, userID).Scan(&has)
	return has, err
}
