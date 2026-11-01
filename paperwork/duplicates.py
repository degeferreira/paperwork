"""Exact normalized segment signatures suggest review, never merge content."""
import hashlib
import json
import re
import unicodedata
from collections import defaultdict

def signatures(proposal):
    groups=defaultdict(list)
    for block in proposal['blocks']:groups[block['context']].append(block)
    result=[]
    for context,blocks in groups.items():
        if not context:continue
        positions={block['id']:i for i,block in enumerate(blocks)}
        normalized=[{'kind':b['kind'],'text':re.sub(r'\s+',' ',unicodedata.normalize('NFKC',b['text'])).strip(),
                     'parent':positions.get(b['parent_id'])} for b in blocks]
        signature=hashlib.sha256(json.dumps([context,normalized],ensure_ascii=False,sort_keys=True).encode()).hexdigest()
        result.append((context,signature))
    return result

def candidates(db,proposal,identity):
    result=[]
    for context,signature in signatures(proposal):
        rows=db.execute('SELECT s.document_id,d.name FROM segment_signatures s JOIN documents d ON s.document_id=d.id WHERE s.signature=? AND s.document_id!=?',(signature,identity))
        result.extend({'context':context,'document_id':r[0],'name':r[1]} for r in rows)
    return result

def register(db,proposal,identity):
    with db:
        for context,signature in signatures(proposal):
            db.execute('INSERT OR IGNORE INTO segment_signatures VALUES (?,?,?)',(signature,identity,context))
