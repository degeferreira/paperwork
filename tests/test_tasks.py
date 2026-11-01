import tempfile
import unittest
from pathlib import Path
import httpx
from paperwork.pfm import parse
from paperwork.store import Store
from paperwork.tasks import export
class Connector:
    def __init__(self):self.created=[]
    async def paginated_vk(self,*args):return []
    async def vk(self,method,path,**kwargs):
        if method=='PUT' and path.endswith('/tasks'):
            self.created.append(kwargs['json']);return httpx.Response(200,json={'id':len(self.created)})
        return httpx.Response(200,json={'related_tasks':{'subtask':[{'id':2}]}})
class TaskTests(unittest.IsolatedAsyncioTestCase):
    async def test_reviewed_tasks_only_and_retry_no_duplicates(self):
        with tempfile.TemporaryDirectory() as folder:
            library=Store(folder);connector=Connector()
            try:
                proposal=parse(['// 2026-10-31\n□ Aula\n> ? Pergunta\n> □ Slides'],'scan')
                result=await export(proposal,library,connector,'instance',1)
                self.assertEqual(len(result),2)
                await export(proposal,library,connector,'instance',1)
                self.assertEqual(len(connector.created),2)
                with self.assertRaises(ValueError):await export(proposal,library,connector,'other-instance',1)
            finally:library.db.close()

    async def test_printed_ids_are_short_and_rescan_never_creates_again(self):
        with tempfile.TemporaryDirectory() as folder:
            library=Store(folder);connector=Connector()
            try:
                original=parse(['// 2026-10-31\n□ Preparar aula'],'scan1')
                result=await export(original,library,connector,'instance',1)
                self.assertEqual(result[0]['task_id'],'T00001')
                rescanned=parse(['// 2026-11-01\n□ T00001 Preparar aula'],'scan2')
                await export(rescanned,library,connector,'instance',1)
                self.assertEqual(len(connector.created),1)
                unknown=parse(['// 2026-11-01\n□ T99999 Desconhecida'],'scan3')
                with self.assertRaises(ValueError):await export(unknown,library,connector,'instance',1)
            finally:library.db.close()

    async def test_tasks_and_paper_completion_work_without_vikunja(self):
        from paperwork.tasks import ensure_local
        with tempfile.TemporaryDirectory() as folder:
            library=Store(folder)
            try:
                first=parse(['// 2026-10-31\n□ Preparar aula'],'paper1')
                ids=ensure_local(first,library)
                self.assertEqual(list(ids.values()),['T00001'])
                completed=parse(['// 2026-11-01\n× T00001'],'paper2')
                ensure_local(completed,library)
                self.assertEqual(library.db.execute('SELECT done FROM task_data').fetchone()[0],1)
                self.assertEqual(library.db.execute('SELECT remote_id FROM tasks').fetchone()[0],None)
            finally:library.db.close()
