package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
)

func TestInsertEmailDuplicateDoesNotLockStore(t *testing.T) {
	store := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	email := testEmail("message-1", "alice@example.com")
	first, err := store.InsertEmail(ctx, email)
	if err != nil {
		t.Fatalf("insert first email: %v", err)
	}
	second, err := store.InsertEmail(ctx, email)
	if err != nil {
		t.Fatalf("insert duplicate email: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate insert to return existing id %d, got %d", first.ID, second.ID)
	}
	emails, err := store.ListEmails(ctx, domain.EmailListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list emails after duplicate insert: %v", err)
	}
	if len(emails) != 1 {
		t.Fatalf("expected one stored email, got %d", len(emails))
	}
}

func TestListEmailsHydratesAfterClosingListRows(t *testing.T) {
	store := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := store.InsertEmail(ctx, testEmail("message-1", "alice@example.com")); err != nil {
		t.Fatalf("insert first email: %v", err)
	}
	if _, err := store.InsertEmail(ctx, testEmail("message-2", "bob@example.com")); err != nil {
		t.Fatalf("insert second email: %v", err)
	}
	emails, err := store.ListEmails(ctx, domain.EmailListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list emails: %v", err)
	}
	if len(emails) != 2 {
		t.Fatalf("expected two emails, got %d", len(emails))
	}
	for _, email := range emails {
		if len(email.Recipients) != 1 {
			t.Fatalf("expected hydrated recipients for email %d, got %+v", email.ID, email.Recipients)
		}
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "emaildash.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}

func testEmail(messageID, recipient string) domain.Email {
	return domain.Email{
		Provider:          "cloudflare",
		ProviderMessageID: messageID,
		MailFrom:          "sender@example.com",
		Recipients:        []string{recipient},
		Subject:           "Subject",
		TextBody:          "Body",
		HTMLBody:          "",
		Headers:           map[string][]string{"from": {"sender@example.com"}},
		RawSize:           128,
		ReceivedAt:        time.Now().UTC(),
	}
}
