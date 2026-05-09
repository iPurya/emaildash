from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from typing import Any

from .aliases import UsernameGenerator
from .exceptions import EmailDashAuthError, EmailDashHTTPError, EmailTimeoutError, NoReadyDomainError
from .models import Email, IssuedAddress, ReceivingDomain, RecipientSummary, format_datetime, parse_datetime


class EmailDash:
    def __init__(
        self,
        base_url: str,
        api_key: str | None = None,
        *,
        timeout: float = 15.0,
        username_generator: UsernameGenerator | None = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.username_generator = username_generator or UsernameGenerator()
        self._issued_at: dict[str, datetime] = {}

    @classmethod
    def from_env(cls) -> "EmailDash":
        return cls(
            base_url=os.environ["EMAILDASH_URL"],
            api_key=os.environ.get("EMAILDASH_API_KEY"),
        )

    def domain_status(self) -> list[ReceivingDomain]:
        data = self._request("GET", "/api/domains")
        return [ReceivingDomain.from_dict(item) for item in data.get("domains", [])]

    def available_domains(self) -> list[str]:
        data = self._request("GET", "/api/domains")
        ready = [str(item) for item in data.get("readyDomains", [])]
        if ready:
            return ready
        return [item.domain for item in self.domain_status() if item.ready]

    def new_username(self, prefix: str | None = None) -> str:
        return self.username_generator.generate(prefix=prefix)

    def issue_address(self, *, domain: str | None = None, username: str | None = None, prefix: str | None = None) -> IssuedAddress:
        ready_domains = self.available_domains()
        if domain is None:
            if not ready_domains:
                raise NoReadyDomainError("no domains are ready to receive email")
            domain = secrets_choice(ready_domains)
        elif domain not in ready_domains:
            raise NoReadyDomainError(f"{domain} is not ready to receive email")

        for _ in range(100):
            if username:
                local_part = self.new_username(prefix=username)
            else:
                local_part = self.new_username(prefix=prefix)
            address = f"{local_part}@{domain}".lower()
            if address not in self._issued_at:
                issued_at = datetime.now(timezone.utc)
                self._issued_at[address] = issued_at
                return IssuedAddress(address=address, username=local_part, domain=domain, issued_at=issued_at)
            username = None
        raise RuntimeError("unable to issue a unique address in this client session")

    def new_address(self, *, domain: str | None = None, username: str | None = None, prefix: str | None = None) -> str:
        return self.issue_address(domain=domain, username=username, prefix=prefix).address

    def list_recipients(self) -> list[RecipientSummary]:
        data = self._request("GET", "/api/recipients")
        return [RecipientSummary.from_dict(item) for item in data.get("recipients", [])]

    def list_emails(
        self,
        *,
        address: str | None = None,
        recipient: str | None = None,
        from_mail: str | None = None,
        unread: bool | None = None,
        received_after: datetime | str | None = None,
        limit: int = 50,
    ) -> list[Email]:
        query: dict[str, Any] = {"limit": limit}
        if address:
            query["to_mail"] = address
        if recipient:
            query["recipient"] = recipient
        if from_mail:
            query["from_mail"] = from_mail
        if unread is not None:
            query["unread"] = unread
        if received_after is not None:
            query["received_after"] = _datetime_query(received_after)

        data = self._request("GET", "/api/emails", query=query)
        return [Email.from_dict(item) for item in data.get("emails", [])]

    def get_email(self, email_id: int) -> Email:
        data = self._request("GET", f"/api/emails/{email_id}")
        return Email.from_dict(data)

    def latest_email(
        self,
        address: str,
        *,
        since: datetime | str | None = None,
        unread: bool | None = None,
        limit: int = 25,
    ) -> Email | None:
        since_dt = self._resolve_since(address, since)
        emails = self.list_emails(address=address, unread=unread, received_after=since_dt, limit=limit)
        if since_dt is not None:
            emails = [email for email in emails if email.received_at >= since_dt]
        if not emails:
            return None
        return max(emails, key=lambda email: (email.received_at, email.id))

    def wait_for_latest_email(
        self,
        address: str,
        *,
        since: datetime | str | None = None,
        timeout: float = 120.0,
        interval: float = 3.0,
        unread: bool | None = None,
        mark_read: bool = False,
    ) -> Email:
        if since is None and address.lower() not in self._issued_at:
            since = datetime.now(timezone.utc)
        deadline = time.monotonic() + timeout
        while True:
            email = self.latest_email(address, since=since, unread=unread)
            if email is not None:
                if mark_read:
                    self.mark_read(email.id)
                return email

            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise EmailTimeoutError(f"no email arrived for {address} within {timeout:g} seconds")
            time.sleep(min(interval, remaining))

    def mark_read(self, email_id: int) -> None:
        self._request("PATCH", f"/api/emails/{email_id}/read")

    def _resolve_since(self, address: str, since: datetime | str | None) -> datetime | None:
        if since is None:
            return self._issued_at.get(address.lower())
        if isinstance(since, datetime):
            if since.tzinfo is None:
                return since.replace(tzinfo=timezone.utc)
            return since.astimezone(timezone.utc)
        return parse_datetime(since)

    def _request(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        json_body: dict[str, Any] | None = None,
    ) -> Any:
        url = self.base_url + path
        if query:
            encoded = urllib.parse.urlencode({key: _query_value(value) for key, value in query.items() if value is not None and value != ""})
            if encoded:
                url = f"{url}?{encoded}"

        body = None
        headers = {
            "Accept": "application/json",
            "User-Agent": "emaildash-python/0.1.0",
        }
        if self.api_key:
            headers["Authorization"] = f"Bearer {self.api_key}"
        if json_body is not None:
            body = json.dumps(json_body).encode("utf-8")
            headers["Content-Type"] = "application/json"

        request = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                payload = response.read()
                if response.status == 204 or not payload:
                    return None
                return json.loads(payload.decode("utf-8"))
        except urllib.error.HTTPError as error:
            payload = error.read().decode("utf-8", errors="replace")
            message = _error_message(payload) or f"EmailDash API returned HTTP {error.code}"
            if error.code in (401, 403):
                raise EmailDashAuthError(error.code, message, payload) from error
            raise EmailDashHTTPError(error.code, message, payload) from error
        except urllib.error.URLError as error:
            raise EmailDashHTTPError(0, f"EmailDash API request failed: {error.reason}") from error


def _query_value(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, datetime):
        return format_datetime(value)
    return str(value)


def _datetime_query(value: datetime | str) -> str:
    if isinstance(value, datetime):
        return format_datetime(value)
    parsed = parse_datetime(value)
    if parsed is None:
        raise ValueError("datetime value is empty")
    return format_datetime(parsed)


def _error_message(payload: str) -> str:
    try:
        data = json.loads(payload)
    except json.JSONDecodeError:
        return payload.strip()
    error = data.get("error")
    return str(error) if error else payload.strip()


def secrets_choice(values: list[str]) -> str:
    import secrets

    return values[secrets.randbelow(len(values))]
