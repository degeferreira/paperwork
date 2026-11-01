import json
import tempfile
import unittest
from pathlib import Path
import httpx
from paperwork.transcription import transcribe,validate
class TranscriptionTests(unittest.IsolatedAsyncioTestCase):
    async def test_gemini_requires_consent_and_preserves_result(self):
        with tempfile.TemporaryDirectory() as folder:
            key=Path(folder)/'key';key.write_text('test-only')
            config={'provider':'gemini','model':'test-model','key_file':str(key),'external_allowed':True}
            calls=[]
            def transport(request):
                calls.append(request)
                return httpx.Response(200,json={'candidates':[{'content':{'parts':[{'text':json.dumps({'pages':['// 2026-10-31\n• Texto']})}]}}]})
            async with httpx.AsyncClient(transport=httpx.MockTransport(transport)) as client:
                with self.assertRaises(ValueError):await transcribe(b'%PDF','application/pdf',config,client)
                self.assertEqual(calls,[])
                result=await transcribe(b'%PDF','application/pdf',config,client,consent=True)
                self.assertEqual(result['pages'],['// 2026-10-31\n• Texto'])
                self.assertEqual(result['model'],'test-model')
    async def test_schema_rejects_invented_object(self):
        for raw in ('{}','{"pages":"text"}','{"pages":[1]}','{"pages":[],"tasks":[]}'):
            with self.assertRaises(ValueError):validate(raw)
