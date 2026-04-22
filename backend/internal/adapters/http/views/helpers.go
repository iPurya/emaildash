package views

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/microcosm-cc/bluemonday"
)

var emailPolicy = bluemonday.UGCPolicy()
var previewPolicy = bluemonday.StrictPolicy()

func DashboardURL(tab, recipient string, emailID int64) string {
	values := url.Values{}
	if tab != "" {
		values.Set("tab", tab)
	}
	if recipient != "" {
		values.Set("recipient", recipient)
	}
	if emailID != 0 {
		values.Set("email", strconv.FormatInt(emailID, 10))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/dashboard"
	}
	return "/dashboard?" + encoded
}

func RecipientsFragmentURL(recipient string, emailID int64) string {
	return "/ui/inbox/recipients?" + fragmentValues(recipient, emailID).Encode()
}

func EmailsFragmentURL(recipient string, emailID int64) string {
	return "/ui/inbox/emails?" + fragmentValues(recipient, emailID).Encode()
}

func ViewerFragmentURL(recipient string, emailID int64) string {
	return "/ui/inbox/viewer?" + fragmentValues(recipient, emailID).Encode()
}

func fragmentValues(recipient string, emailID int64) url.Values {
	values := url.Values{}
	values.Set("tab", "inbox")
	if recipient != "" {
		values.Set("recipient", recipient)
	}
	if emailID != 0 {
		values.Set("email", strconv.FormatInt(emailID, 10))
	}
	return values
}

func NavClass(active bool) string {
	if active {
		return "list-group-item list-group-item-action active"
	}
	return "list-group-item list-group-item-action bg-dark text-light border-secondary"
}

func FormatTime(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.Local().Format("2006-01-02 15:04:05")
}

func FormatTimeValue(value time.Time) string {
	return value.Local().Format("2006-01-02 15:04:05")
}

func FormatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func PreviewText(textBody, htmlBody string) string {
	text := strings.TrimSpace(textBody)
	if text == "" {
		text = strings.TrimSpace(previewPolicy.Sanitize(htmlBody))
	}
	if len(text) > 180 {
		return text[:180] + "…"
	}
	return text
}

func RecipientsText(values []string) string {
	return strings.Join(values, ", ")
}

func SanitizedHTML(input string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, emailPolicy.Sanitize(input))
		return err
	})
}
