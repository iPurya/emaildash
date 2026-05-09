# EmailDash Python SDK

Small no-dependency Python client for EmailDash.

## Install

From this repository:

```bash
pip install "emaildash @ git+https://github.com/iPurya/emaildash.git#subdirectory=sdk/python"
```

Pinned to a tag later:

```bash
pip install "emaildash @ git+https://github.com/iPurya/emaildash.git@sdk-python-v0.1.0#subdirectory=sdk/python"
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

## Common Tasks

List domains ready to receive email:

```python
domains = client.available_domains()
```

Issue a new catch-all address:

```python
address = client.new_address()
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
ellis_lane@example.com
mila.parcel@example.com
```

The generator avoids obvious automation words like `test`, `temp`, `bot`, `fake`, and high-entropy random strings. It is designed for readable temporary inbox aliases and normal QA/product workflows.

## API Notes

The SDK uses:

```http
Authorization: Bearer YOUR_API_KEY
```

For polling, it sends `received_after` so only messages received after the address was issued are considered.
