import unittest
from datetime import datetime, timezone
from typing import Any

from emaildash import EmailDash


class StaticGenerator:
    def generate(self, prefix: str | None = None) -> str:
        return prefix or "nora.calder"


class FakeEmailDash(EmailDash):
    def __init__(self) -> None:
        super().__init__("https://emaildash.example.test", api_key="key", username_generator=StaticGenerator())
        self.calls: list[tuple[str, str, dict[str, Any] | None]] = []

    def _request(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        json_body: dict[str, Any] | None = None,
    ) -> Any:
        self.calls.append((method, path, query))
        if path == "/api/domains":
            return {
                "readyDomains": ["example.com"],
                "domains": [{"domain": "example.com", "ready": True, "reason": "ready"}],
            }
        if path == "/api/emails":
            return {
                "emails": [
                    {
                        "id": 10,
                        "provider": "cloudflare",
                        "providerMessageId": "msg-10",
                        "mailFrom": "sender@example.net",
                        "recipients": [query["to_mail"]],
                        "subject": "Hello",
                        "textBody": "Body",
                        "htmlBody": "",
                        "headers": {},
                        "rawSize": 42,
                        "receivedAt": "2026-05-09T10:00:03Z",
                        "createdAt": "2026-05-09T10:00:04Z",
                        "attachments": [],
                    }
                ]
            }
        raise AssertionError(f"unexpected request: {method} {path}")


class ClientTest(unittest.TestCase):
    def test_issues_address_and_uses_received_after_for_latest_email(self) -> None:
        client = FakeEmailDash()
        address = client.new_address()

        self.assertEqual(address, "nora.calder@example.com")
        email = client.latest_email(address)

        self.assertIsNotNone(email)
        self.assertEqual(email.subject, "Hello")
        email_calls = [call for call in client.calls if call[1] == "/api/emails"]
        self.assertEqual(email_calls[0][2]["to_mail"], address)
        self.assertIn("received_after", email_calls[0][2])
        self.assertIsInstance(email_calls[0][2]["received_after"], str)

    def test_since_can_be_passed_explicitly(self) -> None:
        client = FakeEmailDash()
        since = datetime(2026, 5, 9, 10, 0, 0, tzinfo=timezone.utc)

        email = client.latest_email("nora.calder@example.com", since=since)

        self.assertIsNotNone(email)
        email_calls = [call for call in client.calls if call[1] == "/api/emails"]
        self.assertEqual(email_calls[0][2]["received_after"], "2026-05-09T10:00:00Z")


if __name__ == "__main__":
    unittest.main()
