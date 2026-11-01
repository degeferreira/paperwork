"""Read-only Zotero import for standalone Paperwork; identity is never guessed."""
import json
from html.parser import HTMLParser
from .markdown import literal

class PlainText(HTMLParser):
    def __init__(self):super().__init__();self.parts=[]
    def handle_data(self,data):self.parts.append(data)
def text(value):
    parser=PlainText();parser.feed(value or '');return literal(' '.join(parser.parts))

async def fetch(client,library_type,library_id,key,credential):
    base=f'https://api.zotero.org/{library_type}/{library_id}'
    headers={'Zotero-API-Version':'3','Zotero-API-Key':credential}
    response=await client.get(base+'/items/'+key,headers=headers)
    response.raise_for_status();item=response.json()
    if item.get('key')!=key:raise ValueError('Zotero identity mismatch')
    children=[];start=0
    while True:
        response=await client.get(base+'/items/'+key+'/children',headers=headers,params={'start':start,'limit':100})
        response.raise_for_status();batch=response.json();children.extend(batch);start+=len(batch)
        if not batch or start>=int(response.headers.get('Total-Results',start)):break
    for attachment in list(children):
        if attachment.get('data',{}).get('itemType')!='attachment':continue
        start=0
        while True:
            response=await client.get(base+'/items/'+attachment['key']+'/children',headers=headers,params={'start':start,'limit':100})
            response.raise_for_status();batch=response.json();children.extend(batch);start+=len(batch)
            if not batch or start>=int(response.headers.get('Total-Results',start)):break
    return item,children

def note(item,children,library_type,library_id):
    data=item['data'];key=item['key']
    lines=['---','type: literature','zoteroLibrary: '+json.dumps(('user' if library_type=='users' else 'group')+':'+str(library_id)),
           'zoteroKey: '+json.dumps(key),'---','','# '+literal(data.get('title','Sem título')),'',
           '[Abrir item Zotero](https://www.zotero.org/'+library_type+'/'+str(library_id)+'/items/'+key+')','',
           '## Minhas notas','','','## Metadados importados','',literal(data.get('abstractNote','')),'','## Notas Zotero','']
    for child in children:
        c=child.get('data',{})
        if c.get('itemType')=='note':lines.extend([text(c.get('note','')),''])
        if c.get('itemType')=='annotation':lines.extend(['p. '+literal(c.get('annotationPageLabel','')),
                                                       '> '+literal(c.get('annotationText','')),literal(c.get('annotationComment','')),''])
    return '\n'.join(lines)+'\n'
