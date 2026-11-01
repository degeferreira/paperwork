"""Authenticated-by-SilverBullet loopback API. Manual transcription works offline."""
import base64
import os
from pathlib import Path
import tempfile
from fastapi import APIRouter, HTTPException
from fastapi.responses import FileResponse
from pydantic import BaseModel, Field
from .store import Store

router=APIRouter(prefix='/v1/paperwork')

def store(): return Store(os.environ.get('PAPERWORK_HOME','/paperwork'))

class Import(BaseModel):
    name:str=Field(min_length=1,max_length=200)
    data:str=Field(max_length=24_000_000)

@router.post('/documents')
def import_document(body:Import):
    if Path(body.name).name!=body.name or Path(body.name).suffix.lower() not in ('.pdf','.png','.jpg','.jpeg','.webp'):
        raise HTTPException(422,'Use a PDF or image filename')
    try: data=base64.b64decode(body.data,validate=True)
    except ValueError: raise HTTPException(422,'Invalid base64') from None
    signature={'.pdf':data.startswith(b'%PDF-'),'.png':data.startswith(b'\x89PNG\r\n\x1a\n'),
               '.jpg':data.startswith(b'\xff\xd8\xff'),'.jpeg':data.startswith(b'\xff\xd8\xff'),
               '.webp':data.startswith(b'RIFF') and data[8:12]==b'WEBP'}
    if not signature.get(Path(body.name).suffix.lower()):raise HTTPException(422,'File extension does not match document signature')
    library=store()
    try:
        with tempfile.TemporaryDirectory() as folder:
            path=Path(folder)/body.name;path.write_bytes(data)
            identity=library.import_original(path)
        return {'id':identity,'name':body.name}
    finally:library.db.close()

@router.get('/documents')
def documents():
    library=store()
    try: return [{'id':row[0],'name':row[1]} for row in library.db.execute('SELECT id,name FROM documents ORDER BY rowid DESC')]
    finally:library.db.close()

@router.get('/documents/{identity}/original')
def original(identity:str):
    if len(identity)!=64 or any(c not in '0123456789abcdef' for c in identity):raise HTTPException(404)
    library=store()
    try: row=library.db.execute('SELECT name FROM documents WHERE id=?',(identity,)).fetchone()
    finally:library.db.close()
    if not row:raise HTTPException(404)
    suffix=Path(row[0]).suffix.lower()
    return FileResponse(Path(os.environ.get('PAPERWORK_HOME','/paperwork'))/'originals'/identity,
                        media_type={'.pdf':'application/pdf','.png':'image/png','.jpg':'image/jpeg','.jpeg':'image/jpeg','.webp':'image/webp'}[suffix],
                        filename=row[0],content_disposition_type='inline',headers={'X-Content-Type-Options':'nosniff'})

from datetime import date
class Transcription(BaseModel):
    default_date:date|None=None
    note_title:str|None=Field(default=None,max_length=200)
    pages:list[str]=Field(min_length=1,max_length=500)

@router.post('/documents/{identity}/propose')
def propose(identity:str,body:Transcription):
    if sum(map(len,body.pages))>2_000_000:raise HTTPException(413)
    library=store()
    try:
        if body.default_date and body.note_title:raise HTTPException(422,'Choose a date or a note title, not both')
        context='Daily/'+body.default_date.isoformat() if body.default_date else 'Page/'+body.note_title if body.note_title else None
        return library.propose(identity,body.pages,context)
    except ValueError:raise HTTPException(404,'Unknown document') from None
    finally:library.db.close()

@router.get('/search')
def search(q:str):
    if len(q)>200:raise HTTPException(422)
    library=store()
    try:
        # Quoted phrase prevents user input from becoming an FTS expression.
        rows=library.db.execute('SELECT document_id,snippet(search,1,?,?,?,24) FROM search WHERE search MATCH ? LIMIT 50',('[',']','…','"'+q.replace('"','""')+'"')).fetchall()
        return [{'document':r[0],'excerpt':r[1]} for r in rows]
    finally:library.db.close()

class RenderRequest(BaseModel):
    # Explicit review acknowledgement; proposal itself stays server-side.
    reviewed:bool=False
    run_id:str|None=None
    allow_duplicates:bool=False

@router.post('/documents/{identity}/markdown')
def markdown(identity:str,body:RenderRequest):
    import json
    from .markdown import render
    if not body.reviewed:raise HTTPException(409,'Review acknowledgement required')
    library=store()
    try:
        row=library.db.execute('SELECT payload FROM proposals WHERE document_id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        return {'notes':render(json.loads(row[0]),identity)}
    except ValueError as error:raise HTTPException(409,str(error)) from None
    finally:library.db.close()

from fastapi import Request
@router.post('/documents/{identity}/incorporate')
async def incorporate_document(identity:str,body:RenderRequest,request:Request):
    import json
    from .incorporate import incorporate
    if not body.reviewed:raise HTTPException(409,'Review acknowledgement required')
    service=getattr(request.app.state,'service',None)
    if service is None:
        import asyncio
        from types import SimpleNamespace
        from .space import SpaceConnection
        try:
            connection=SpaceConnection()
        except (OSError,ValueError,KeyError):
            raise HTTPException(503,'SilverBullet connection not configured') from None
        lock=getattr(request.app.state,'space_lock',None)
        if lock is None:
            lock=asyncio.Lock();request.app.state.space_lock=lock
        service=SimpleNamespace(lock=lock,read=connection.read,write=connection.write)
    library=store()
    try:
        row=library.db.execute('SELECT payload FROM proposals WHERE document_id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        async with service.lock:
            _check_review_revision(library,identity,body.run_id)
            if json.loads(row[0]).get('possible_duplicates') and not body.allow_duplicates:
                raise HTTPException(409,'Review possible duplicate segments before incorporation')
            import hashlib
            proposal_hash=hashlib.sha256(row[0].encode()).hexdigest()
            reserved=library.db.execute('SELECT proposal_hash,phase FROM incorporation WHERE document_id=?',(identity,)).fetchone()
            if reserved and reserved[0]!=proposal_hash:
                raise HTTPException(409,'Document already incorporated with another revision; reconcile existing blocks before changing the tree')
            if reserved and reserved[1]=='done':
                return {'notes':[], 'already_incorporated':True}
            from .references import setup
            setup(library.db)
            aliases=dict(library.db.execute('SELECT alias,note_path FROM reference_aliases WHERE retired=0 AND note_path IS NOT NULL'))
            with library.db:
                library.db.execute('INSERT OR IGNORE INTO incorporation VALUES (?,?,?)',(identity,proposal_hash,'pending'))
            from .tasks import ensure_local
            task_ids=ensure_local(json.loads(row[0]),library)
            result=await incorporate(json.loads(row[0]),identity,service,aliases,task_ids)
            with library.db:library.db.execute('UPDATE incorporation SET phase=? WHERE document_id=?',('done',identity))
            from .duplicates import register
            register(library.db,json.loads(row[0]),identity)
        return {'notes':result}
    except ValueError as error:raise HTTPException(409,str(error)) from None
    finally:library.db.close()

class OCRRequest(BaseModel):
    consent_external:bool=False

async def run_transcription(identity,job,consent,configuration):
    import json
    import mimetypes
    import httpx
    from .transcription import transcribe
    library=store()
    try:
        row=library.db.execute('SELECT name FROM documents WHERE id=?',(identity,)).fetchone()
        data=(library.originals/identity).read_bytes()
        mime=mimetypes.guess_type(row[0])[0] or 'application/octet-stream'
        async with httpx.AsyncClient(follow_redirects=False) as client:
            result=await transcribe(data,mime,configuration,client,consent)
        # Raw provider response is durable before proposing any interpretation.
        queued=library.db.execute('SELECT detail FROM processing WHERE id=?',(job,)).fetchone()
        result={**(json.loads(queued[0]) if queued else {}),**result}
        with library.db:
            library.db.execute('UPDATE processing SET state=?,detail=? WHERE id=?',('transcribed',json.dumps(result,ensure_ascii=False),job))
        proposed=library.propose(identity,result['pages'])
        result['run_id']=proposed['run_id']
        with library.db:library.db.execute('UPDATE processing SET state=?,detail=? WHERE id=?',('review',json.dumps(result,ensure_ascii=False),job))
    except Exception as error:
        # Never log upstream bodies, keys or document contents.
        prior=library.db.execute('SELECT detail FROM processing WHERE id=?',(job,)).fetchone()
        detail=json.loads(prior[0]) if prior else {}
        detail['error']=type(error).__name__
        if isinstance(error,httpx.HTTPStatusError):detail['status_code']=error.response.status_code
        with library.db:
            library.db.execute('UPDATE processing SET state=?,detail=? WHERE id=?',('error',json.dumps(detail,ensure_ascii=False),job))
    finally:library.db.close()

@router.post('/documents/{identity}/transcribe')
async def queue_transcription(identity:str,body:OCRRequest,request:Request):
    import asyncio
    import uuid
    library=store()
    try:
        if not library.db.execute('SELECT 1 FROM documents WHERE id=?',(identity,)).fetchone():raise HTTPException(404)
        from .settings import configuration
        from urllib.parse import urlsplit
        configured=configuration().get('transcription',{})
        if configured.get('provider') not in ('gemini','local-compatible'):
            raise HTTPException(503,'Configure a transcription provider first; manual transcription remains available')
        external=configured['provider']=='gemini' or urlsplit(configured.get('url','')).hostname not in ('localhost','127.0.0.1','::1')
        policy=library.db.execute('SELECT local_only FROM document_metadata WHERE document_id=?',(identity,)).fetchone()
        if external and (not policy or policy[0] or not body.consent_external or not configured.get('external_allowed')):
            raise HTTPException(409,'Document is local-only or external transmission was not authorized')
        # One job at a time, protecting this server and preventing repeated clicks.
        tasks=getattr(request.app.state,'paperwork_tasks',set())
        if any(not t.done() for t in tasks):raise HTTPException(409,'A transcription is already running')
        job=str(uuid.uuid4())
        from .transcription import PROMPT,PROMPT_VERSION
        metadata={'provider':configured['provider'],'model':configured.get('model'),'prompt':PROMPT,'prompt_version':PROMPT_VERSION,'input_sha256':identity,'consent_external':body.consent_external,'cost':None}
        with library.db:library.db.execute('INSERT INTO processing VALUES (?,?,?,?)',(job,identity,'queued',json.dumps(metadata,ensure_ascii=False)))
        task=asyncio.create_task(run_transcription(identity,job,body.consent_external,configured.copy()))
        tasks.add(task);task.add_done_callback(tasks.discard);request.app.state.paperwork_tasks=tasks
        return {'job':job,'state':'queued'}
    finally:library.db.close()

@router.get('/jobs')
def jobs():
    library=store()
    try:return [{'id':r[0],'document':r[1],'state':r[2],'error':r[3],'status_code':r[4]} for r in library.db.execute("SELECT id,document_id,state,json_extract(detail,'$.error'),json_extract(detail,'$.status_code') FROM processing ORDER BY rowid DESC LIMIT 100")]
    finally:library.db.close()

class ExportTasks(BaseModel):
    reviewed:bool=False
    run_id:str|None=None
    allow_duplicates:bool=False
    project_id:int=Field(gt=0)

@router.post('/documents/{identity}/tasks/export')
async def export_tasks(identity:str,body:ExportTasks,request:Request):
    import json
    from .tasks import export,apply_updates
    if not body.reviewed:raise HTTPException(409,'Review acknowledgement required')
    connector=getattr(request.app.state,'service',None)
    if connector is None:
        import asyncio
        from types import SimpleNamespace
        from .space import TaskConnection
        try:connection=TaskConnection()
        except (KeyError,ValueError,OSError):raise HTTPException(503,'Vikunja connection not configured') from None
        lock=getattr(request.app.state,'tasks_lock',None)
        if lock is None:lock=asyncio.Lock();request.app.state.tasks_lock=lock
        connector=SimpleNamespace(lock=lock,vk=connection.vk,paginated_vk=connection.paginated_vk,config={'vikunja':{'url':connection.url}})
    library=store()
    try:
        row=library.db.execute('SELECT payload FROM proposals WHERE document_id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        async with connector.lock:
            _check_review_revision(library,identity,body.run_id)
            if json.loads(row[0]).get('possible_duplicates') and not body.allow_duplicates:
                raise HTTPException(409,'Review possible duplicate segments before exporting tasks')
            results=await export(json.loads(row[0]),library,connector,connector.config['vikunja']['url'],body.project_id)
            updates=await apply_updates(json.loads(row[0]),library,connector,connector.config['vikunja']['url'])
        return {'tasks':results,'updates':updates}
    except ValueError as error:raise HTTPException(409,str(error)) from None
    finally:library.db.close()

class ReferenceAlias(BaseModel):
    alias:str=Field(min_length=1,max_length=100)
    library_type:str
    library_id:int=Field(gt=0)
    item_key:str=Field(min_length=8,max_length=8)
    note_path:str|None=None

@router.post('/references')
def reference_alias(body:ReferenceAlias):
    from .references import register
    library=store()
    try:return register(library.db,**body.model_dump())
    except ValueError as error:raise HTTPException(409,str(error)) from None
    finally:library.db.close()

@router.get('/documents/{identity}/references')
def document_references(identity:str):
    import json
    from .references import resolve
    library=store()
    try:
        row=library.db.execute('SELECT payload FROM proposals WHERE document_id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        return resolve(library.db,json.loads(row[0]))
    finally:library.db.close()

from .settings import router as settings_router
router.include_router(settings_router)

@router.post('/references/{alias}/import')
async def import_reference(alias:str,request:Request):
    # Reuse the existing personal Zotero importer when identities really match.
    library=store()
    try:
        from .references import setup
        setup(library.db)
        row=library.db.execute('SELECT library_type,library_id,item_key,note_path FROM reference_aliases WHERE alias=? AND retired=0',(alias,)).fetchone()
        if not row:raise HTTPException(404)
        service=getattr(request.app.state,'service',None)
        if service and row[0]=='users' and await service.zotero_user()==row[1]:
            result=await service.import_zotero(row[2])
            path=result['path'].removesuffix('.md')
        else:
            import asyncio
            import httpx
            from .settings import config_path,configuration
            from .zotero import fetch,note
            from .space import SpaceConnection
            config=configuration()
            # Preserve Relay's existing Literature Notes instead of creating a copy.
            if config.get('__shared') and row[0]=='users':
                async with httpx.AsyncClient(timeout=90,follow_redirects=False) as client:
                    response=await client.get(os.environ.get('PAPERWORK_RELAY_URL','http://127.0.0.1:8088')+'/v1/settings')
                    response.raise_for_status();known=response.json().get('zotero_user_id')
                    if known is None:
                        response=await client.get(os.environ.get('PAPERWORK_RELAY_URL','http://127.0.0.1:8088')+'/v1/zotero/identity')
                        response.raise_for_status();known=response.json()['user_id']
                    if known==row[1]:
                        response=await client.post(os.environ.get('PAPERWORK_RELAY_URL','http://127.0.0.1:8088')+'/v1/zotero/items/'+row[2]+'/import')
                        response.raise_for_status();path=response.json()['path'].removesuffix('.md')
                        if '${' in path or any(c in path for c in '[]`<>\n\r'):
                            raise HTTPException(409,'Reference note path needs manual review before linking')
                        with library.db:library.db.execute('UPDATE reference_aliases SET note_path=? WHERE alias=?',(path,alias))
                        return {'path':path,'alias':alias}
            try:
                key_file=config.get('zotero',{}).get('key_file',str(config_path().parent/'secrets/zotero_key'))
                credential=Path(key_file).read_text().strip()
                connection=service or SpaceConnection()
            except (OSError,ValueError,KeyError):raise HTTPException(503,'Zotero or SilverBullet credentials not configured') from None
            async with httpx.AsyncClient(timeout=30,follow_redirects=False) as client:
                item,children=await fetch(client,row[0],row[1],row[2],credential)
            path=row[3] or f'Literature/Zotero-{row[0]}-{row[1]}-{row[2]}'
            existing,etag=await connection.read(path+'.md')
            if existing is None:
                await connection.write(path+'.md',note(item,children,row[0],row[1]),None)
        if '${' in path or any(c in path for c in '[]`<>\n\r'):
            raise HTTPException(409,'Reference note path needs manual review before linking')
        with library.db:library.db.execute('UPDATE reference_aliases SET note_path=? WHERE alias=?',(path,alias))
        return {'path':path,'alias':alias}
    finally:library.db.close()

@router.get('/references')
def list_references():
    from .references import setup
    library=store()
    try:
        setup(library.db)
        return [{'alias':r[0],'library_type':r[1],'library_id':r[2],'item_key':r[3],'note_path':r[4]}
                for r in library.db.execute('SELECT alias,library_type,library_id,item_key,note_path FROM reference_aliases WHERE retired=0 ORDER BY alias')]
    finally:library.db.close()

@router.get('/documents/{identity}/proposal')
def proposal(identity:str):
    import json
    library=store()
    try:
        row=library.db.execute('SELECT payload FROM proposals WHERE document_id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        result=json.loads(row[0])
        latest=library.db.execute('SELECT pages,id FROM runs WHERE document_id=? ORDER BY rowid DESC LIMIT 1',(identity,)).fetchone()
        if latest:result.update(pages=json.loads(latest[0]),run_id=latest[1])
        return result
    finally:library.db.close()

@router.get('/tasks')
def local_tasks():
    library=store()
    try:return [{'block_id':r[0],'task_id':r[1],'instance':r[2],'remote_id':r[3],'phase':r[4],'title':r[5] or r[1],'done':bool(r[6])}
                for r in library.db.execute('SELECT t.block_id,t.task_id,t.instance,t.remote_id,t.phase,d.title,d.done FROM tasks t LEFT JOIN task_data d ON t.block_id=d.block_id ORDER BY t.rowid DESC LIMIT 500')]
    finally:library.db.close()

class DocumentPolicy(BaseModel):local_only:bool=True
@router.put('/documents/{identity}/policy')
def document_policy(identity:str,body:DocumentPolicy):
    library=store()
    try:
        if not library.db.execute('SELECT 1 FROM documents WHERE id=?',(identity,)).fetchone():raise HTTPException(404)
        if library.db.execute("SELECT 1 FROM processing WHERE document_id=? AND state IN ('queued','transcribed')",(identity,)).fetchone():
            raise HTTPException(409,'Wait for current processing before changing policy')
        with library.db:library.db.execute('UPDATE document_metadata SET local_only=? WHERE document_id=?',(int(body.local_only),identity))
        return {'local_only':body.local_only}
    finally:library.db.close()

class BackupRequest(BaseModel):
    name:str=Field(min_length=1,max_length=100,pattern=r'^[A-Za-z0-9][A-Za-z0-9_.-]*$')
@router.post('/backups')
async def create_backup(body:BackupRequest,request:Request):
    import asyncio
    from types import SimpleNamespace
    from .backup import backup_with_notes
    from .space import SpaceConnection
    service=getattr(request.app.state,'service',None)
    if service is None:
        try:connection=SpaceConnection()
        except (ValueError,KeyError,OSError):raise HTTPException(503,'Configure SilverBullet connection') from None
        lock=getattr(request.app.state,'space_lock',None)
        if lock is None:lock=asyncio.Lock();request.app.state.space_lock=lock
        service=SimpleNamespace(lock=lock,read=connection.read,files=connection.files)
    root=Path(os.environ.get('PAPERWORK_BACKUP_ROOT',str(Path(os.environ.get('PAPERWORK_HOME','/paperwork')).parent/'paperwork-backups')))
    destination=root/body.name
    if destination.exists():raise HTTPException(409,'Snapshot name already exists')
    library=store()
    try:
        async with service.lock:return await backup_with_notes(library,service,destination)
    except ValueError as error:raise HTTPException(409,str(error)) from None
    finally:library.db.close()

@router.post('/daily/open')
async def open_daily(request:Request):
    import datetime as dt
    from zoneinfo import ZoneInfo
    from .settings import configuration
    from .space import SpaceConnection
    service=getattr(request.app.state,'service',None)
    if service:return await service.open_daily()
    try:
        connection=SpaceConnection()
        day=dt.datetime.now(ZoneInfo(configuration().get('timezone','UTC'))).date().isoformat()
        path='Daily/'+day+'.md'
        existing,_=await connection.read(path)
        if existing is None:await connection.write(path,'---\ntype: daily\ndate: '+day+'\n---\n\n# '+day+'\n\n',None)
        return {'path':path.removesuffix('.md')}
    except (ValueError,KeyError,OSError):raise HTTPException(503,'Configure SilverBullet connection and timezone') from None

class TaskStatus(BaseModel):done:bool
@router.patch('/tasks/{identity}')
def update_local_task(identity:str,body:TaskStatus):
    library=store()
    try:
        row=library.db.execute('SELECT block_id FROM tasks WHERE task_id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        with library.db:
            library.db.execute('UPDATE task_data SET done=? WHERE block_id=?',(int(body.done),row[0]))
            library.db.execute('UPDATE tasks SET phase=? WHERE task_id=?',('pending-update',identity))
        return {'task_id':identity,'done':body.done,'remote_pending':True}
    finally:library.db.close()

def _check_review_revision(library,identity,run_id):
    latest=library.db.execute('SELECT id FROM runs WHERE document_id=? ORDER BY rowid DESC LIMIT 1',(identity,)).fetchone()
    if not run_id or not latest or latest[0]!=run_id:
        raise HTTPException(409,'Load and review the current proposal; its revision changed or was not supplied')

@router.get('/documents/{identity}')
def document(identity:str):
    library=store()
    try:
        row=library.db.execute('SELECT name FROM documents WHERE id=?',(identity,)).fetchone()
        if not row:raise HTTPException(404)
        return {'id':identity,'name':row[0]}
    finally:library.db.close()
