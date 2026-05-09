import unittest
from datetime import datetime, timedelta, timezone
from typing import Any

from emaildash import EmailDash, NoReadyDomainError


class StaticGenerator:
    def generate(self, prefix: str | None = None) -> str:
        return prefix or "nora.calder"


class FakeEmailDash(EmailDash):
    def __init__(
        self,
        *,
        ready_domains: list[str] | None = None,
        domain_rows: list[dict[str, Any]] | None = None,
        **client_kwargs: Any,
    ) -> None:
        super().__init__(
            "https://emaildash.example.test",
            api_key="key",
            username_generator=StaticGenerator(),
            **client_kwargs,
        )
        self.ready_domains = ["example.com"] if ready_domains is None else ready_domains
        self.domain_rows = domain_rows
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
                "readyDomains": self.ready_domains,
                "domains": self.domain_rows
                if self.domain_rows is not None
                else [{"domain": domain, "ready": True, "reason": "ready"} for domain in self.ready_domains],
            }
        if path == "/api/emails":
            received_at = "2026-05-09T10:00:03Z"
            if query and query.get("received_after"):
                requested_after = datetime.fromisoformat(str(query["received_after"]).replace("Z", "+00:00"))
                received_at = (requested_after + timedelta(seconds=3)).strftime("%Y-%m-%dT%H:%M:%SZ")
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
                        "receivedAt": received_at,
                        "createdAt": received_at,
                        "attachments": [],
                    }
                ]
            }
        raise AssertionError(f"unexpected request: {method} {path}")


def _domain_call_count(client: FakeEmailDash) -> int:
    return len([call for call in client.calls if call[1] == "/api/domains"])


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

    def test_available_domains_respects_blacklist(self) -> None:
        client = FakeEmailDash(
            ready_domains=["example.com", "blocked.com", "Other.COM."],
            domain_blacklist=["BLOCKED.com"],
        )

        self.assertEqual(client.available_domains(), ["example.com", "other.com"])
        self.assertEqual(client.available_domains(exclude_domains=["example.com"]), ["other.com"])

        client.add_domain_blacklist("other.com")
        self.assertEqual(client.domain_blacklist, ("blocked.com", "other.com"))
        self.assertEqual(client.available_domains(), ["example.com"])

    def test_explicit_blacklisted_domain_is_rejected(self) -> None:
        client = FakeEmailDash(ready_domains=["example.com", "blocked.com"], domain_blacklist=["blocked.com"])

        with self.assertRaisesRegex(NoReadyDomainError, "blacklisted"):
            client.new_address(domain="Blocked.com.")

    def test_available_domains_falls_back_to_domain_status(self) -> None:
        client = FakeEmailDash(
            ready_domains=[],
            domain_rows=[
                {"domain": "example.com", "ready": True, "reason": "ready"},
                {"domain": "not-ready.com", "ready": False, "reason": "missing catch-all"},
            ],
        )

        self.assertEqual(client.available_domains(), ["example.com"])

    def test_available_domains_uses_cache_until_refresh(self) -> None:
        client = FakeEmailDash(ready_domains=["example.com"])

        self.assertEqual(client.available_domains(), ["example.com"])
        client.ready_domains = ["changed.com"]

        self.assertEqual(client.available_domains(), ["example.com"])
        self.assertEqual(_domain_call_count(client), 1)

        self.assertEqual(client.available_domains(refresh=True), ["changed.com"])
        self.assertEqual(client.new_address(), "nora.calder@changed.com")
        self.assertEqual(_domain_call_count(client), 2)
        self.assertEqual(client.calls[-1][2], {"refresh": True})

    def test_new_address_can_refresh_domain_cache(self) -> None:
        client = FakeEmailDash(ready_domains=["example.com"])

        self.assertEqual(client.available_domains(), ["example.com"])
        client.ready_domains = ["changed.com"]

        self.assertEqual(client.new_address(refresh_domains=True), "nora.calder@changed.com")
        self.assertEqual(_domain_call_count(client), 2)
        self.assertEqual(client.calls[-1][2], {"refresh": True})

    def test_domain_cache_can_be_disabled(self) -> None:
        client = FakeEmailDash(ready_domains=["example.com"], domain_cache_ttl=0)

        self.assertEqual(client.available_domains(), ["example.com"])
        client.ready_domains = ["changed.com"]

        self.assertEqual(client.available_domains(), ["changed.com"])
        self.assertEqual(_domain_call_count(client), 2)


if __name__ == "__main__":
    unittest.main()
