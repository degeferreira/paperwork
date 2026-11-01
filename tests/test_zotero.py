import unittest
import httpx
from paperwork.zotero import fetch,note
class ZoteroTests(unittest.IsolatedAsyncioTestCase):
    async def test_group_read_only_and_imported_text_not_executable(self):
        calls=[]
        def transport(request):
            calls.append(request)
            if request.url.path.endswith('/children'):return httpx.Response(200,json=[])
            return httpx.Response(200,json={'key':'ABCD1234','data':{'title':'Title ${evil()}','abstractNote':'Abstract'}})
        async with httpx.AsyncClient(transport=httpx.MockTransport(transport)) as client:
            item,children=await fetch(client,'groups',123,'ABCD1234','test-only')
        self.assertTrue(all(r.method=='GET' for r in calls))
        self.assertTrue(all('/groups/123/' in r.url.path for r in calls))
        rendered=note(item,children,'groups',123)
        self.assertNotIn('${',rendered)
        self.assertIn('group:123',rendered)
