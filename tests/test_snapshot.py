from pathlib import Path
import tempfile
import unittest
from paperwork.store import Store,verify_snapshot
class SnapshotTests(unittest.TestCase):
    def test_corruption_detected(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);source=root/'original.pdf';source.write_bytes(b'%PDF-scan')
            library=Store(root/'live')
            try:
                identity=library.import_original(source)
                library.backup(root/'backup')
                self.assertTrue(verify_snapshot(root/'backup')['verified'])
                (root/'backup/originals'/identity).write_bytes(b'corrupt')
                with self.assertRaises(ValueError):verify_snapshot(root/'backup')
            finally:library.db.close()
