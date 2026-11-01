"""Standalone Paperwork API; no Relay installation or worker required."""
from fastapi import FastAPI,Request
from fastapi.responses import JSONResponse
from .api import router
from contextlib import asynccontextmanager
@asynccontextmanager
async def lifespan(app):
    import asyncio
    from .store import recover_processing
    recover_processing()
    yield
    tasks=list(getattr(app.state,'paperwork_tasks',set()))
    for task in tasks:task.cancel()
    if tasks:await asyncio.gather(*tasks,return_exceptions=True)
app=FastAPI(title='Paperwork',docs_url=None,redoc_url=None,openapi_url=None,lifespan=lifespan)
from .http_limit import BodyLimit
app.add_middleware(BodyLimit)
@app.middleware('http')
async def local_only(request:Request,call_next):
    if request.client and request.client.host not in ('127.0.0.1','::1'):
        return JSONResponse({'detail':'Loopback only'},status_code=403)
    try: size=int(request.headers.get('content-length','0'))
    except ValueError:return JSONResponse({'detail':'Invalid Content-Length'},status_code=400)
    if size<0 or size>24_100_000:return JSONResponse({'detail':'Request too large'},status_code=413)
    return await call_next(request)
app.include_router(router)
@app.get('/health/live')
def live():return {'ok':True}

import httpx
@app.exception_handler(httpx.HTTPError)
async def upstream_error(request:Request,error):
    code=error.response.status_code if isinstance(error,httpx.HTTPStatusError) else None
    if code==412:return JSONResponse({'detail':'A note changed during incorporation. Wait for sync and retry the same proposal.'},status_code=409)
    return JSONResponse({'detail':'Upstream connection unavailable','upstream_status':code},status_code=502)
