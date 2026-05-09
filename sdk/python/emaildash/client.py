from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
from collections.abc import Iterable
from datetime import datetime, timezone
from typing import Any

from .aliases import UsernameGenerator
from .exceptions import EmailDashAuthError, EmailDashHTTPError, EmailTimeoutError, NoReadyDomainError
from .models import Email, IssuedAddress, ReceivingDomain, RecipientSummary, format_datetime, parse_datetime


DEFAULT_DOMAIN_CACHE_TTL = 3600.0


class EmailDash:
    def __init__(
        self,
        base_url: str,
        api_key: str | None = None,
        *,
        timeout: float = 15.0,
        username_generator: UsernameGenerator | None = None,
        domain_blacklist: Iterable[str] | str | None = None,
        domain_cache_ttl: float = DEFAULT_DOMAIN_CACHE_TTL,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.username_generator = username_generator or UsernameGenerator()
        self._domain_blacklist = _normalize_domains(domain_blacklist)
        self.domain_cache_ttl = _normalize_cache_ttl(domain_cache_ttl)
        self._ready_domains_cache: list[str] | None = None
        self._ready_domains_cache_expires_at = 0.0
        self._issued_at: dict[str, datetime] = {}

    @classmethod
    def from_env(cls) -> "EmailDash":
        return cls(
            base_url=os.environ["EMAILDASH_URL"],
            api_key=os.environ.get("EMAILDASH_API_KEY"),
            domain_blacklist=_env_domain_blacklist(),
            domain_cache_ttl=_env_domain_cache_ttl(),
        )

    @property
    def domain_blacklist(self) -> tuple[str, ...]:
        return tuple(sorted(self._domain_blacklist))

    def set_domain_blacklist(self, domains: Iterable[str] | str | None) -> None:
        self._domain_blacklist = _normalize_domains(domains)

    def add_domain_blacklist(self, *domains: str) -> None:
        self._domain_blacklist.update(_normalize_domains(domains))

    def clear_domain_blacklist(self) -> None:
        self._domain_blacklist.clear()

    def clear_domain_cache(self) -> None:
        self._ready_domains_cache = None
        self._ready_domains_cache_expires_at = 0.0

    def domain_status(self) -> list[ReceivingDomain]:
        data = self._request("GET", "/api/domains")
        return _domain_statuses(data)

    def available_domains(
        self,
        *,
        exclude_domains: Iterable[str] | str | None = None,
        refresh: bool = False,
    ) -> list[str]:
        return _filter_domains(self._load_ready_domains(refresh=refresh), self._blocked_domains(exclude_domains))

    def new_username(self, prefix: str | None = None) -> str:
        return self.username_generator.generate(prefix=prefix)

    def issue_address(
        self,
        *,
        domain: str | None = None,
        username: str | None = None,
        prefix: str | None = None,
        exclude_domains: Iterable[str] | str | None = None,
        refresh_domains: bool = False,
    ) -> IssuedAddress:
        ready_domains = self.available_domains(exclude_domains=exclude_domains, refresh=refresh_domains)
        if domain is None:
            if not ready_domains:
                raise NoReadyDomainError("no domains are ready to receive email")
            domain = secrets_choice(ready_domains)
        else:
            domain = _normalize_domain(domain)
            if domain in self._blocked_domains(exclude_domains):
                raise NoReadyDomainError(f"{domain} is blacklisted")
            if domain not in ready_domains:
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

    def new_address(
        self,
        *,
        domain: str | None = None,
        username: str | None = None,
        prefix: str | None = None,
        exclude_domains: Iterable[str] | str | None = None,
        refresh_domains: bool = False,
    ) -> str:
        return self.issue_address(
            domain=domain,
            username=username,
            prefix=prefix,
            exclude_domains=exclude_domains,
            refresh_domains=refresh_domains,
        ).address

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
            "User-Agent": "emaildash-python/0.1.2",
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

    def _blocked_domains(self, extra_domains: Iterable[str] | str | None = None) -> set[str]:
        blocked = set(self._domain_blacklist)
        blocked.update(_normalize_domains(extra_domains))
        return blocked

    def _load_ready_domains(self, *, refresh: bool = False) -> list[str]:
        now = time.monotonic()
        if (
            not refresh
            and self.domain_cache_ttl > 0
            and self._ready_domains_cache is not None
            and now < self._ready_domains_cache_expires_at
        ):
            return list(self._ready_domains_cache)

        query = {"refresh": True} if refresh else None
        data = self._request("GET", "/api/domains", query=query)
        ready_domains = _ready_domains(data)
        if self.domain_cache_ttl > 0:
            self._ready_domains_cache = list(ready_domains)
            self._ready_domains_cache_expires_at = now + self.domain_cache_ttl
        else:
            self.clear_domain_cache()
        return ready_domains


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


def _domain_statuses(data: dict[str, Any]) -> list[ReceivingDomain]:
    return [ReceivingDomain.from_dict(item) for item in data.get("domains") or []]


def _ready_domains(data: dict[str, Any]) -> list[str]:
    ready = _normalize_domain_list(data.get("readyDomains") or [])
    if ready:
        return ready
    return _normalize_domain_list(status.domain for status in _domain_statuses(data) if status.ready)


def _normalize_domain_list(domains: Iterable[Any]) -> list[str]:
    values: list[str] = []
    seen: set[str] = set()
    for domain in domains:
        normalized = _normalize_domain(str(domain))
        if normalized and normalized not in seen:
            values.append(normalized)
            seen.add(normalized)
    return values


def _filter_domains(domains: list[str], blocked: set[str]) -> list[str]:
    if not blocked:
        return domains
    return [domain for domain in domains if _normalize_domain(domain) not in blocked]


def _normalize_domains(domains: Iterable[str] | str | None) -> set[str]:
    if domains is None:
        return set()
    if isinstance(domains, str):
        domains = [domains]
    return {normalized for domain in domains if (normalized := _normalize_domain(str(domain)))}


def _normalize_domain(domain: str) -> str:
    return domain.strip().lower().rstrip(".")


def _env_domain_blacklist() -> set[str]:
    value = os.environ.get("EMAILDASH_DOMAIN_BLACKLIST", "")
    return _normalize_domains(item.strip() for item in value.split(","))


def _env_domain_cache_ttl() -> float:
    value = os.environ.get("EMAILDASH_DOMAIN_CACHE_TTL_SECONDS", "")
    if not value.strip():
        return DEFAULT_DOMAIN_CACHE_TTL
    try:
        return _normalize_cache_ttl(float(value))
    except ValueError as error:
        raise ValueError("EMAILDASH_DOMAIN_CACHE_TTL_SECONDS must be a non-negative number") from error


def _normalize_cache_ttl(value: float) -> float:
    value = float(value)
    if value < 0:
        raise ValueError("domain_cache_ttl must be non-negative")
    return value


def secrets_choice(values: list[str]) -> str:
    import secrets

    return values[secrets.randbelow(len(values))]
