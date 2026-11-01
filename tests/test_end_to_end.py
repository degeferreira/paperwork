import asyncio
import base64
import os
import tempfile
import unittest
from unittest.mock import patch
from fastapi import FastAPI
import httpx
from paperwork.api import router
class Space:
    def __init__(self):self.lock=asyncio.Lock();self.notes={'Daily/2026-10-29.md':'# Meu diário\nTexto pessoal.\n'}
    async def read(self,path):return self.notes.get(path),'etag' if path in self.notes else None
    async def write(self,path,text,etag=None):self.notes[path]=text
class EndToEndTests(unittest.IsolatedAsyncioTestCase):
    async def test_three_days_citation_long_note_and_safe_reimport(self):
        with tempfile.TemporaryDirectory() as folder,patch.dict(os.environ,{'PAPERWORK_HOME':folder}):
            app=FastAPI();app.include_router(router);space=Space();app.state.service=space
            async with httpx.AsyncClient(transport=httpx.ASGITransport(app=app),base_url='http://local') as client:
                identity=(await client.post('/v1/paperwork/documents',json={'name':'caderno.pdf','data':base64.b64encode(b'%PDF-original').decode()})).json()['id']
                alias=await client.post('/v1/paperwork/references',json={'alias':'Silva24-Mem','library_type':'users','library_id':123,'item_key':'ABCD1234','note_path':'Literature/Silva'})
                self.assertEqual(alias.status_code,200)
                pages=['// 2026-10-29\n• Leitura Ƶ Silva24-Mem p.12\n  • Recuo\n> ! Ideia\n>> ? Pergunta',
                       '// 2026-10-30\n□ Aula\n> □ Slides\n//end',
                       '// 2026-10-31\n• Terceiro dia\n//end\n:: Ensaio\nTexto longo.\n::end']
                proposed=await client.post(f'/v1/paperwork/documents/{identity}/propose',json={'pages':pages})
                self.assertEqual(proposed.status_code,200)
                run_id=proposed.json()['run_id']
                endpoint=f'/v1/paperwork/documents/{identity}/incorporate'
                self.assertEqual((await client.post(endpoint,json={'reviewed':False})).status_code,409)
                self.assertEqual((await client.post(endpoint,json={'reviewed':True,'run_id':run_id})).status_code,200)
                self.assertIn('Texto pessoal.',space.notes['Daily/2026-10-29.md'])
                self.assertIn('[[Literature/Silva|Ƶ Silva24-Mem]]',space.notes['Daily/2026-10-29.md'])
                self.assertIn('Paperwork/Pages/Ensaio.md',space.notes)
                before=dict(space.notes)
                self.assertEqual((await client.post(endpoint,json={'reviewed':True,'run_id':run_id})).status_code,200)
                self.assertEqual(space.notes,before)
                space.notes['Moved/Ensaio.md']=space.notes.pop('Paperwork/Pages/Ensaio.md')
                before=dict(space.notes)
                repeated=await client.post(endpoint,json={'reviewed':True,'run_id':run_id})
                self.assertTrue(repeated.json()['already_incorporated'])
                self.assertEqual(space.notes,before)
                pages[0]=pages[0].replace('Leitura','Alterada')
                await client.post(f'/v1/paperwork/documents/{identity}/propose',json={'pages':pages})
                self.assertEqual((await client.post(endpoint,json={'reviewed':True,'run_id':run_id})).status_code,409)
                self.assertEqual(space.notes,before)
