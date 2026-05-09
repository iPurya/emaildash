import unittest

from emaildash.aliases import UsernameGenerator


class UsernameGeneratorTest(unittest.TestCase):
    def test_generates_valid_readable_usernames(self) -> None:
        generator = UsernameGenerator()
        values = {generator.generate() for _ in range(100)}

        self.assertGreater(len(values), 80)
        for value in values:
            self.assertTrue(generator.is_valid(value), value)
            self.assertFalse(any(part in value for part in ("temp", "test", "bot", "fake", "random")))
            self.assertLessEqual(len(value), 32)

    def test_prefix_validation(self) -> None:
        generator = UsernameGenerator()

        self.assertEqual(generator.generate(prefix="Nora.Calder"), "nora.calder")
        with self.assertRaises(ValueError):
            generator.generate(prefix="temp-bot-test")


if __name__ == "__main__":
    unittest.main()
