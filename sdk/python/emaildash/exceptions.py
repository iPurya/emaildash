class EmailDashError(Exception):
    """Base exception for EmailDash SDK errors."""


class EmailDashHTTPError(EmailDashError):
    def __init__(self, status_code: int, message: str, body: str = "") -> None:
        super().__init__(message)
        self.status_code = status_code
        self.body = body


class EmailDashAuthError(EmailDashHTTPError):
    """Raised when the EmailDash API rejects credentials."""


class NoReadyDomainError(EmailDashError):
    """Raised when no domain is ready to receive email."""


class EmailTimeoutError(EmailDashError):
    """Raised when waiting for an email exceeds the timeout."""
