import unittest
from paperwork.incorporate import incorporate
from paperwork.pfm import parse
class MemorySpace:
    def __init__(self): self.files={'Daily/2026-10-31.md':'# Diário\n\nTexto pessoal.\n'};self.writes=0
    async def read(self,path):return self.files.get(path),'revision' if path in self.files else None
    async def write(self,path,text,etag):self.files[path]=text;self.writes+=1
class IncorporationTests(unittest.IsolatedAsyncioTestCase):
    async def test_retry_preserves_manual_content_and_does_not_duplicate(self):
        space=MemorySpace();proposal=parse(['// 2026-10-31\n• Caderno\n> □ Aula'],'scan')
        await incorporate(proposal,'a'*64,space)
        self.assertIn('Texto pessoal.',space.files['Daily/2026-10-31.md'])
        await incorporate(proposal,'a'*64,space)
        self.assertEqual(space.writes,2)
        self.assertEqual(space.files['Daily/2026-10-31.md'].count('paperwork-block:'),2)
