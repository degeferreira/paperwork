"""SilverBullet filesystem gateway with conditional writes, usable without Relay."""
import json
import os
from pathlib import Path
from urllib.parse import quote,urlsplit
import httpx

class SpaceConnection:
    def __init__(self):
        from .settings import configuration
        config=configuration()
        self.url=config['silverbullet_url'].rstrip('/')
        parsed=urlsplit(self.url)
        if parsed.scheme not in ('http','https') or not parsed.hostname or parsed.username or parsed.password:
            raise ValueError('Invalid SilverBullet URL')
        self.token_file=Path(config['silverbullet_token_file'])
        if not self.token_file.read_text().strip():raise ValueError('SilverBullet token not configured')

    def headers(self):return {'Authorization':'Bearer '+self.token_file.read_text().strip(),'X-Sync-Mode':'true'}
    def path(self,path):
        if path.startswith('/') or any(p in ('.','..','') for p in path.split('/')) or '\\' in path:
            raise ValueError('Unsafe path')
        return self.url+'/.fs/'+quote(path,safe='/')

    async def files(self):
        async with httpx.AsyncClient(timeout=30,follow_redirects=False) as client:
            response=await client.get(self.url+'/.fs',headers=self.headers())
        response.raise_for_status();data=response.json()
        return list(data) if isinstance(data,dict) else [p if isinstance(p,str) else p['name'] for p in data]

    async def read(self,path):
        async with httpx.AsyncClient(timeout=30,follow_redirects=False) as client:
            response=await client.get(self.path(path),headers=self.headers())
        if response.status_code==404:return None,None
        response.raise_for_status()
        return response.text,response.headers.get('etag')

    async def write(self,path,text,etag=None):
        headers=self.headers();headers['Content-Type']='text/markdown; charset=utf-8'
        headers['If-Match' if etag else 'If-None-Match']=etag or '*'
        async with httpx.AsyncClient(timeout=30,follow_redirects=False) as client:
            response=await client.put(self.path(path),headers=headers,content=text.encode())
        response.raise_for_status()

class TaskConnection:
    def __init__(self):
        from .settings import configuration,url
        cfg=configuration()['vikunja']
        self.url=url(cfg['url']);self.token_file=Path(cfg['token_file'])
        if not self.token_file.read_text().strip():raise ValueError('Vikunja token not configured')
    async def vk(self,method,path,retries=False,**kwargs):
        async with httpx.AsyncClient(timeout=30,follow_redirects=False) as client:
            response=await client.request(method,self.url+path,headers={'Authorization':'Bearer '+self.token_file.read_text().strip()},**kwargs)
        response.raise_for_status();return response
    async def paginated_vk(self,path,params=None):
        result=[];page=1
        while True:
            response=await self.vk('GET',path,params={**(params or {}),'page':page,'per_page':100})
            data=response.json() or [];result.extend(data)
            pages=int(response.headers.get('x-pagination-total-pages',page))
            if page>=pages:return result
            page+=1
