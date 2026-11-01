import json
import os
from pathlib import Path
from urllib.parse import urlsplit
from fastapi import APIRouter,HTTPException,Request
from pydantic import BaseModel,Field,ConfigDict
router=APIRouter(prefix='/settings')

def config_path():return Path(os.environ.get('PAPERWORK_CONFIG','/paperwork/config.json'))
def configuration():
    try:config=json.loads(config_path().read_text())
    except FileNotFoundError:config={}
    config.pop('__shared',None)
    shared=os.environ.get('PAPERWORK_RELAY_CONFIG')
    if shared:
        settings=Path(os.environ.get('PAPERWORK_RELAY_SETTINGS','/relay-settings'))
        effective=settings/'effective.json'
        source=json.loads(effective.read_text()).get('connections',{}) if effective.exists() else json.loads(Path(shared).read_text())
        if not source:source=json.loads(Path(shared).read_text())
        def credential(name):
            active=settings/'secrets'/name
            return str(active if active.exists() else Path('/run/secrets')/name)
        config.update(silverbullet_url=source['silverbullet_url'],silverbullet_token_file=credential('silverbullet_token'),
                      timezone=source['timezone'],vikunja={**source['vikunja'],'token_file':credential('vikunja_token')},
                      zotero={**source.get('zotero',{}),'key_file':credential('zotero_key')},__shared=True)
    return config

def private_write(path,value):
    path.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
    temp=path.with_name(path.name+'.tmp')
    fd=os.open(temp,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    try:
        with os.fdopen(fd,'w') as output:output.write(value);output.flush();os.fsync(output.fileno())
        os.replace(temp,path)
    finally:temp.unlink(missing_ok=True)

def url(value):
    parsed=urlsplit(value)
    if parsed.scheme not in ('http','https') or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError('Invalid server URL')
    return value.rstrip('/')

class Settings(BaseModel):
    model_config=ConfigDict(extra='forbid')
    timezone:str='UTC'
    silverbullet_url:str='http://127.0.0.1:3000'
    vikunja_url:str='http://127.0.0.1:3456/api/v1'
    provider:str='manual'
    model:str=Field(default='',max_length=150)
    local_url:str='http://127.0.0.1:11434/v1'
    external_allowed:bool=False

@router.get('')
def get_settings(request:Request):
    config=configuration();ocr=config.get('transcription',{})
    service=getattr(request.app.state,'service',None)
    if service:
        config={**config,'silverbullet_url':service.config['silverbullet_url'],'vikunja':service.config['vikunja']}
    result=Settings(timezone=service.config['timezone'] if service else config.get('timezone','UTC'),silverbullet_url=config.get('silverbullet_url','http://127.0.0.1:3000'),
                    vikunja_url=config.get('vikunja',{}).get('url','http://127.0.0.1:3456/api/v1'),
                    provider=ocr.get('provider','manual'),model=ocr.get('model',''),local_url=ocr.get('url','http://127.0.0.1:11434/v1'),
                    external_allowed=ocr.get('external_allowed',False)).model_dump()
    return {**result,'shared_connections':bool(service) or config.get('__shared',False)}

@router.put('')
def put_settings(body:Settings,request:Request):
    if body.provider not in ('manual','gemini','local-compatible'):raise HTTPException(422,'Unsupported provider')
    try:
        sb,vk,local=map(url,(body.silverbullet_url,body.vikunja_url,body.local_url))
    except ValueError:raise HTTPException(422,'Invalid server URL') from None
    service=getattr(request.app.state,'service',None)
    shared=configuration()
    if shared.get('__shared') and (sb!=shared['silverbullet_url'].rstrip('/') or vk!=shared['vikunja']['url'].rstrip('/') or body.timezone!=shared['timezone']):
        raise HTTPException(409,'Configure shared server connections in Relay settings')
    if service and (sb!=service.config['silverbullet_url'].rstrip('/') or vk!=service.config['vikunja']['url'].rstrip('/') or body.timezone!=service.config['timezone']):
        raise HTTPException(409,'Configure shared server connections in Relay settings')
    from zoneinfo import ZoneInfo
    try:ZoneInfo(body.timezone)
    except (ValueError,KeyError):raise HTTPException(422,'Invalid timezone') from None
    config=configuration();config['timezone']=body.timezone;config['silverbullet_url']=sb
    secret_dir=config_path().parent/'secrets'
    config.setdefault('silverbullet_token_file',str(secret_dir/'silverbullet_token'))
    config['vikunja']={'url':vk,'token_file':str(secret_dir/'vikunja_token')}
    config['transcription']={'provider':body.provider,'model':body.model,'url':local,'external_allowed':body.external_allowed,
                             'key_file':str(secret_dir/('gemini_key' if body.provider=='gemini' else 'local_key'))}
    private_write(config_path(),json.dumps(config,indent=2)+'\n')
    return {'saved':True}

class Credential(BaseModel):value:str=Field(min_length=1,max_length=4096)
@router.put('/secrets/{name}')
def credential(name:str,body:Credential,request:Request):
    if name not in ('silverbullet_token','vikunja_token','zotero_key','gemini_key','local_key'):raise HTTPException(404)
    if (getattr(request.app.state,'service',None) or configuration().get('__shared')) and name in ('silverbullet_token','vikunja_token','zotero_key'):
        raise HTTPException(409,'Configure shared credentials in Relay settings')
    value=body.value.strip()
    if not value or '\n' in value or '\r' in value:raise HTTPException(422)
    path=config_path().parent/'secrets'/name
    private_write(path,value+'\n')
    if name=='silverbullet_token':
        config=configuration();config['silverbullet_token_file']=str(path)
        private_write(config_path(),json.dumps(config,indent=2)+'\n')
    return {'saved':True}
