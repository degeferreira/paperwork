import base64,json,os,sqlite3,tempfile,unittest
from pathlib import Path
from unittest.mock import patch
import httpx
from fastapi import FastAPI
from paperwork.api import router
from paperwork.pfm import parse
from paperwork.markdown import render
from paperwork.transcription import transcribe

class TypedMentions(unittest.TestCase):
    def test_people_projects_inherit_separately_and_link_to_digital_pages(self):
        data=parse(['// 2026-11-01\n• Reunião @tese @@ana #memoria\n> • Notas\n• Email ana@example.org e \\@@literal'],'scan')
        root,child,other=data['blocks']
        self.assertEqual(root['people'],['ana']);self.assertEqual(root['projects'],['tese'])
        self.assertEqual(child['people'],['ana']);self.assertEqual(child['projects'],['tese'])
        self.assertEqual(other['people'],[]);self.assertEqual(other['projects'],[])
        text=render(data,'a'*64)['Daily/2026-11-01']
        self.assertIn('[[People/ana|@@ana]]',text);self.assertIn('[[Projects/tese|@tese]]',text)
        self.assertNotIn('Paperwork/Daily',text);self.assertNotIn('Projects/example',text)

class IndexAndSettings(unittest.IsolatedAsyncioTestCase):
    async def test_atomic_index_rejects_reassignment_and_settings_switch_keeps_models(self):
        with tempfile.TemporaryDirectory() as folder,patch.dict(os.environ,{'PAPERWORK_HOME':folder,'PAPERWORK_CONFIG':folder+'/config.json'},clear=False):
            app=FastAPI();app.include_router(router)
            async with httpx.AsyncClient(transport=httpx.ASGITransport(app=app),base_url='http://local') as c:
                first={'alias':'Silva24-Mem','library_type':'users','library_id':123,'item_key':'ABCD1234'}
                self.assertEqual((await c.post('/v1/paperwork/references',json=first)).status_code,200)
                second={**first,'alias':'Outro24-Obr','item_key':'EFGH5678'}
                bad={**first,'item_key':'EFGH5678'}
                result=await c.post('/v1/paperwork/references/index',json={'format':'paperwork-zotero-aliases-v1','references':[second,bad]})
                self.assertEqual(result.status_code,409)
                self.assertEqual(len((await c.get('/v1/paperwork/references')).json()),1)
                self.assertEqual((await c.post('/v1/paperwork/references/index',json={'format':'paperwork-zotero-aliases-v1','references':[first,second]})).status_code,200)
                for provider in ['gemini','claude','openai','openai-compatible','local-compatible']:
                    response=await c.put('/v1/paperwork/settings',json={'provider':provider,'model':provider+'-test'})
                    self.assertEqual(response.status_code,200,response.text)
                settings=(await c.get('/v1/paperwork/settings')).json()
                self.assertEqual(settings['profiles']['openai']['model'],'openai-test')
                self.assertNotIn('key_file',json.dumps(settings))

class NewProviders(unittest.IsolatedAsyncioTestCase):
    async def test_claude_and_openai_pdf_image_and_no_request_without_consent(self):
        with tempfile.TemporaryDirectory() as folder:
            key=Path(folder)/'key';key.write_text('synthetic-test-key')
            for provider in ['claude','openai']:
                calls=[]
                def handler(request):
                    calls.append(json.loads(request.content))
                    if provider=='claude':return httpx.Response(200,json={'stop_reason':'end_turn','content':[{'type':'text','text':'{"pages":["• Fiel @tese @@ana"]}'}]})
                    return httpx.Response(200,json={'status':'completed','output':[{'content':[{'type':'output_text','text':'{"pages":["• Fiel @tese @@ana"]}'}]}]})
                config={'provider':provider,'model':'test-model','key_file':str(key),'external_allowed':True}
                async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as client:
                    with self.assertRaises(ValueError):await transcribe(b'%PDF','application/pdf',config,client)
                    self.assertEqual(calls,[])
                    for mime in ['application/pdf','image/jpeg']:
                        result=await transcribe(b'test',mime,config,client,True)
                        self.assertEqual(result['pages'],['• Fiel @tese @@ana'])
                    if provider=='claude':self.assertEqual(calls[0]['messages'][0]['content'][1]['type'],'document')
                    else:
                        self.assertEqual(calls[0]['input'][0]['content'][1]['type'],'input_file')
                        self.assertFalse(calls[0]['store'])
    async def test_incomplete_results_do_not_become_transcriptions(self):
        with tempfile.TemporaryDirectory() as folder:
            key=Path(folder)/'key';key.write_text('synthetic')
            for provider,result in [('claude',{'stop_reason':'max_tokens'}),('openai',{'status':'incomplete'})]:
                async with httpx.AsyncClient(transport=httpx.MockTransport(lambda r:httpx.Response(200,json=result))) as c:
                    with self.assertRaises(ValueError):await transcribe(b'test','image/png',{'provider':provider,'model':'test','key_file':str(key),'external_allowed':True},c,True)
