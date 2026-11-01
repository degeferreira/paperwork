import tempfile
import unittest
from pathlib import Path
from paperwork.store import Store
from paperwork.backup import backup_with_notes
class Space:
    def __init__(self,unstable=False):self.calls=0;self.unstable=unstable
    async def files(self):return ['Daily/2026-10-31.md','Library/Test.md']
    async def read(self,path):
        self.calls+=1
        return '# Diário','new' if self.unstable and self.calls>1 else 'etag'
class BackupNotesTests(unittest.IsolatedAsyncioTestCase):
    async def test_snapshot_includes_native_notes(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);library=Store(root/'live')
            try:
                result=await backup_with_notes(library,Space(),root/'snapshot')
                self.assertTrue(result['verified'])
                self.assertEqual((root/'snapshot/notes/Daily/2026-10-31.md').read_text(),'# Diário')
                self.assertFalse((root/'snapshot/notes/Library').exists())
                with self.assertRaises(ValueError):await backup_with_notes(library,Space(True),root/'unstable')
                self.assertFalse((root/'unstable/manifest.json').exists())
            finally:library.db.close()

class RestoreTests(unittest.IsolatedAsyncioTestCase):
    async def test_restore_preserves_originals_and_native_note_paths(self):
        from paperwork.backup import restore
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);original=root/'scan.pdf';original.write_bytes(b'%PDF-original')
            library=Store(root/'live')
            try:
                identity=library.import_original(original)
                await backup_with_notes(library,Space(),root/'snapshot')
            finally:library.db.close()
            result=restore(root/'snapshot',root/'restored',root/'space')
            self.assertTrue(result['restored'])
            restored=Store(root/'restored')
            try:self.assertEqual((restored.originals/identity).read_bytes(),b'%PDF-original')
            finally:restored.db.close()
            self.assertEqual((root/'space/Daily/2026-10-31.md').read_text(),'# Diário')
            with self.assertRaises(ValueError):restore(root/'snapshot',root/'restored',root/'other-space')
