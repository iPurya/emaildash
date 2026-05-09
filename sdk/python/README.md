# EmailDash Python SDK

Small no-dependency Python client for EmailDash.

## Install

From this repository:

```bash
pip install "emaildash @ git+https://github.com/iPurya/emaildash.git#subdirectory=sdk/python"
```

Pinned to the latest SDK tag:

```bash
pip install "emaildash @ git+https://github.com/iPurya/emaildash.git@sdk-python-v0.1.1#subdirectory=sdk/python"
```

Upgrade an existing GitHub install:

```bash
pip install --upgrade --force-reinstall "emaildash @ git+https://github.com/iPurya/emaildash.git@sdk-python-v0.1.1#subdirectory=sdk/python"
```

## Quick Start

```python
import os

from emaildash import EmailDash

client = EmailDash(
    base_url=os.environ["EMAILDASH_URL"],
    api_key=os.environ["EMAILDASH_API_KEY"],
)

address = client.new_address()
print(address)

email = client.wait_for_latest_email(address, timeout=180)
print(email.subject)
print(email.body)
```

Environment helper:

```python
from emaildash import EmailDash

client = EmailDash.from_env()
```

Required environment variables:

```bash
export EMAILDASH_URL="https://emaildash.example.com"
export EMAILDASH_API_KEY="YOUR_API_KEY"
```

Optional domain blacklist:

```bash
export EMAILDASH_DOMAIN_BLACKLIST="old-domain.com,testing-only.com"
```

## Common Tasks

List domains ready to receive email:

```python
domains = client.available_domains()
```

List ready domains while excluding specific domains for this run:

```python
domains = client.available_domains(exclude_domains=["old-domain.com"])
```

Issue a new catch-all address:

```python
address = client.new_address()
```

Issue a new address without using blacklisted domains:

```python
client = EmailDash(
    base_url=os.environ["EMAILDASH_URL"],
    api_key=os.environ["EMAILDASH_API_KEY"],
    domain_blacklist=["old-domain.com", "testing-only.com"],
)

address = client.new_address()
```

Exclude a domain for only one generated address:

```python
address = client.new_address(exclude_domains=["marketing-domain.com"])
```

Issue an address on a specific domain:

```python
address = client.new_address(domain="example.com")
```

Wait for the first email received after the SDK issued the address:

```python
email = client.wait_for_latest_email(address, timeout=120)
```

Mark a message as read:

```python
client.mark_read(email.id)
```

## Address Generation

EmailDash uses catch-all routing, so the SDK does not need to create a recipient before use. It generates natural, readable local-parts such as:

```text
nora.calder@example.com
ellis.lane@example.com
mila.parcel@example.com
```

The generator avoids obvious automation words like `test`, `temp`, `bot`, `fake`, and high-entropy random strings. It is designed for readable temporary inbox aliases and normal QA/product workflows.

## API Notes

The SDK uses:

```http
Authorization: Bearer YOUR_API_KEY
```

For polling, it sends `received_after` so only messages received after the address was issued are considered.
