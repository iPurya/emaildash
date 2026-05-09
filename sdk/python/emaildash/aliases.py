from __future__ import annotations

import re
import secrets


_FIRST_NAMES = (
    "alex",
    "arden",
    "ari",
    "ashton",
    "ava",
    "ben",
    "blair",
    "cam",
    "casey",
    "chloe",
    "drew",
    "eden",
    "eli",
    "ellis",
    "emery",
    "eva",
    "finn",
    "grace",
    "hayden",
    "ivy",
    "jamie",
    "jordan",
    "kai",
    "lara",
    "leo",
    "lina",
    "logan",
    "mara",
    "mila",
    "mina",
    "niko",
    "nina",
    "nora",
    "owen",
    "rayan",
    "remy",
    "riley",
    "sara",
    "sasha",
    "talia",
    "theo",
    "violet",
    "zara",
)

_LAST_NAMES = (
    "adler",
    "ames",
    "avery",
    "bennett",
    "brooks",
    "calder",
    "carr",
    "ellis",
    "foster",
    "grant",
    "hart",
    "hayes",
    "lane",
    "marin",
    "miller",
    "monroe",
    "nolan",
    "parker",
    "quinn",
    "reed",
    "rivera",
    "rowan",
    "shaw",
    "stone",
    "taylor",
    "vale",
    "wells",
)

_SOFT_NOUNS = (
    "archive",
    "atelier",
    "bloom",
    "canvas",
    "chapter",
    "corner",
    "field",
    "folder",
    "garden",
    "harbor",
    "ledger",
    "market",
    "meadow",
    "notebook",
    "paper",
    "parcel",
    "river",
    "studio",
    "summer",
    "window",
)

_BANNED_PARTS = {
    "ai",
    "bot",
    "fake",
    "mail",
    "random",
    "robot",
    "spam",
    "temp",
    "test",
    "throw",
    "trash",
}

_USERNAME_RE = re.compile(r"^[a-z0-9](?:[a-z0-9._-]{1,62}[a-z0-9])?$")


class UsernameGenerator:
    """Generate readable local-parts that look like normal personal aliases."""

    def __init__(self) -> None:
        self._random = secrets.SystemRandom()

    def generate(self, prefix: str | None = None) -> str:
        if prefix:
            username = self._clean(prefix)
            if self.is_valid(username):
                return username
            raise ValueError("prefix is not a valid email local-part")

        for _ in range(100):
            username = self._candidate()
            if self.is_valid(username):
                return username
        raise RuntimeError("unable to generate a valid username")

    def is_valid(self, username: str) -> bool:
        if not 5 <= len(username) <= 32:
            return False
        if not _USERNAME_RE.match(username):
            return False
        compact = re.sub(r"[^a-z0-9]", "", username)
        return not any(part in compact for part in _BANNED_PARTS)

    def _candidate(self) -> str:
        pattern = self._random.choices(
            population=("name", "initial", "noun", "name_noun"),
            weights=(50, 18, 14, 18),
            k=1,
        )[0]
        separator = self._random.choices((".", "_", ""), weights=(58, 14, 28), k=1)[0]
        first = self._random.choice(_FIRST_NAMES)
        last = self._random.choice(_LAST_NAMES)
        noun = self._random.choice(_SOFT_NOUNS)

        if pattern == "initial":
            base = f"{first[0]}{separator}{last}"
        elif pattern == "noun":
            base = f"{first}{separator}{noun}"
        elif pattern == "name_noun":
            base = f"{first}{separator}{last}{self._random.choice(('', '', '.', '_'))}{noun}"
        else:
            base = f"{first}{separator}{last}"

        if self._random.random() < 0.18:
            suffix = str(self._random.choice((7, 11, 12, 14, 17, 19, 21, 23, 24, 27, 31, 42, 64, 81)))
            base = f"{base}{self._random.choice(('', '', '.', '_'))}{suffix}"

        return self._clean(base)

    @staticmethod
    def _clean(value: str) -> str:
        value = value.strip().lower()
        value = re.sub(r"[^a-z0-9._-]+", "", value)
        value = re.sub(r"[._-]{2,}", ".", value)
        return value.strip("._-")
