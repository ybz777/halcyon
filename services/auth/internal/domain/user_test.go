package domain

import (
	"errors"
	"testing"
	"time"
)

var testNow = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

func testUser(t *testing.T) *User {
	t.Helper()

	id, err := ParseUserID("usr_abcdefghijklmnopqrst")
	if err != nil {
		t.Fatalf("ParseUserID: %v", err)
	}

	email, err := ParseEmail("user@example.com")
	if err != nil {
		t.Fatalf("ParseEmail: %v", err)
	}

	hash, err := ParsePasswordHash(validBcryptHash)
	if err != nil {
		t.Fatalf("ParsePasswordHash: %v", err)
	}

	user, err := RegisterUser(id, email, hash, testNow)
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}

	user.ClearUncommittedEvents()
	return user
}

func TestRegisterUser_Success(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_abcdefghijklmnopqrst")
	email, _ := ParseEmail("new@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)

	user, err := RegisterUser(id, email, hash, testNow)
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	if user.Status() != UserStatusActive {
		t.Errorf("Status() = %q, want active", user.Status())
	}
	if user.Version() != 1 {
		t.Errorf("Version() = %d, want 1", user.Version())
	}

	events := user.UncommittedEvents()
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}

	registered, ok := events[0].(UserRegistered)
	if !ok {
		t.Fatalf("event type = %T, want UserRegistered", events[0])
	}
	if !registered.Email.Equal(email) {
		t.Error("registered event email mismatch")
	}
}

func TestRegisterUser_InvalidInput(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_abcdefghijklmnopqrst")
	email, _ := ParseEmail("user@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)

	tests := []struct {
		name string
		id   UserID
		em   Email
		ph   PasswordHash
		now  time.Time
		want error
	}{
		{"zero id", UserID{}, email, hash, testNow, ErrInvalidUserID},
		{"zero email", id, Email{}, hash, testNow, ErrInvalidEmail},
		{"zero hash", id, email, PasswordHash{}, testNow, ErrInvalidPasswordHash},
		{"zero time", id, email, hash, time.Time{}, ErrInvalidTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := RegisterUser(tt.id, tt.em, tt.ph, tt.now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestReconstituteUser_Invalid(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_abcdefghijklmnopqrst")
	email, _ := ParseEmail("user@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)

	tests := []struct {
		name    string
		id      UserID
		email   Email
		hash    PasswordHash
		status  UserStatus
		created time.Time
		updated time.Time
		version int
		want    error
	}{
		{"invalid status", id, email, hash, UserStatus("locked"), testNow, testNow, 1, ErrInvalidUserStatus},
		{"invalid version", id, email, hash, UserStatusActive, testNow, testNow, 0, ErrInvalidVersion},
		{"zero created", id, email, hash, UserStatusActive, time.Time{}, testNow, 1, ErrInvalidTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ReconstituteUser(tt.id, tt.email, tt.hash, tt.status, tt.created, tt.updated, tt.version)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestUser_Enable_Idempotent(t *testing.T) {
	t.Parallel()

	user := testUser(t)
	user.ClearUncommittedEvents()

	if err := user.Enable(testNow.Add(time.Hour)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(user.UncommittedEvents()) != 0 {
		t.Fatal("idempotent enable should not emit events")
	}
}

func TestReconstituteUser_Success(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_abcdefghijklmnopqrst")
	email, _ := ParseEmail("user@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)

	user, err := ReconstituteUser(id, email, hash, UserStatusActive, testNow, testNow, 3)
	if err != nil {
		t.Fatalf("ReconstituteUser() error = %v", err)
	}

	if len(user.UncommittedEvents()) != 0 {
		t.Fatal("reconstituted user should have no uncommitted events")
	}
	if user.Version() != 3 {
		t.Errorf("Version() = %d, want 3", user.Version())
	}
}

func TestUser_ChangeEmail(t *testing.T) {
	t.Parallel()

	user := testUser(t)
	newEmail, _ := ParseEmail("updated@example.com")

	if err := user.ChangeEmail(newEmail, testNow.Add(time.Hour)); err != nil {
		t.Fatalf("ChangeEmail() error = %v", err)
	}

	if !user.Email().Equal(newEmail) {
		t.Errorf("Email() = %q, want %q", user.Email(), newEmail)
	}
	if user.Version() != 2 {
		t.Errorf("Version() = %d, want 2", user.Version())
	}

	events := user.UncommittedEvents()
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].EventType() != "auth.user.email_changed" {
		t.Errorf("event type = %q", events[0].EventType())
	}
}

func TestUser_ChangeEmail_Unchanged(t *testing.T) {
	t.Parallel()

	user := testUser(t)
	err := user.ChangeEmail(user.Email(), testNow.Add(time.Hour))
	if !errors.Is(err, ErrEmailUnchanged) {
		t.Fatalf("error = %v, want ErrEmailUnchanged", err)
	}
}

func TestUser_ChangeEmail_DisabledUser(t *testing.T) {
	t.Parallel()

	user := testUser(t)
	_ = user.Disable(testNow.Add(time.Minute))

	newEmail, _ := ParseEmail("other@example.com")
	err := user.ChangeEmail(newEmail, testNow.Add(time.Hour))
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("error = %v, want ErrUserDisabled", err)
	}
}

func TestUser_DisableAndEnable(t *testing.T) {
	t.Parallel()

	user := testUser(t)

	if err := user.Disable(testNow.Add(time.Hour)); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if user.Status() != UserStatusDisabled {
		t.Fatal("expected disabled status")
	}
	if err := user.RequireActive(); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("RequireActive() error = %v", err)
	}

	if err := user.Enable(testNow.Add(2 * time.Hour)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if user.Status() != UserStatusActive {
		t.Fatal("expected active status")
	}
}

func TestUser_Disable_Idempotent(t *testing.T) {
	t.Parallel()

	user := testUser(t)
	_ = user.Disable(testNow.Add(time.Hour))
	user.ClearUncommittedEvents()

	if err := user.Disable(testNow.Add(2 * time.Hour)); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if len(user.UncommittedEvents()) != 0 {
		t.Fatal("idempotent disable should not emit events")
	}
}

func TestUser_ClearUncommittedEvents(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_clearuncommitted1")
	email, _ := ParseEmail("clear@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)

	user, err := RegisterUser(id, email, hash, testNow)
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if len(user.UncommittedEvents()) == 0 {
		t.Fatal("expected events before clear")
	}

	user.ClearUncommittedEvents()
	if user.UncommittedEvents() != nil {
		t.Fatal("expected nil events after clear")
	}
}

func TestUser_UncommittedEvents_ReturnsCopy(t *testing.T) {
	t.Parallel()

	id, _ := ParseUserID("usr_uncommittedcopy1")
	email, _ := ParseEmail("copy@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)

	user, err := RegisterUser(id, email, hash, testNow)
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	events := user.UncommittedEvents()
	events[0] = NewUserDisabled(id, testNow)

	if user.UncommittedEvents()[0].EventType() != "auth.user.registered" {
		t.Fatal("UncommittedEvents should return a defensive copy")
	}
}
