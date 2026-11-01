"""Add reviewed blocks with conditional writes and retry-safe block markers."""
from .markdown import render

async def incorporate(proposal,identity,space,aliases=None,task_ids=None):
    notes=render(proposal,identity,aliases,task_ids)
    result=[]
    document_path='Paperwork/Documents/'+identity+'.md'
    document,_=await space.read(document_path)
    if document is None:
        link='/.proxy/127.0.0.1:8088/v1/paperwork/documents/'+identity+'/original'
        await space.write(document_path,'# Documento original\n\n<a href="'+link+'" target="_blank" rel="noopener">Abrir original preservado</a>\n',None)
    # One note at a time. On partial failure, rerunning skips completed blocks.
    for page,content in notes.items():
        path=page+'.md'
        previous,etag=await space.read(path)
        existing=previous or ''
        missing=[]
        for line in content.splitlines():
            import re
            marker=re.search(r'<!-- paperwork-block:[^>]+ -->',line)
            if marker and marker[0] in existing:continue
            missing.append(line)
        if not missing:
            result.append({'path':page,'changed':False});continue
        if previous is not None and not etag:
            raise ValueError('Server did not provide a revision; refusing unsafe overwrite')
        updated=existing.rstrip()+'\n\n## Caderno\n\n'+'\n'.join(missing)+'\n'
        await space.write(path,updated,etag)
        result.append({'path':page,'changed':True})
    return result
