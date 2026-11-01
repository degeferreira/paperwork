import shutil
import struct
import tempfile
import unittest
from pathlib import Path
from pdf_fixture import create_pdf
from paperwork.pdf import page_count,render_page
@unittest.skipUnless(shutil.which('pdfinfo') and shutil.which('pdftoppm') and shutil.which('prlimit'),'Poppler and prlimit required')
class PDFTests(unittest.TestCase):
    def test_native_arm64_pdf_rendering_is_bounded(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);source=root/'test.pdf';create_pdf(source)
            self.assertEqual(page_count(source),1)
            data=render_page(source,1,root/'page')
            self.assertTrue(data.startswith(b'\x89PNG\r\n\x1a\n'))
            width,height=struct.unpack('>II',data[16:24])
            self.assertLessEqual(max(width,height),1800)
