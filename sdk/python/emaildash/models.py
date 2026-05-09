from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any


def parse_datetime(value: str | None) -> datetime | None:
    if not value:
        return None
    normalized = value.replace("Z", "+00:00")
    parsed = datetime.fromisoformat(normalized)
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def format_datetime(value: datetime) -> str:
    if value.tzinfo is None:
        value = value.replace(tzinfo=timezone.utc)
    return value.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


@dataclass(frozen=True)
class ReceivingDomain:
    domain: str
    zone_id: str = ""
    ready: bool = False
    reason: str = ""
    status_error: str = ""
    email_routing_enabled: bool = False
    email_routing_status: str = ""
    catch_all_enabled: bool = False
    catch_all_destination: str = ""
    worker_script_name: str = ""

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "ReceivingDomain":
        return cls(
            domain=str(data.get("domain", "")),
            zone_id=str(data.get("zoneId", "")),
            ready=bool(data.get("ready", False)),
            reason=str(data.get("reason", "")),
            status_error=str(data.get("statusError", "")),
            email_routing_enabled=bool(data.get("emailRoutingEnabled", False)),
            email_routing_status=str(data.get("emailRoutingStatus", "")),
            catch_all_enabled=bool(data.get("catchAllEnabled", False)),
            catch_all_destination=str(data.get("catchAllDestination", "")),
            worker_script_name=str(data.get("workerScriptName", "")),
        )


@dataclass(frozen=True)
class Attachment:
    id: int
    filename: str
    content_type: str
    size: int
    sha256: str

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "Attachment":
        return cls(
            id=int(data.get("id", 0)),
            filename=str(data.get("filename", "")),
            content_type=str(data.get("contentType", "")),
            size=int(data.get("size", 0)),
            sha256=str(data.get("sha256", "")),
        )


@dataclass(frozen=True)
class Email:
    id: int
    provider: str
    provider_message_id: str
    mail_from: str
    recipients: list[str]
    subject: str
    text_body: str
    html_body: str
    headers: dict[str, list[str]]
    raw_size: int
    received_at: datetime
    created_at: datetime | None = None
    read_at: datetime | None = None
    attachments: list[Attachment] = field(default_factory=list)

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "Email":
        received_at = parse_datetime(str(data.get("receivedAt", "")))
        if received_at is None:
            received_at = datetime.fromtimestamp(0, tz=timezone.utc)
        headers = data.get("headers") or {}
        normalized_headers = {
            str(key): [str(item) for item in value]
            for key, value in headers.items()
            if isinstance(value, list)
        }
        return cls(
            id=int(data.get("id", 0)),
            provider=str(data.get("provider", "")),
            provider_message_id=str(data.get("providerMessageId", "")),
            mail_from=str(data.get("mailFrom", "")),
            recipients=[str(item) for item in data.get("recipients", [])],
            subject=str(data.get("subject", "")),
            text_body=str(data.get("textBody", "")),
            html_body=str(data.get("htmlBody", "")),
            headers=normalized_headers,
            raw_size=int(data.get("rawSize", 0)),
            read_at=parse_datetime(data.get("readAt")),
            received_at=received_at,
            created_at=parse_datetime(data.get("createdAt")),
            attachments=[Attachment.from_dict(item) for item in data.get("attachments", [])],
        )

    @property
    def body(self) -> str:
        return self.text_body or self.html_body

    @property
    def is_unread(self) -> bool:
        return self.read_at is None


@dataclass(frozen=True)
class RecipientSummary:
    address: str
    count: int
    unread_count: int
    latest_email_id: int | None = None
    latest_subject: str | None = None
    latest_received: datetime | None = None

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "RecipientSummary":
        return cls(
            address=str(data.get("address", "")),
            count=int(data.get("count", 0)),
            unread_count=int(data.get("unreadCount", 0)),
            latest_email_id=data.get("latestEmailId"),
            latest_subject=data.get("latestSubject"),
            latest_received=parse_datetime(data.get("latestReceived")),
        )


@dataclass(frozen=True)
class IssuedAddress:
    address: str
    username: str
    domain: str
    issued_at: datetime
