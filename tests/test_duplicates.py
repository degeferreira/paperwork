from pathlib import Path
import tempfile
import unittest
from paperwork.store import Store
from paperwork.duplicates import register
class DuplicateTests(unittest.TestCase):
    def test_new_capture_same_segment_is_flagged_without_merging(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);library=Store(root/'library')
            try:
                a=root/'a.pdf';a.write_bytes(b'%PDF-capture-a')
                b=root/'b.pdf';b.write_bytes(b'%PDF-capture-b')
                first=library.import_original(a);second=library.import_original(b)
                pages=['// 2026-10-31\n• Leitura\n> ! Ideia']
                proposal=library.propose(first,pages);register(library.db,proposal,first)
                duplicate=library.propose(second,pages)
                self.assertEqual(duplicate['possible_duplicates'][0]['document_id'],first)
                self.assertNotEqual(duplicate['blocks'][0]['id'],proposal['blocks'][0]['id'])
                self.assertNotIn('possible_duplicates',library.propose(first,pages))
            finally:library.db.close()
