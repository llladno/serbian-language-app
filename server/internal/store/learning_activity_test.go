package store

import (
	"testing"
	"time"
)

func TestHasLearningActivity(t *testing.T) {
	s := newStore(t)
	uid, err := s.CreateUser("Гриша")
	if err != nil {
		t.Fatal(err)
	}
	has, err := s.HasLearningActivity(uid)
	if err != nil || has {
		t.Fatalf("new account: has = %v, err = %v; want false", has, err)
	}

	// Opening a lesson counts, even before a single answer.
	other, err := s.CreateUser("Ана")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.User(other).SetLessonStatus("00", "in_progress", time.Now()); err != nil {
		t.Fatal(err)
	}
	if has, _ := s.HasLearningActivity(other); !has {
		t.Fatal("an opened lesson should count as learning activity")
	}
	// ...and it is per account.
	if has, _ := s.HasLearningActivity(uid); has {
		t.Fatal("the other learner's lesson leaked into this account")
	}

	// An answer alone counts too.
	third, err := s.CreateUser("Лука")
	if err != nil {
		t.Fatal(err)
	}
	answer(t, s.User(third), "ex1", true, time.Now())
	if has, _ := s.HasLearningActivity(third); !has {
		t.Fatal("an answered exercise should count as learning activity")
	}
}
