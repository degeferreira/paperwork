import tempfile
import unittest
from pathlib import Path
from paperwork.store import Store
class HistoryTests(unittest.TestCase):
    def test_reprocessing_keeps_prior_transcription(self):
        with tempfile.TemporaryDirectory() as folder:
            source=Path(folder)/'scan.pdf';source.write_bytes(b'%PDF-original')
            library=Store(Path(folder)/'library')
            try:
                identity=library.import_original(source)
                library.propose(identity,['// 2026-10-31\n• Primeira leitura'])
                library.propose(identity,['// 2026-10-31\n• Segunda leitura'])
                rows=library.db.execute('SELECT pages FROM runs ORDER BY rowid').fetchall()
                self.assertEqual(len(rows),2)
                self.assertIn('Primeira leitura',rows[0][0])
                self.assertIn('Segunda leitura',rows[1][0])
            finally:library.db.close()
