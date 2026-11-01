import asyncio
import base64
import os
import tempfile
import unittest
from unittest.mock import patch
import httpx
from fastapi import FastAPI
from paperwork.api import router

class APITests(unittest.IsolatedAsyncioTestCase):
    async def test_import_proposal_original_and_search(self):
        with tempfile.TemporaryDirectory() as folder,patch.dict(os.environ,{'PAPERWORK_HOME':folder}):
            app=FastAPI();app.include_router(router)
            async with httpx.AsyncClient(transport=httpx.ASGITransport(app=app),base_url='http://local') as client:
                payload={'name':'scan.pdf','data':base64.b64encode(b'%PDF-test-original').decode()}
                response=await client.post('/v1/paperwork/documents',json=payload)
                self.assertEqual(response.status_code,200)
                identity=response.json()['id']
                self.assertEqual((await client.post('/v1/paperwork/documents',json=payload)).json()['id'],identity)
                self.assertEqual(len((await client.get('/v1/paperwork/documents')).json()),1)
                original=await client.get(f'/v1/paperwork/documents/{identity}/original')
                self.assertEqual(original.content,b'%PDF-test-original')
                proposal=await client.post(f'/v1/paperwork/documents/{identity}/propose',json={'pages':['// 2026-10-31\n• Memória']})
                self.assertEqual(proposal.json()['blocks'][0]['context'],'Daily/2026-10-31')
                result=await client.get('/v1/paperwork/search',params={'q':'Memória'})
                self.assertEqual(result.json()[0]['document'],identity)
                invalid=await client.post('/v1/paperwork/documents',json={'name':'../scan.pdf','data':payload['data']})
                self.assertEqual(invalid.status_code,422)
