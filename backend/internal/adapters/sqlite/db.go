package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/purya/emaildash/backend/internal/domain"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	store := &Store{db: db}
	if err := store.applyMigrations(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) applyMigrations() error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		query, err := migrationFS.ReadFile(filepath.ToSlash(filepath.Join("migrations", entry.Name())))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if _, err := s.db.Exec(string(query)); err != nil {
			return fmt.Errorf("exec migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (s *Store) SetupStatus(ctx context.Context) (domain.SetupStatus, error) {
	var initialized bool
	if err := s.db.QueryRowContext(ctx, `SELECT initialized FROM app_state WHERE id = 1`).Scan(&initialized); err != nil {
		return domain.SetupStatus{}, fmt.Errorf("query setup status: %w", err)
	}
	return domain.SetupStatus{Initialized: initialized}, nil
}

func (s *Store) InitializeUser(ctx context.Context, hash, params string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var initialized bool
	if err := tx.QueryRowContext(ctx, `SELECT initialized FROM app_state WHERE id = 1`).Scan(&initialized); err != nil {
		return fmt.Errorf("query app state: %w", err)
	}
	if initialized {
		return fmt.Errorf("already initialized")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, password_hash, password_params, created_at, updated_at) VALUES (1, ?, ?, ?, ?)`, hash, params, now, now); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE app_state SET initialized = 1, updated_at = ? WHERE id = 1`, now); err != nil {
		return fmt.Errorf("update app state: %w", err)
	}
	return tx.Commit()
}

func (s *Store) GetUser(ctx context.Context) (domain.User, error) {
	var user domain.User
	var createdAt string
	var updatedAt string
	if err := s.db.QueryRowContext(ctx, `SELECT id, password_hash, password_params, created_at, updated_at FROM users WHERE id = 1`).Scan(&user.ID, &user.PasswordHash, &user.PasswordParams, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, fmt.Errorf("user not found")
		}
		return domain.User{}, fmt.Errorf("query user: %w", err)
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return user, nil
}

func (s *Store) UpdatePassword(ctx context.Context, hash, params string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, password_params = ?, updated_at = ? WHERE id = 1`, hash, params, now)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, tokenHash, csrfToken string, expiresAt time.Time) (domain.Session, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `INSERT INTO sessions (user_id, token_hash, csrf_token, expires_at, created_at, last_seen_at) VALUES (1, ?, ?, ?, ?, ?)`, tokenHash, csrfToken, expiresAt.Format(time.RFC3339), now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return domain.Session{}, fmt.Errorf("insert session: %w", err)
	}
	id, _ := result.LastInsertId()
	return domain.Session{ID: id, UserID: 1, TokenHash: tokenHash, CSRFToken: csrfToken, ExpiresAt: expiresAt, CreatedAt: now, LastSeenAt: now}, nil
}

func (s *Store) GetSessionByHash(ctx context.Context, tokenHash string) (domain.Session, error) {
	var session domain.Session
	var expiresAt string
	var createdAt string
	var lastSeenAt string
	var revokedAt sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT id, user_id, token_hash, csrf_token, expires_at, created_at, last_seen_at, revoked_at FROM sessions WHERE token_hash = ?`, tokenHash).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.CSRFToken, &expiresAt, &createdAt, &lastSeenAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, fmt.Errorf("session not found")
		}
		return domain.Session{}, fmt.Errorf("query session: %w", err)
	}
	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	session.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)
	if revokedAt.Valid {
		parsed, _ := time.Parse(time.RFC3339, revokedAt.String)
		session.RevokedAt = &parsed
	}
	return session, nil
}

func (s *Store) TouchSession(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (s *Store) RevokeSession(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET revoked_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Store) PutSecret(ctx context.Context, key, ciphertext string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO secrets (key, ciphertext, updated_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET ciphertext = excluded.ciphertext, updated_at = excluded.updated_at`, key, ciphertext, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("put secret: %w", err)
	}
	return nil
}

func (s *Store) GetSecret(ctx context.Context, key string) (string, error) {
	var ciphertext string
	if err := s.db.QueryRowContext(ctx, `SELECT ciphertext FROM secrets WHERE key = ?`, key).Scan(&ciphertext); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("secret not found")
		}
		return "", fmt.Errorf("query secret: %w", err)
	}
	return ciphertext, nil
}

func (s *Store) ReplaceZones(ctx context.Context, zones []domain.CloudflareZone) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM cloudflare_zones`); err != nil {
		return fmt.Errorf("clear zones: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, zone := range zones {
		if _, err := tx.ExecContext(ctx, `INSERT INTO cloudflare_zones (id, name, account_id, selected, status, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, zone.ID, zone.Name, zone.AccountID, zone.Selected, zone.Status, now); err != nil {
			return fmt.Errorf("insert zone: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) SelectZone(ctx context.Context, zoneID, status string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE cloudflare_zones SET selected = 0`); err != nil {
		return fmt.Errorf("clear selected zone: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE cloudflare_zones SET selected = 1, status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC().Format(time.RFC3339), zoneID); err != nil {
		return fmt.Errorf("select zone: %w", err)
	}
	return tx.Commit()
}

func (s *Store) ListZones(ctx context.Context) ([]domain.CloudflareZone, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, account_id, selected, status, updated_at FROM cloudflare_zones ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query zones: %w", err)
	}
	defer rows.Close()
	zones := make([]domain.CloudflareZone, 0)
	for rows.Next() {
		var zone domain.CloudflareZone
		var updatedAt string
		if err := rows.Scan(&zone.ID, &zone.Name, &zone.AccountID, &zone.Selected, &zone.Status, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan zone: %w", err)
		}
		zone.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		zones = append(zones, zone)
	}
	return zones, nil
}

func (s *Store) GetSelectedZone(ctx context.Context) (domain.CloudflareZone, error) {
	var zone domain.CloudflareZone
	var updatedAt string
	if err := s.db.QueryRowContext(ctx, `SELECT id, name, account_id, selected, status, updated_at FROM cloudflare_zones WHERE selected = 1 LIMIT 1`).Scan(&zone.ID, &zone.Name, &zone.AccountID, &zone.Selected, &zone.Status, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CloudflareZone{}, fmt.Errorf("selected zone not found")
		}
		return domain.CloudflareZone{}, fmt.Errorf("query selected zone: %w", err)
	}
	zone.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return zone, nil
}

func (s *Store) InsertEmail(ctx context.Context, email domain.Email) (domain.Email, error) {
	headersJSON, err := json.Marshal(email.Headers)
	if err != nil {
		return domain.Email{}, fmt.Errorf("marshal headers: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Email{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO emails (provider, provider_message_id, mail_from, subject, text_body, html_body, headers_json, raw_size, received_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, email.Provider, email.ProviderMessageID, email.MailFrom, email.Subject, email.TextBody, email.HTMLBody, string(headersJSON), email.RawSize, email.ReceivedAt.UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return domain.Email{}, fmt.Errorf("insert email: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		row := s.db.QueryRowContext(ctx, `SELECT id, created_at FROM emails WHERE provider = ? AND provider_message_id = ?`, email.Provider, email.ProviderMessageID)
		var existingID int64
		var createdAt string
		if err := row.Scan(&existingID, &createdAt); err != nil {
			return domain.Email{}, fmt.Errorf("query existing email: %w", err)
		}
		email.ID = existingID
		email.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		return email, nil
	}
	emailID, _ := result.LastInsertId()
	email.ID = emailID
	email.CreatedAt = time.Now().UTC()
	for _, recipient := range email.Recipients {
		if _, err := tx.ExecContext(ctx, `INSERT INTO email_recipients (email_id, recipient) VALUES (?, ?)`, emailID, recipient); err != nil {
			return domain.Email{}, fmt.Errorf("insert recipient: %w", err)
		}
	}
	for _, attachment := range email.Attachments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO attachments (email_id, filename, content_type, size, sha256, storage_path) VALUES (?, ?, ?, ?, ?, ?)`, emailID, attachment.Filename, attachment.ContentType, attachment.Size, attachment.SHA256, attachment.StoragePath); err != nil {
			return domain.Email{}, fmt.Errorf("insert attachment: %w", err)
		}
	}
	return email, tx.Commit()
}

func (s *Store) ListEmails(ctx context.Context, recipient string, unreadOnly bool, limit int) ([]domain.Email, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{}
	builder := strings.Builder{}
	builder.WriteString(`SELECT DISTINCT e.id, e.provider, e.provider_message_id, e.mail_from, e.subject, e.text_body, e.html_body, e.headers_json, e.raw_size, e.read_at, e.received_at, e.created_at FROM emails e`)
	if recipient != "" {
		builder.WriteString(` JOIN email_recipients r ON r.email_id = e.id`)
	}
	builder.WriteString(` WHERE 1 = 1`)
	if recipient != "" {
		builder.WriteString(` AND r.recipient = ?`)
		args = append(args, recipient)
	}
	if unreadOnly {
		builder.WriteString(` AND e.read_at IS NULL`)
	}
	builder.WriteString(` ORDER BY e.received_at DESC LIMIT ?`)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, builder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query emails: %w", err)
	}
	defer rows.Close()
	results := make([]domain.Email, 0)
	for rows.Next() {
		email, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		full, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, full)
	}
	return results, nil
}

func (s *Store) GetEmail(ctx context.Context, id int64) (domain.Email, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, provider, provider_message_id, mail_from, subject, text_body, html_body, headers_json, raw_size, read_at, received_at, created_at FROM emails WHERE id = ?`, id)
	email, err := scanEmail(row)
	if err != nil {
		return domain.Email{}, err
	}
	recipientRows, err := s.db.QueryContext(ctx, `SELECT recipient FROM email_recipients WHERE email_id = ? ORDER BY recipient`, id)
	if err != nil {
		return domain.Email{}, fmt.Errorf("query recipients: %w", err)
	}
	defer recipientRows.Close()
	for recipientRows.Next() {
		var recipient string
		if err := recipientRows.Scan(&recipient); err != nil {
			return domain.Email{}, fmt.Errorf("scan recipient: %w", err)
		}
		email.Recipients = append(email.Recipients, recipient)
	}
	attachmentRows, err := s.db.QueryContext(ctx, `SELECT id, filename, content_type, size, sha256, storage_path FROM attachments WHERE email_id = ? ORDER BY id`, id)
	if err != nil {
		return domain.Email{}, fmt.Errorf("query attachments: %w", err)
	}
	defer attachmentRows.Close()
	for attachmentRows.Next() {
		var attachment domain.Attachment
		if err := attachmentRows.Scan(&attachment.ID, &attachment.Filename, &attachment.ContentType, &attachment.Size, &attachment.SHA256, &attachment.StoragePath); err != nil {
			return domain.Email{}, fmt.Errorf("scan attachment: %w", err)
		}
		email.Attachments = append(email.Attachments, attachment)
	}
	return email, nil
}

func (s *Store) MarkEmailRead(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE emails SET read_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("mark email read: %w", err)
	}
	return nil
}

func (s *Store) ListRecipients(ctx context.Context) ([]domain.RecipientSummary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT r.recipient, COUNT(*), SUM(CASE WHEN e.read_at IS NULL THEN 1 ELSE 0 END), MAX(e.id), MAX(e.subject), MAX(e.received_at) FROM email_recipients r JOIN emails e ON e.id = r.email_id GROUP BY r.recipient ORDER BY MAX(e.received_at) DESC`)
	if err != nil {
		return nil, fmt.Errorf("query recipients: %w", err)
	}
	defer rows.Close()
	results := make([]domain.RecipientSummary, 0)
	for rows.Next() {
		var summary domain.RecipientSummary
		var latestEmailID sql.NullInt64
		var latestSubject sql.NullString
		var latestReceived sql.NullString
		if err := rows.Scan(&summary.Address, &summary.Count, &summary.UnreadCount, &latestEmailID, &latestSubject, &latestReceived); err != nil {
			return nil, fmt.Errorf("scan recipient summary: %w", err)
		}
		if latestEmailID.Valid {
			summary.LatestEmailID = &latestEmailID.Int64
		}
		if latestSubject.Valid {
			summary.LatestSubject = &latestSubject.String
		}
		if latestReceived.Valid {
			parsed, _ := time.Parse(time.RFC3339, latestReceived.String)
			summary.LatestReceived = &parsed
		}
		results = append(results, summary)
	}
	return results, nil
}

func (s *Store) InsertAuditLog(ctx context.Context, eventType string, details map[string]any) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit log: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO audit_log (event_type, details_json, created_at) VALUES (?, ?, ?)`, eventType, string(payload), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEmail(row scanner) (domain.Email, error) {
	var email domain.Email
	var headersJSON string
	var readAt sql.NullString
	var receivedAt string
	var createdAt string
	if err := row.Scan(&email.ID, &email.Provider, &email.ProviderMessageID, &email.MailFrom, &email.Subject, &email.TextBody, &email.HTMLBody, &headersJSON, &email.RawSize, &readAt, &receivedAt, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Email{}, fmt.Errorf("email not found")
		}
		return domain.Email{}, fmt.Errorf("scan email: %w", err)
	}
	if err := json.Unmarshal([]byte(headersJSON), &email.Headers); err != nil {
		return domain.Email{}, fmt.Errorf("unmarshal headers: %w", err)
	}
	email.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAt)
	email.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if readAt.Valid {
		parsed, _ := time.Parse(time.RFC3339, readAt.String)
		email.ReadAt = &parsed
	}
	return email, nil
}
