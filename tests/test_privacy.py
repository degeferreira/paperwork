import base64
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import httpx
from fastapi import FastAPI
from paperwork.api import router
class PrivacyTests(unittest.IsolatedAsyncioTestCase):
    async def test_local_only_blocks_even_consented_gemini_request(self):
        with tempfile.TemporaryDirectory() as folder,patch.dict(os.environ,{'PAPERWORK_HOME':folder,'PAPERWORK_CONFIG':str(Path(folder)/'config.json')}):
            Path(folder,'config.json').write_text(json.dumps({'transcription':{'provider':'gemini','external_allowed':True,'model':'test'}}))
            app=FastAPI();app.include_router(router)
            async with httpx.AsyncClient(transport=httpx.ASGITransport(app=app),base_url='http://local') as client:
                identity=(await client.post('/v1/paperwork/documents',json={'name':'private.pdf','data':base64.b64encode(b'%PDF-original').decode()})).json()['id']
                response=await client.post(f'/v1/paperwork/documents/{identity}/transcribe',json={'consent_external':True})
                self.assertEqual(response.status_code,409)
                self.assertEqual((await client.get('/v1/paperwork/jobs')).json(),[])
