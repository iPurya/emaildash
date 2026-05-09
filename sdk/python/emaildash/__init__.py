from .client import EmailDash
from .exceptions import (
    EmailDashAuthError,
    EmailDashError,
    EmailDashHTTPError,
    EmailTimeoutError,
    NoReadyDomainError,
)
from .models import Attachment, Email, IssuedAddress, ReceivingDomain, RecipientSummary

__all__ = [
    "Attachment",
    "Email",
    "EmailDash",
    "EmailDashAuthError",
    "EmailDashError",
    "EmailDashHTTPError",
    "EmailTimeoutError",
    "IssuedAddress",
    "NoReadyDomainError",
    "ReceivingDomain",
    "RecipientSummary",
]

__version__ = "0.1.1"
