package store

import (
	"testing"
	"time"
)

func TestSurveyOfferIsOncePerLearnerAndNeedsThreeLessons(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	id, err := s.CreateUser("Анна")
	if err != nil {
		t.Fatal(err)
	}
	for i, l := range []string{"00", "01", "02"} {
		if offer, err := s.SurveyOffer(id, now); err != nil || offer {
			t.Fatalf("after %d lessons: offer=%v err=%v", i, offer, err)
		}
		if err := s.User(id).SetLessonStatus(l, "done", now); err != nil {
			t.Fatal(err)
		}
	}
	if offer, err := s.SurveyOffer(id, now); err != nil || !offer {
		t.Fatalf("after 3 lessons: offer=%v err=%v", offer, err)
	}
	if offer, _ := s.SurveyOffer(id, now); offer {
		t.Fatal("offered twice")
	}
}

func TestSaveSurveyPaysOnceAndDismissNeverDowngrades(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	id, _ := s.CreateUser("Анна")
	a := SurveyAnswers{Rating: 5, Likes: []string{"lessons"}, Pains: []string{"nothing"}, Purpose: "move", Wish: "speaking"}
	if err := s.SaveSurvey(id, a, 10, now); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSurvey(id, a, 10, now); err != ErrSurveyClosed {
		t.Fatalf("second save = %v, want ErrSurveyClosed", err)
	}
	if err := s.DismissSurvey(id, now); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := s.db.QueryRow(`SELECT status FROM survey_responses WHERE user_id = ?`, id).Scan(&status); err != nil || status != "answered" {
		t.Fatalf("status = %q, %v; want answered", status, err)
	}
	if bal, _ := s.Balance(id); bal != 10 {
		t.Fatalf("balance = %d, want 10", bal)
	}
}
