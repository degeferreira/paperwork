import hashlib
import sqlite3
import tempfile
import unittest
from pathlib import Path
from paperwork.store import Store

class StoreTests(unittest.TestCase):
    def test_dedup_correction_and_restore(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary); original=root/'scan.pdf'; original.write_bytes(b'%PDF-original')
            store=Store(root/'live')
            try:
                identity=store.import_original(original)
                self.assertEqual(identity,store.import_original(original))
                proposal=store.propose(identity,['// 2026-10-31\n• Memória'])
                block=proposal['blocks'][0]['id']
                with store.db: store.db.execute('INSERT INTO corrections VALUES (?,?)',(block,'Correção manual'))
                store.propose(identity,['// 2026-10-31\n• Nova leitura'])
                self.assertEqual(store.db.execute('SELECT text FROM corrections').fetchone()[0],'Correção manual')
                manifest=store.backup(root/'snapshot')
                for name,digest in manifest.items():
                    self.assertEqual(hashlib.sha256((root/'snapshot'/name).read_bytes()).hexdigest(),digest)
                with sqlite3.connect(root/'snapshot/paperwork.sqlite') as restored:
                    self.assertEqual(restored.execute('PRAGMA integrity_check').fetchone()[0],'ok')
                    self.assertEqual(restored.execute('SELECT count(*) FROM documents').fetchone()[0],1)
            finally: store.db.close()

    def test_corrupted_original_is_not_silently_deleted_or_replaced(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);source=root/'scan.pdf';source.write_bytes(b'%PDF-source')
            store=Store(root/'library')
            try:
                identity=store.import_original(source)
                target=store.originals/identity;target.write_bytes(b'corruption')
                with self.assertRaises(ValueError):store.import_original(source)
                self.assertEqual(target.read_bytes(),b'corruption')
            finally:store.db.close()
