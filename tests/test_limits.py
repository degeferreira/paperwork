import unittest
import httpx
from fastapi import FastAPI,Request
from paperwork.http_limit import BodyLimit
class LimitsTests(unittest.IsolatedAsyncioTestCase):
    async def test_chunked_body_bounded_without_content_length(self):
        app=FastAPI();app.add_middleware(BodyLimit)
        @app.post('/test')
        async def endpoint(request:Request):return {'size':len(await request.body())}
        async def chunks():
            yield b'x'*40000
            yield b'x'*40000
        async with httpx.AsyncClient(transport=httpx.ASGITransport(app=app),base_url='http://local') as client:
            response=await client.post('/test',content=chunks())
            self.assertEqual(response.status_code,413)
